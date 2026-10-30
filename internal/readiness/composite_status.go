package readiness

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type CompositeStatusOptions struct {
	StateDir   string
	Entry      registry.SkillEntry
	Version    string
	PackageDir string
	TargetRoot string
	Runtime    string
	Now        time.Time
}

type CompositeProviderStatus struct {
	ToolID             string     `json:"tool_id"`
	ToolVersion        string     `json:"tool_version"`
	Requirement        string     `json:"requirement"`
	Access             string     `json:"access"`
	State              string     `json:"state"`
	Configured         bool       `json:"configured"`
	Authenticated      bool       `json:"authenticated"`
	Methods            []string   `json:"methods"`
	CredentialBindings []string   `json:"credential_bindings"`
	Scopes             []string   `json:"scopes"`
	SetupURL           string     `json:"setup_url,omitempty"`
	ConfigureCommands  [][]string `json:"configure_commands,omitempty"`
	Authentication     AuthReport `json:"authentication"`
	Reason             string     `json:"reason,omitempty"`
}

type CompositeStatusReport struct {
	EntryID       string                    `json:"entry_id"`
	EntryVersion  string                    `json:"entry_version"`
	Runtime       string                    `json:"runtime"`
	CoverageState string                    `json:"coverage_state"`
	Ready         bool                      `json:"ready"`
	SetupRequired bool                      `json:"setup_required"`
	Providers     []CompositeProviderStatus `json:"providers"`
}

// CompositeStatus verifies the installed plugin closure, resolves each
// provider contract from that exact installed tool version, and then delegates
// identity and scope verification to the provider tool's existing auth driver.
// The plugin derives coverage only; it never reads or stores provider secrets.
func CompositeStatus(ctx context.Context, opts CompositeStatusOptions) (CompositeStatusReport, error) {
	report := CompositeStatusReport{
		EntryID: opts.Entry.ID, EntryVersion: opts.Version, Runtime: opts.Runtime,
		CoverageState: "complete", Ready: true, Providers: []CompositeProviderStatus{},
	}
	if len(opts.Entry.ProviderDependencies) == 0 {
		return report, errors.New("plugin has no provider dependency contract")
	}
	if strings.TrimSpace(opts.PackageDir) == "" || strings.TrimSpace(opts.TargetRoot) == "" {
		return report, errors.New("installed plugin target is required for composite authentication status")
	}
	contractDigest, err := installer.RuntimeContractSHA256(opts.Entry, opts.Version)
	if err != nil {
		return report, err
	}
	receipt, err := verifyInstalledClosure(InspectOptions{
		Module: "plugins", Entry: opts.Entry, Version: opts.Version, PackageDir: opts.PackageDir,
		TargetRoot: opts.TargetRoot, Runtime: opts.Runtime,
	}, contractDigest)
	if err != nil {
		return report, errors.New("installed plugin closure is invalid")
	}
	toolTarget, err := installer.ResolvePluginDependencyTarget(opts.Runtime, "tools", opts.TargetRoot)
	if err != nil {
		return report, err
	}
	versions := make(map[string]string)
	for _, member := range receipt.Closure {
		if member.Module != "tools" {
			continue
		}
		if _, duplicate := versions[member.ID]; duplicate {
			return report, errors.New("installed plugin closure repeats a provider tool")
		}
		versions[member.ID] = member.Version
	}

	dependencies := append([]registry.ProviderDependencyMetadata(nil), opts.Entry.ProviderDependencies...)
	sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].Tool < dependencies[j].Tool })
	requiredUnavailable := false
	optionalUnavailable := false
	for _, dependency := range dependencies {
		provider := CompositeProviderStatus{
			ToolID: dependency.Tool, ToolVersion: versions[dependency.Tool], Requirement: dependency.Requirement,
			Access: dependency.Access, State: "unavailable", Authentication: AuthReport{},
		}
		if provider.ToolVersion == "" {
			provider.Reason = "provider tool is absent from the installed plugin closure"
		} else {
			toolDir := filepath.Join(toolTarget.TargetPath, filepath.FromSlash(dependency.Tool))
			manifestPath := filepath.Join(toolDir, "tool.yaml")
			manifest, manifestErr := registry.ValidateHistoricalPackageManifest(manifestPath)
			if manifestErr == nil {
				manifest, manifestErr = registry.ValidatePackageManifest(manifestPath)
			}
			if manifestErr != nil || manifest.ID != dependency.Tool || manifest.Version != provider.ToolVersion {
				provider.Reason = "installed provider contract is unavailable, expired, or does not match the plugin lock"
			} else {
				entry := registry.ProjectManifest(manifest)
				if entry.Authentication == nil || entry.Operational == nil || entry.Operational.AccessLevel != dependency.Access {
					provider.Reason = "installed provider authority does not match the plugin contract"
				} else {
					provider.Methods = append([]string(nil), entry.Authentication.Methods...)
					provider.CredentialBindings = append([]string(nil), entry.Authentication.CredentialBindings...)
					provider.Scopes = append([]string(nil), entry.Authentication.Scopes...)
					provider.SetupURL = entry.Authentication.SetupURL
					provider.ConfigureCommands = compositeConfigureCommands(
						dependency.Tool, provider.ToolVersion, opts.Runtime, toolTarget.TargetPath, opts.StateDir, *entry.Authentication,
					)
					auth, statusErr := Status(ctx, InspectOptions{
						StateDir: opts.StateDir, Module: "tools", Entry: entry, Version: provider.ToolVersion,
						PackageDir: toolDir, TargetRoot: toolTarget.TargetPath, Runtime: opts.Runtime, Now: opts.Now,
					})
					provider.Authentication = auth
					provider.Configured = auth.Configured
					provider.Authenticated = statusErr == nil && auth.Authenticated && auth.EntryID == dependency.Tool && auth.EntryVersion == provider.ToolVersion
					if provider.Authenticated {
						provider.State = "ready"
						provider.Reason = ""
					} else if statusErr != nil {
						provider.Reason = "provider authentication status is unavailable"
					} else {
						provider.Reason = auth.Reason
						if provider.Reason == "" {
							provider.Reason = "provider authentication is not ready"
						}
					}
				}
			}
		}
		if !provider.Authenticated {
			report.SetupRequired = true
			if dependency.Requirement == "required" {
				requiredUnavailable = true
			} else {
				optionalUnavailable = true
			}
		}
		report.Providers = append(report.Providers, provider)
	}
	if requiredUnavailable {
		report.CoverageState = "unavailable"
		report.Ready = false
	} else if optionalUnavailable {
		report.CoverageState = "partial"
	}
	return report, nil
}

func compositeConfigureCommands(toolID, version, runtimeName, targetRoot, stateDir string, auth registry.AuthenticationMetadata) [][]string {
	commands := make([][]string, 0, len(auth.Methods))
	for _, method := range auth.Methods {
		command := []string{
			"skills-hub", "auth", "configure", toolID + "@" + version,
			"--module", "tools", "--runtime", runtimeName, "--target", targetRoot,
			"--state-dir", stateDir, "--method", method,
		}
		for _, binding := range auth.CredentialBindings {
			command = append(command, "--binding", binding+"=env:"+binding, "--generation", binding+"=env:"+binding+"_GENERATION")
		}
		commands = append(commands, command)
	}
	return commands
}
