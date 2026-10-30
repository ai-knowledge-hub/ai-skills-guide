package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

const agentRuntimeSchema = "skills-hub.agent-runtime/v1"

type compiledAgentRuntime struct {
	SchemaVersion   string                                   `json:"schema_version"`
	Runtime         string                                   `json:"runtime"`
	Agent           compiledRuntimePackageRoot               `json:"agent"`
	Status          string                                   `json:"status"`
	BlockingReasons []string                                 `json:"blocking_reasons"`
	Model           registry.OrchestrationModelRequirement   `json:"model"`
	Memory          registry.OrchestrationProfileRequirement `json:"memory"`
	Governance      registry.OrchestrationProfileRequirement `json:"governance"`
	Bindings        []compiledAgentBinding                   `json:"bindings"`
}

type compiledAgentBinding struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Package       string `json:"package,omitempty"`
	Version       string `json:"version,omitempty"`
	Requirement   string `json:"requirement"`
	Access        string `json:"access,omitempty"`
	Status        string `json:"status"`
	Path          string `json:"path,omitempty"`
	Endpoint      string `json:"endpoint,omitempty"`
	Reason        string `json:"reason,omitempty"`
	ConfigureHint string `json:"configure_hint,omitempty"`
}

// InstallAgentDependencies installs the pinned, direct orchestration closure.
// The manifest contract remains authoritative; registry latest pointers may not
// silently move an agent to a different dependency version.
func InstallAgentDependencies(entry registry.SkillEntry, runtimeName, agentTargetRoot string, moduleRoots, registryPaths map[string]string, force bool) (DependencyInstallResult, error) {
	result := DependencyInstallResult{}
	bindings, err := PreflightAgentDependencies(entry, runtimeName, agentTargetRoot, moduleRoots, registryPaths)
	if err != nil {
		return result, err
	}
	sets := []struct {
		kind, module string
		ids          []string
	}{
		{kind: "skill", module: "skills", ids: dependencyIDs(bindings, "skill")},
		{kind: "agent", module: "agents", ids: dependencyIDs(bindings, "agent")},
		{kind: "tool", module: "tools", ids: dependencyIDs(bindings, "tool")},
	}
	for _, set := range sets {
		if len(set.ids) == 0 {
			continue
		}
		installed, skipped, members, err := installDependencySet(set.ids, runtimeName, agentTargetRoot, set.module, moduleRoots[set.module], registryPaths[set.module], force)
		if err != nil {
			return result, err
		}
		switch set.kind {
		case "skill":
			result.InstalledSkills, result.SkippedSkills = installed, skipped
		case "agent":
			result.InstalledAgents, result.SkippedAgents = installed, skipped
		case "tool":
			result.InstalledTools, result.SkippedTools = installed, skipped
		}
		result.Closure = append(result.Closure, members...)
	}
	return result, nil
}

func PreflightAgentDependencies(entry registry.SkillEntry, runtimeName, agentTargetRoot string, moduleRoots, registryPaths map[string]string) ([]registry.OrchestrationBinding, error) {
	if entry.Orchestration == nil {
		return nil, fmt.Errorf("agent %s has no orchestration binding contract", entry.ID)
	}
	bindings, err := collectLocalAgentBindings(entry.Orchestration.Bindings, moduleRoots["agents"])
	if err != nil {
		return nil, err
	}
	sets := []struct {
		kind, module string
		ids          []string
	}{
		{kind: "skill", module: "skills", ids: dependencyIDs(bindings, "skill")},
		{kind: "agent", module: "agents", ids: dependencyIDs(bindings, "agent")},
		{kind: "tool", module: "tools", ids: dependencyIDs(bindings, "tool")},
	}
	// Resolve and admit the complete closure before the first filesystem write.
	for _, set := range sets {
		if len(set.ids) == 0 {
			continue
		}
		if err := preflightDependencySet(set.ids, runtimeName, agentTargetRoot, set.module, moduleRoots[set.module], registryPaths[set.module]); err != nil {
			return nil, err
		}
		if err := verifyPinnedRegistryVersions(bindings, set.kind, registryPaths[set.module]); err != nil {
			return nil, err
		}
	}
	return bindings, nil
}

func collectLocalAgentBindings(root []registry.OrchestrationBinding, agentsRoot string) ([]registry.OrchestrationBinding, error) {
	bindings := append([]registry.OrchestrationBinding(nil), root...)
	seenAgents := map[string]bool{}
	for index := 0; index < len(bindings); index++ {
		binding := bindings[index]
		if binding.Kind != "agent" || binding.Availability != "resolved" || seenAgents[binding.Package] {
			continue
		}
		seenAgents[binding.Package] = true
		manifest, err := registry.ValidateHistoricalPackageManifest(filepath.Join(agentsRoot, filepath.FromSlash(binding.Package), "agent.yaml"))
		if err != nil {
			return nil, fmt.Errorf("validate transitive agent %s: %w", binding.Package, err)
		}
		if manifest.Version != binding.Version {
			return nil, fmt.Errorf("transitive agent %s is %s, want pinned %s", binding.Package, manifest.Version, binding.Version)
		}
		if manifest.Orchestration == nil {
			return nil, fmt.Errorf("transitive agent %s has no orchestration contract", binding.Package)
		}
		bindings = append(bindings, manifest.Orchestration.Bindings...)
	}
	return bindings, nil
}

func dependencyIDs(bindings []registry.OrchestrationBinding, kind string) []string {
	seen := map[string]bool{}
	var ids []string
	for _, binding := range bindings {
		if binding.Kind == kind && binding.Availability == "resolved" && !seen[binding.Package] {
			seen[binding.Package] = true
			ids = append(ids, binding.Package)
		}
	}
	sort.Strings(ids)
	return ids
}

func verifyPinnedRegistryVersions(bindings []registry.OrchestrationBinding, kind, registryPath string) error {
	index, err := registry.LoadIndex(registryPath)
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.Kind != kind || binding.Availability != "resolved" {
			continue
		}
		entry, ok := registry.FindSkill(index, binding.Package)
		if !ok {
			return fmt.Errorf("pinned %s dependency not found: %s", kind, binding.Package)
		}
		if _, err := registry.ResolveVersion(entry, binding.Version); err != nil {
			return fmt.Errorf("pinned %s dependency %s@%s is unavailable: %w", kind, binding.Package, binding.Version, err)
		}
		if entry.Latest != binding.Version {
			return fmt.Errorf("local dependency installer cannot select historical %s %s@%s (latest is %s)", kind, binding.Package, binding.Version, entry.Latest)
		}
	}
	return nil
}

func CompileAgentRuntimePackage(agentDir, runtimeName, agentTargetRoot string) (string, error) {
	return compileAgentRuntimePackage(agentDir, runtimeName, agentTargetRoot, "")
}

func compileAgentRuntimePackage(agentDir, runtimeName, agentTargetRoot, bundledRoot string) (string, error) {
	manifest, err := registry.ValidatePackageManifest(filepath.Join(agentDir, "agent.yaml"))
	if err != nil {
		return "", fmt.Errorf("validate agent compiler input: %w", err)
	}
	if manifest.Orchestration == nil {
		return "", fmt.Errorf("agent %s has no orchestration binding contract", manifest.ID)
	}
	runtimeName = strings.ToLower(strings.TrimSpace(runtimeName))
	if runtimeName != "codex" && runtimeName != "claude" && runtimeName != "generic" {
		return "", fmt.Errorf("unsupported agent runtime %q", runtimeName)
	}
	compiled := compiledAgentRuntime{
		SchemaVersion: agentRuntimeSchema, Runtime: runtimeName,
		Agent:  compiledRuntimePackageRoot{ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Description: manifest.Description},
		Status: "ready", BlockingReasons: []string{}, Model: manifest.Orchestration.Model,
		Memory: manifest.Orchestration.Memory, Governance: manifest.Orchestration.Governance,
		Bindings: []compiledAgentBinding{},
	}
	for _, binding := range manifest.Orchestration.Bindings {
		resolved := compiledAgentBinding{Name: binding.Name, Kind: binding.Kind, Package: binding.Package, Version: binding.Version, Requirement: binding.Requirement, Access: binding.Access}
		if binding.Availability == "unavailable" {
			resolved.Status, resolved.Reason = "unavailable", binding.Reason
		} else {
			module := map[string]string{"skill": "skills", "agent": "agents", "tool": "tools"}[binding.Kind]
			target, targetErr := ResolvePluginDependencyTarget(runtimeName, module, agentTargetRoot)
			if targetErr != nil {
				return "", targetErr
			}
			dependencyDir := filepath.Join(target.TargetPath, filepath.FromSlash(binding.Package))
			dependencySourceDir := dependencyDir
			if bundledRoot != "" {
				moduleDir := map[string]string{"skill": "skills", "agent": "agents", "tool": "tools-mcp"}[binding.Kind]
				dependencySourceDir = filepath.Join(bundledRoot, "bundled", moduleDir, filepath.FromSlash(binding.Package))
			}
			resolved.Path = dependencyDir
			dependencyManifestName, _ := manifestNameForModule(module)
			dependencyManifest, validateErr := registry.ValidateHistoricalPackageManifest(filepath.Join(dependencySourceDir, dependencyManifestName))
			if validateErr != nil || dependencyManifest.Version != binding.Version {
				resolved.Status = "missing"
				resolved.Reason = fmt.Sprintf("install pinned %s dependency %s@%s", binding.Kind, binding.Package, binding.Version)
			} else {
				resolved.Status = dependencyRuntimeStatus(binding.Kind, dependencyManifest.Usability.Availability)
				if binding.Kind == "tool" {
					resolved.Endpoint = discoverToolEndpoint(dependencySourceDir)
					if dependencyManifest.Authentication.Status != "none" {
						resolved.ConfigureHint = fmt.Sprintf("skills-hub auth status %s@%s --module tools --runtime %s --target %q", binding.Package, binding.Version, runtimeName, target.TargetPath)
					}
				}
			}
		}
		if binding.Requirement == "required" && resolved.Status != "ready" {
			if resolved.Status == "setup-required" && compiled.Status == "ready" {
				compiled.Status = "setup-required"
			}
			if resolved.Status != "setup-required" {
				compiled.Status = "blocked"
			}
			compiled.BlockingReasons = append(compiled.BlockingReasons, fmt.Sprintf("%s: %s", binding.Name, firstNonEmpty(resolved.Reason, resolved.Status)))
		}
		compiled.Bindings = append(compiled.Bindings, resolved)
	}
	runtimeDir := filepath.Join(agentDir, ".runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(runtimeDir, runtimeName+".json")
	if err := writeDeterministicJSON(path, compiled); err != nil {
		return "", err
	}
	return path, nil
}

func dependencyRuntimeStatus(kind, availability string) string {
	if kind == "skill" && availability == "documentation-only" {
		return "ready"
	}
	switch availability {
	case "usable-now":
		return "ready"
	case "setup-required":
		return "setup-required"
	case "not-verified", "template-only", "documentation-only":
		return "not-ready"
	default:
		return "not-ready"
	}
}

func discoverToolEndpoint(packageDir string) string {
	for _, name := range []string{"mcp.json", ".mcp.json"} {
		payload, err := os.ReadFile(filepath.Join(packageDir, name))
		if err != nil {
			continue
		}
		var doc mcpServerDocument
		if json.Unmarshal(payload, &doc) == nil && len(doc.MCPServers) > 0 {
			names := sortedMCPServerNames(doc.MCPServers)
			return "mcp://" + names[0]
		}
	}
	return "package://" + filepath.ToSlash(packageDir)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "not ready"
}
