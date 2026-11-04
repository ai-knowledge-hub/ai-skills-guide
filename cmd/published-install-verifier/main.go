// published-install-verifier is a read-only companion to the public-delivery
// checker. It never installs, authenticates, executes package code or writes
// readiness evidence.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("published-install-verifier", flag.ContinueOnError)
	path := flags.String("manifest", "", "digest-bound pinned package manifest")
	digest := flags.String("manifest-sha256", "", "expected manifest SHA-256")
	id := flags.String("id", "", "expected pinned package ID")
	version := flags.String("version", "", "expected pinned version")
	packageDir := flags.String("package-dir", "", "optional installed package to verify")
	runtime := flags.String("runtime", "", "selected adapter")
	artifact := flags.String("artifact-sha256", "", "expected published archive SHA-256")
	module := flags.String("module", "skills", "bounded sample module: skills, tools or plugins")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *path == "" || *id == "" || *version == "" || !validDigest(*digest) {
		return errors.New("manifest, manifest-sha256, id and version are required")
	}
	payload, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), *digest) {
		return errors.New("pinned manifest digest mismatch")
	}
	manifest, err := registry.ValidateHistoricalPackageManifest(*path)
	if err != nil {
		return err
	}
	if manifest.ID != *id || manifest.Version != *version {
		return errors.New("manifest does not match the pinned subject")
	}
	switch *module {
	case "skills":
		if filepath.Base(*path) != "skill.yaml" || manifest.Usability.Execution != "instructions" || manifest.Usability.Availability != "documentation-only" || len(manifest.Usability.ExecutableHelpers) != 0 {
			return errors.New("pinned sample must be instruction-only without executable helpers")
		}
	case "tools":
		contracts := map[string]struct {
			script   string
			bindings []string
			scopes   []string
		}{
			"analytics/ga4-mcp-connector":         {"scripts/ga4_mcp_server.mjs", []string{"GA4_CREDENTIAL"}, []string{"https://www.googleapis.com/auth/analytics.readonly"}},
			"warehouse/bigquery-mcp-query-runner": {"scripts/bigquery_mcp_server.mjs", []string{"BQ_CREDENTIAL", "BQ_POLICY"}, []string{"https://www.googleapis.com/auth/bigquery"}},
			"ads/meta-ads-mcp-connector":          {"scripts/meta_ads_mcp_server.mjs", []string{"META_ADS_CREDENTIAL", "META_ADS_POLICY"}, []string{"ads_read", "business_management"}},
		}
		pinned, supported := contracts[manifest.ID]
		if !supported || filepath.Base(*path) != "tool.yaml" || manifest.Version != "0.2.0" ||
			manifest.Usability.Execution != "remote-integration" || manifest.Usability.Availability != "setup-required" ||
			manifest.Execution.Kind != "mcp-server" || !manifest.Artifact.SelfContained ||
			!reflect.DeepEqual(manifest.Execution.Command, []string{"node", pinned.script}) ||
			!reflect.DeepEqual(manifest.Execution.Healthcheck, []string{"node", pinned.script, "--healthcheck"}) ||
			!reflect.DeepEqual(manifest.Execution.SupportedRuntimes, []string{"node22"}) ||
			!reflect.DeepEqual(manifest.Execution.SupportedPlatforms, []string{"linux", "macos"}) ||
			manifest.Authentication.Status != "required" ||
			!reflect.DeepEqual(manifest.Authentication.CredentialBindings, pinned.bindings) ||
			!reflect.DeepEqual(manifest.Authentication.Scopes, pinned.scopes) {
			return errors.New("tool sample must match an explicitly pinned 0.2.0 setup-required MCP contract")
		}
	case "plugins":
		skills := append([]string(nil), manifest.Includes.Skills...)
		sort.Strings(skills)
		if filepath.Base(*path) != "plugin.yaml" || manifest.ID != "engineering/code-maintenance-plugin" || manifest.Version != "0.2.0" ||
			manifest.Usability.Execution != "bundle" || manifest.Execution.Kind != "bundle" || !manifest.Artifact.SelfContained ||
			!reflect.DeepEqual(skills, codeMaintenanceSkills) || len(manifest.Includes.Agents)+len(manifest.Includes.Tools) != 0 ||
			!reflect.DeepEqual(manifest.Includes.Hooks, []string{"verification-before-complete"}) ||
			!reflect.DeepEqual(manifest.Requires.Approvals, []string{"human-review-before-merge"}) ||
			len(manifest.Dependencies.Skills)+len(manifest.Dependencies.Agents)+len(manifest.Dependencies.Tools) != 0 ||
			!reflect.DeepEqual(manifest.Execution.SupportedRuntimes, []string{"node22"}) ||
			manifest.Authentication.Status != "none" || len(manifest.Authentication.CredentialBindings)+len(manifest.Authentication.Scopes) != 0 {
			return errors.New("plugin sample must match the pinned code-maintenance 0.2.0 bundle")
		}
	default:
		return errors.New("unsupported sample module")
	}
	contract, err := installer.RuntimeContractSHA256(registry.ProjectManifest(manifest), manifest.Version)
	if err != nil {
		return err
	}
	result := struct {
		ID                    string   `json:"id"`
		Version               string   `json:"version"`
		Runtimes              []string `json:"runtimes"`
		RuntimeContractSHA256 string   `json:"runtime_contract_sha256"`
		ReceiptVerified       bool     `json:"receipt_verified,omitempty"`
		ClosureVerified       bool     `json:"closure_verified,omitempty"`
		Name                  string   `json:"name"`
		Description           string   `json:"description"`
	}{ID: manifest.ID, Version: manifest.Version, Runtimes: manifest.Runtimes, RuntimeContractSHA256: contract, Name: manifest.Name, Description: manifest.Description}
	if *packageDir != "" {
		if !validDigest(*artifact) || !declaresRuntime(manifest.Runtimes, *runtime) {
			return errors.New("installed verification requires an artifact digest and declared runtime")
		}
		receipt, err := installer.VerifyInstallReceipt(*packageDir, *module, *id, *version, *runtime, *artifact, contract)
		if err != nil {
			return err
		}
		if receipt.Source != "remote" {
			return errors.New("installed receipt source must be remote")
		}
		if *module == "plugins" {
			if err := installer.VerifyInstallClosure(*packageDir, filepath.Dir(filepath.Dir(*packageDir)), receipt, &manifest.Includes); err != nil {
				return err
			}
			for _, id := range codeMaintenanceSkills {
				component, err := registry.ValidateHistoricalPackageManifest(filepath.Join(filepath.Dir(*path), "bundled", "skills", filepath.FromSlash(id), "skill.yaml"))
				if err != nil {
					return err
				}
				if component.ID != id || component.Version != "0.1.0" {
					return errors.New("bundled skill identity differs from pinned closure")
				}
				digest, err := installer.RuntimeContractSHA256(registry.ProjectManifest(component), component.Version)
				if err != nil {
					return err
				}
				runtimeRoot := filepath.Dir(filepath.Dir(filepath.Dir(*packageDir)))
				for _, directory := range []string{filepath.Join(*packageDir, "bundled", "skills", filepath.FromSlash(id)), filepath.Join(runtimeRoot, "skills", filepath.FromSlash(id))} {
					bundled, err := installer.VerifyInstallReceipt(directory, "skills", id, "0.1.0", *runtime, "", digest)
					if err != nil {
						return err
					}
					if bundled.Source != "bundled" || bundled.ParentArtifactSHA256 != *artifact || len(bundled.Closure) != 0 || bundled.ArtifactSHA256 != "" {
						return errors.New("bundled receipt is not bound exclusively to selected plugin")
					}
					for _, member := range receipt.Closure {
						if member.ID == id && (member.Version != "0.1.0" || member.RuntimeContractSHA256 != digest || member.TreeSHA256 != bundled.TreeSHA256) {
							return errors.New("plugin closure member differs from pinned skill contract")
						}
					}
				}
			}
			result.ClosureVerified = true
		}
		result.ReceiptVerified = true
	}
	return json.NewEncoder(output).Encode(result)
}

var codeMaintenanceSkills = []string{
	"engineering/code-change-verification", "engineering/coverage-gap-reporter", "engineering/implementation-strategy",
	"engineering/pr-review-and-draft", "engineering/test-gap-analyzer",
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func declaresRuntime(values []string, selected string) bool {
	for _, value := range values {
		if value == selected {
			return true
		}
	}
	return false
}
