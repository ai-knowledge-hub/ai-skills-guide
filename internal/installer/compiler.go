package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

const (
	runtimePackageSchema       = "skills-hub.runtime-package/v1"
	agentPluginManifestSchema  = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
	agentPluginMCPConfigSchema = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"
)

type runtimeDependencyLock struct {
	LockVersion string                           `json:"lock_version"`
	Root        runtimeDependencyLockRoot        `json:"root"`
	Components  []runtimeDependencyLockComponent `json:"components"`
}

type runtimeDependencyLockRoot struct {
	Module  string `json:"module"`
	ID      string `json:"id"`
	Version string `json:"version"`
}

type runtimeDependencyLockComponent struct {
	Module       string   `json:"module"`
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	ManifestPath string   `json:"manifest_path"`
	Dependencies []string `json:"dependencies"`
}

type compiledRuntimePackage struct {
	SchemaVersion string                       `json:"schema_version"`
	Runtime       string                       `json:"runtime"`
	SourceLock    string                       `json:"source_lock"`
	Package       compiledRuntimePackageRoot   `json:"package"`
	Registrations compiledRuntimeRegistrations `json:"registrations"`
}

type compiledRuntimePackageRoot struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type compiledRuntimeRegistrations struct {
	Skills     []compiledRuntimeRegistration `json:"skills"`
	Agents     []compiledRuntimeRegistration `json:"agents"`
	MCPServers []compiledRuntimeRegistration `json:"mcp_servers"`
	Config     []compiledRuntimeRegistration `json:"config"`
	Hooks      []compiledRuntimeRegistration `json:"hooks"`
}

type compiledRuntimeRegistration struct {
	ID          string   `json:"id"`
	Version     string   `json:"version,omitempty"`
	Path        string   `json:"path"`
	Enforcement string   `json:"enforcement"`
	Reason      string   `json:"reason,omitempty"`
	Names       []string `json:"names,omitempty"`
}

type nativePluginManifest struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Skills      interface{} `json:"skills,omitempty"`
	Agents      interface{} `json:"agents,omitempty"`
	Hooks       interface{} `json:"hooks,omitempty"`
	MCPServers  interface{} `json:"mcpServers,omitempty"`
}

type portablePluginManifest struct {
	Schema      string                    `json:"$schema"`
	Name        string                    `json:"name"`
	Version     string                    `json:"version,omitempty"`
	Description string                    `json:"description,omitempty"`
	Extensions  map[string]map[string]any `json:"extensions,omitempty"`
}

type mcpServerDocument struct {
	Schema     string                     `json:"$schema,omitempty"`
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

type portableMCPStdioServer struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	CWD     string            `json:"cwd,omitempty"`
}

// CompilePluginRuntimePackage translates a verified, self-contained plugin
// release into the selected runtime's discoverable package layout. The lock is
// the authority for component identity and order; generic plugin.json metadata
// is never treated as proof of a complete runtime package.
func CompilePluginRuntimePackage(packageDir, runtimeName string) ([]string, error) {
	runtimeName = strings.ToLower(strings.TrimSpace(runtimeName))
	if runtimeName != "codex" && runtimeName != "claude" && runtimeName != "generic" {
		return nil, fmt.Errorf("unsupported runtime compiler %q", runtimeName)
	}

	manifestPath := filepath.Join(packageDir, "plugin.yaml")
	manifest, err := registry.ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("parse compiler source manifest: %w", err)
	}
	if !manifest.Artifact.SelfContained || manifest.Artifact.DependencyLock == nil {
		if manifest.SchemaVersion != "2.1" {
			return prepareLegacyRuntimeArtifact(packageDir, runtimeName)
		}
		return nil, fmt.Errorf("plugin %s@%s has no self-contained dependency lock; minimal plugin metadata is not a complete runtime package", manifest.ID, manifest.Version)
	}
	if stat, statErr := os.Stat(filepath.Join(packageDir, "bundled")); os.IsNotExist(statErr) || (statErr == nil && !stat.IsDir()) {
		// Repository source installs resolve dependencies into sibling runtime
		// directories and have no archive-local closure to compile. Keep them on
		// the visibly legacy path; published lock-bearing releases must include
		// bundled/ and are compiled below.
		return prepareLegacyRuntimeArtifact(packageDir, runtimeName)
	}
	if err := validateCleanRuntimeDiscoveryInputs(packageDir, manifest); err != nil {
		return nil, err
	}
	manifest, err = registry.ValidatePackageManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("validate compiler source manifest: %w", err)
	}
	lockPath := filepath.Join(packageDir, filepath.FromSlash(*manifest.Artifact.DependencyLock))
	lock, err := loadRuntimeDependencyLock(lockPath, manifest)
	if err != nil {
		return nil, err
	}

	compiled := compiledRuntimePackage{
		SchemaVersion: runtimePackageSchema,
		Runtime:       runtimeName,
		SourceLock:    filepath.ToSlash(*manifest.Artifact.DependencyLock),
		Package: compiledRuntimePackageRoot{
			ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Description: manifest.Description,
		},
		Registrations: compiledRuntimeRegistrations{
			Skills: []compiledRuntimeRegistration{}, Agents: []compiledRuntimeRegistration{},
			MCPServers: []compiledRuntimeRegistration{}, Config: []compiledRuntimeRegistration{},
			Hooks: []compiledRuntimeRegistration{},
		},
	}

	nativeNames := make(map[string]string, len(lock.Components))
	for _, component := range lock.Components {
		if component.Module == "skills" || component.Module == "agents" {
			key := component.Module + ":" + filepath.Base(filepath.FromSlash(component.ID))
			if previous, exists := nativeNames[key]; exists {
				return nil, fmt.Errorf("locked components %s and %s collide at native runtime name %s", previous, component.ID, key)
			}
			nativeNames[key] = component.ID
		}
		registration, err := materializeRuntimeComponent(packageDir, runtimeName, component)
		if err != nil {
			return nil, err
		}
		switch component.Module {
		case "skills":
			compiled.Registrations.Skills = append(compiled.Registrations.Skills, registration)
		case "agents":
			compiled.Registrations.Agents = append(compiled.Registrations.Agents, registration)
		case "tools-mcp":
			compiled.Registrations.MCPServers = append(compiled.Registrations.MCPServers, registration)
		default:
			return nil, fmt.Errorf("dependency lock contains unsupported runtime module %q", component.Module)
		}
	}
	if runtimeName == "codex" && len(compiled.Registrations.MCPServers) > 0 {
		if err := compileCodexMCPConfig(packageDir, lock.Components, compiled.Registrations.MCPServers); err != nil {
			return nil, err
		}
	}

	if stat, statErr := os.Stat(filepath.Join(packageDir, "config")); statErr == nil && stat.IsDir() {
		compiled.Registrations.Config = append(compiled.Registrations.Config, compileConfigRegistration(runtimeName))
	}
	for _, hookName := range manifest.Includes.Hooks {
		registration, err := compileHookRegistration(packageDir, runtimeName, hookName)
		if err != nil {
			return nil, err
		}
		compiled.Registrations.Hooks = append(compiled.Registrations.Hooks, registration)
	}

	runtimeDir := filepath.Join(packageDir, ".runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, fmt.Errorf("create runtime contract directory: %w", err)
	}
	contractPath := filepath.Join(runtimeDir, runtimeName+".json")
	if err := writeDeterministicJSON(contractPath, compiled); err != nil {
		return nil, err
	}
	artifacts := []string{contractPath}

	if runtimeName != "generic" {
		nativePath, err := writeNativePluginManifest(packageDir, runtimeName, manifest, compiled)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, nativePath)
	}
	if err := validateFinalRuntimeDiscovery(packageDir, runtimeName, compiled); err != nil {
		return nil, fmt.Errorf("validate final %s discovery layout: %w", runtimeName, err)
	}
	if err := ValidateCompiledRuntimePackage(packageDir, runtimeName); err != nil {
		return nil, fmt.Errorf("validate compiled %s runtime package: %w", runtimeName, err)
	}
	return artifacts, nil
}

func compileConfigRegistration(runtimeName string) compiledRuntimeRegistration {
	registration := compiledRuntimeRegistration{ID: "plugin-config", Path: "./config", Enforcement: "native"}
	if runtimeName == "codex" || runtimeName == "claude" {
		registration.Enforcement = "advisory"
		registration.Reason = "the selected runtime has no native registration contract for an arbitrary config directory"
	}
	return registration
}

func validateCleanRuntimeDiscoveryInputs(packageDir string, manifest registry.Manifest) error {
	reserved := []string{
		"skills", "agents", "commands", "bin", "monitors", "output-styles", "themes", "workflows",
		"SKILL.md", ".mcp.json", "mcp.json", ".lsp.json", "settings.json", ".codex-plugin", ".claude-plugin",
	}
	for _, relative := range reserved {
		if _, err := os.Lstat(filepath.Join(packageDir, filepath.FromSlash(relative))); err == nil {
			return fmt.Errorf("runtime discovery input %s is not derived from dependencies.lock.json", relative)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect runtime discovery input %s: %w", relative, err)
		}
	}

	defaultHookPath := filepath.Join(packageDir, "hooks", "hooks.json")
	if _, err := os.Lstat(defaultHookPath); err == nil {
		declared := false
		for _, hookName := range manifest.Includes.Hooks {
			resolved, resolveErr := resolveHookPath(filepath.Join(packageDir, "hooks"), hookName)
			if resolveErr == nil && resolved == defaultHookPath {
				declared = true
				break
			}
		}
		if !declared {
			return fmt.Errorf("runtime discovery input hooks/hooks.json is not declared by the plugin manifest")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect runtime discovery input hooks/hooks.json: %w", err)
	}
	return nil
}

func validateFinalRuntimeDiscovery(packageDir, runtimeName string, compiled compiledRuntimePackage) error {
	wantSkills := make(map[string]struct{}, len(compiled.Registrations.Skills))
	for _, registration := range compiled.Registrations.Skills {
		if registration.Enforcement == "native" {
			wantSkills[filepath.Base(filepath.FromSlash(registration.Path))] = struct{}{}
		}
	}
	if err := validateDiscoveryDirectory(filepath.Join(packageDir, "skills"), wantSkills); err != nil {
		return fmt.Errorf("skills: %w", err)
	}

	wantAgents := make(map[string]struct{})
	if runtimeName == "claude" {
		for _, registration := range compiled.Registrations.Agents {
			if registration.Enforcement == "native" {
				wantAgents[filepath.Base(filepath.FromSlash(registration.Path))] = struct{}{}
			}
		}
	}
	if err := validateDiscoveryDirectory(filepath.Join(packageDir, "agents"), wantAgents); err != nil {
		return fmt.Errorf("agents: %w", err)
	}
	if runtimeName == "codex" || runtimeName == "claude" {
		defaultHookPath := filepath.Join(packageDir, "hooks", "hooks.json")
		_, defaultHookErr := os.Stat(defaultHookPath)
		expectedDefaultHook := false
		for _, registration := range compiled.Registrations.Hooks {
			if registration.Enforcement == "native" && registration.Path == "./hooks/hooks.json" {
				expectedDefaultHook = true
				break
			}
		}
		if expectedDefaultHook && defaultHookErr != nil {
			return fmt.Errorf("native default hook document is unavailable")
		}
		if !expectedDefaultHook && defaultHookErr == nil {
			return fmt.Errorf("unregistered hooks/hooks.json is discoverable")
		}
	}

	if runtimeName == "codex" {
		hasNativeMCP := false
		wantServerNames := make(map[string]struct{})
		for _, registration := range compiled.Registrations.MCPServers {
			if registration.Enforcement == "native" {
				hasNativeMCP = true
				if registration.Path != "./mcp.json" {
					return fmt.Errorf("Codex MCP registration %s does not target root mcp.json", registration.ID)
				}
				for _, name := range registration.Names {
					if _, duplicate := wantServerNames[name]; duplicate {
						return fmt.Errorf("Codex MCP server name %s is registered more than once", name)
					}
					wantServerNames[name] = struct{}{}
				}
			}
		}
		mcpPath := filepath.Join(packageDir, "mcp.json")
		_, err := os.Stat(mcpPath)
		if hasNativeMCP && err != nil {
			return fmt.Errorf("native Codex MCP config is unavailable")
		}
		if !hasNativeMCP && err == nil {
			return fmt.Errorf("unregistered Codex mcp.json is discoverable")
		}
		if hasNativeMCP {
			document, err := loadMCPServerConfig(mcpPath)
			if err != nil {
				return err
			}
			if document.Schema != agentPluginMCPConfigSchema {
				return fmt.Errorf("Codex mcp.json does not declare the portable MCP schema")
			}
			if err := validatePortableMCPServers(document.MCPServers); err != nil {
				return err
			}
			if err := compareMCPServerNames(document.MCPServers, wantServerNames); err != nil {
				return err
			}
		}
	} else {
		for _, registration := range compiled.Registrations.MCPServers {
			if registration.Enforcement != "native" {
				continue
			}
			configPath := filepath.Join(packageDir, filepath.FromSlash(strings.TrimPrefix(registration.Path, "./")))
			document, err := loadMCPServerConfig(configPath)
			if err != nil {
				return err
			}
			wantServerNames := make(map[string]struct{}, len(registration.Names))
			for _, name := range registration.Names {
				wantServerNames[name] = struct{}{}
			}
			if err := compareMCPServerNames(document.MCPServers, wantServerNames); err != nil {
				return fmt.Errorf("MCP registration %s: %w", registration.ID, err)
			}
		}
	}
	if err := validateNativeManifestDiscovery(packageDir, runtimeName, compiled); err != nil {
		return err
	}
	return nil
}

func compareMCPServerNames(servers map[string]json.RawMessage, expected map[string]struct{}) error {
	if len(servers) != len(expected) {
		return fmt.Errorf("discoverable MCP server count %d does not match runtime contract count %d", len(servers), len(expected))
	}
	for name := range servers {
		if _, ok := expected[name]; !ok {
			return fmt.Errorf("discoverable MCP server %s is absent from the runtime contract", name)
		}
	}
	return nil
}

func validateNativeManifestDiscovery(packageDir, runtimeName string, compiled compiledRuntimePackage) error {
	if runtimeName == "generic" {
		return nil
	}
	wantHooks := nativeRegistrationPaths(compiled.Registrations.Hooks)
	if runtimeName == "codex" {
		payload, err := os.ReadFile(filepath.Join(packageDir, "plugin.json"))
		if err != nil {
			return fmt.Errorf("read Codex portable manifest: %w", err)
		}
		var manifest portablePluginManifest
		if err := decodeStrictJSON(payload, &manifest); err != nil {
			return fmt.Errorf("parse Codex portable manifest: %w", err)
		}
		if manifest.Schema != agentPluginManifestSchema || manifest.Version != compiled.Package.Version {
			return fmt.Errorf("Codex portable manifest identity does not match the runtime contract")
		}
		if len(manifest.Extensions) == 0 {
			if len(wantHooks) != 0 {
				return fmt.Errorf("Codex portable manifest omits native hooks")
			}
			return nil
		}
		if len(manifest.Extensions) != 1 {
			return fmt.Errorf("Codex portable manifest contains undeclared runtime extensions")
		}
		openAI, ok := manifest.Extensions["com.openai"]
		if !ok || len(openAI) != 1 {
			return fmt.Errorf("Codex portable manifest contains undeclared OpenAI extensions")
		}
		actualHooks, err := normalizeNativePathField(openAI["hooks"])
		if err != nil || !equalStringSets(actualHooks, wantHooks) {
			return fmt.Errorf("Codex portable hook discovery does not match the runtime contract")
		}
		return nil
	}

	payload, err := os.ReadFile(filepath.Join(packageDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		return fmt.Errorf("read Claude manifest: %w", err)
	}
	var manifest nativePluginManifest
	if err := decodeStrictJSON(payload, &manifest); err != nil {
		return fmt.Errorf("parse Claude manifest: %w", err)
	}
	if manifest.Version != compiled.Package.Version {
		return fmt.Errorf("Claude manifest identity does not match the runtime contract")
	}
	wantSkills := []string{}
	if len(nativeRegistrationPaths(compiled.Registrations.Skills)) > 0 {
		wantSkills = []string{"./skills/"}
	}
	wantAgents := []string{}
	if len(nativeRegistrationPaths(compiled.Registrations.Agents)) > 0 {
		wantAgents = []string{"./agents/"}
	}
	wantMCP := nativeRegistrationPaths(compiled.Registrations.MCPServers)
	for label, comparison := range map[string]struct {
		actual any
		want   []string
	}{
		"skills":      {actual: manifest.Skills, want: wantSkills},
		"agents":      {actual: manifest.Agents, want: wantAgents},
		"hooks":       {actual: manifest.Hooks, want: wantHooks},
		"MCP servers": {actual: manifest.MCPServers, want: wantMCP},
	} {
		actual, err := normalizeNativePathField(comparison.actual)
		if err != nil || !equalStringSets(actual, comparison.want) {
			return fmt.Errorf("Claude %s discovery does not match the runtime contract", label)
		}
	}
	return nil
}

func nativeRegistrationPaths(registrations []compiledRuntimeRegistration) []string {
	paths := make([]string, 0, len(registrations))
	for _, registration := range registrations {
		if registration.Enforcement == "native" {
			paths = append(paths, registration.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

func normalizeNativePathField(value any) ([]string, error) {
	if value == nil {
		return []string{}, nil
	}
	paths := make([]string, 0)
	switch typed := value.(type) {
	case string:
		paths = append(paths, typed)
	case []any:
		for _, item := range typed {
			path, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("native path list contains a non-string value")
			}
			paths = append(paths, path)
		}
	default:
		return nil, fmt.Errorf("native path field has unsupported type")
	}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		canonicalPath := strings.TrimSuffix(path, "/")
		if err := validateCompiledRelativePath(canonicalPath); err != nil {
			return nil, err
		}
		if _, duplicate := seen[path]; duplicate {
			return nil, fmt.Errorf("native path field repeats %s", path)
		}
		seen[path] = struct{}{}
	}
	sort.Strings(paths)
	return paths, nil
}

func equalStringSets(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	return strings.Join(left, "\x00") == strings.Join(right, "\x00")
}

func validateDiscoveryDirectory(path string, expected map[string]struct{}) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		if len(expected) == 0 {
			return nil
		}
		return fmt.Errorf("missing discovery directory")
	}
	if err != nil {
		return err
	}
	if len(entries) != len(expected) {
		return fmt.Errorf("contains %d entries; runtime contract declares %d", len(entries), len(expected))
	}
	for _, entry := range entries {
		if _, ok := expected[entry.Name()]; !ok {
			return fmt.Errorf("contains undeclared entry %s", entry.Name())
		}
	}
	return nil
}

// prepareLegacyRuntimeArtifact preserves installs of pre-2.1 packages without
// representing their three-field manifest as a compiled runtime contract.
// Only lock-bearing 2.1 releases enter the native compiler above.
func prepareLegacyRuntimeArtifact(packageDir, runtimeName string) ([]string, error) {
	if runtimeName == "generic" {
		return nil, nil
	}
	sourceManifest := filepath.Join(packageDir, "plugin.json")
	payload, err := os.ReadFile(sourceManifest)
	if err != nil {
		return nil, fmt.Errorf("read legacy plugin manifest: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, fmt.Errorf("parse legacy plugin manifest: %w", err)
	}
	manifest["runtime"] = runtimeName
	manifest["compatibility"] = "legacy-uncompiled"
	runtimeDir := filepath.Join(packageDir, "."+runtimeName+"-plugin")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, fmt.Errorf("create legacy runtime manifest directory: %w", err)
	}
	runtimeManifestPath := filepath.Join(runtimeDir, "plugin.json")
	if err := writeDeterministicJSON(runtimeManifestPath, manifest); err != nil {
		return nil, err
	}
	return []string{runtimeManifestPath}, nil
}

func loadRuntimeDependencyLock(lockPath string, manifest registry.Manifest) (runtimeDependencyLock, error) {
	payload, err := os.ReadFile(lockPath)
	if err != nil {
		return runtimeDependencyLock{}, fmt.Errorf("read runtime dependency lock: %w", err)
	}
	var lock runtimeDependencyLock
	if err := json.Unmarshal(payload, &lock); err != nil {
		return runtimeDependencyLock{}, fmt.Errorf("parse runtime dependency lock: %w", err)
	}
	if lock.LockVersion != "1.0" || lock.Root.Module != "plugins" || lock.Root.ID != manifest.ID || lock.Root.Version != manifest.Version {
		return runtimeDependencyLock{}, fmt.Errorf("runtime dependency lock root does not match plugins:%s@%s", manifest.ID, manifest.Version)
	}
	sort.Slice(lock.Components, func(i, j int) bool {
		return lock.Components[i].Module+":"+lock.Components[i].ID < lock.Components[j].Module+":"+lock.Components[j].ID
	})
	seen := make(map[string]struct{}, len(lock.Components))
	for _, component := range lock.Components {
		key := component.Module + ":" + component.ID
		if _, exists := seen[key]; exists {
			return runtimeDependencyLock{}, fmt.Errorf("runtime dependency lock repeats component %s", key)
		}
		seen[key] = struct{}{}
		manifestName := map[string]string{"skills": "skill.yaml", "agents": "agent.yaml", "tools-mcp": "tool.yaml"}[component.Module]
		wantPath := "bundled/" + component.Module + "/" + component.ID + "/" + manifestName
		if manifestName == "" || component.ManifestPath != wantPath {
			return runtimeDependencyLock{}, fmt.Errorf("runtime dependency lock component %s has non-canonical manifest path", key)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(lockPath), filepath.FromSlash(component.ManifestPath))); err != nil {
			return runtimeDependencyLock{}, fmt.Errorf("runtime dependency lock component %s is unavailable: %w", key, err)
		}
	}
	return lock, nil
}

func materializeRuntimeComponent(packageDir, runtimeName string, component runtimeDependencyLockComponent) (compiledRuntimeRegistration, error) {
	sourceRoot := filepath.Dir(filepath.Join(packageDir, filepath.FromSlash(component.ManifestPath)))
	slug := filepath.Base(filepath.FromSlash(component.ID))
	registration := compiledRuntimeRegistration{ID: component.ID, Version: component.Version}

	switch component.Module {
	case "skills":
		target := filepath.Join(packageDir, "skills", slug)
		if err := replaceCompiledTree(sourceRoot, target); err != nil {
			return registration, fmt.Errorf("compile skill %s: %w", component.ID, err)
		}
		registration.Path = "./skills/" + slug
		registration.Enforcement = "native"
	case "agents":
		componentManifest, err := registry.ValidatePackageManifest(filepath.Join(packageDir, filepath.FromSlash(component.ManifestPath)))
		if err != nil {
			return registration, fmt.Errorf("validate agent %s: %w", component.ID, err)
		}
		spec := strings.TrimSpace(componentManifest.Entrypoints["spec"])
		if spec == "" {
			spec = "AGENT.md"
		}
		sourcePath := filepath.Join(sourceRoot, filepath.FromSlash(spec))
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			return registration, fmt.Errorf("read agent %s entrypoint: %w", component.ID, err)
		}
		if runtimeName == "claude" {
			target := filepath.Join(packageDir, "agents", slug+".md")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return registration, fmt.Errorf("create agent directory: %w", err)
			}
			description, _ := json.Marshal(componentManifest.Description)
			rendered := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", slug, description, strings.TrimLeft(string(source), "\n"))
			if !strings.HasSuffix(rendered, "\n") {
				rendered += "\n"
			}
			if err := os.WriteFile(target, []byte(rendered), 0o644); err != nil {
				return registration, fmt.Errorf("compile agent %s: %w", component.ID, err)
			}
			registration.Path = "./agents/" + slug + ".md"
			registration.Enforcement = "native"
		} else {
			relativeSource, err := filepath.Rel(packageDir, sourcePath)
			if err != nil {
				return registration, fmt.Errorf("resolve advisory agent %s: %w", component.ID, err)
			}
			registration.Path = "./" + filepath.ToSlash(relativeSource)
			registration.Enforcement = "advisory"
			registration.Reason = "the selected runtime has no native plugin-agent registration contract"
		}
	case "tools-mcp":
		registration.Path = "./bundled/tools-mcp/" + component.ID
		registration.Enforcement = "advisory"
		registration.Reason = "the locked tool package does not contain a launchable MCP server configuration"
		for _, candidate := range []string{".mcp.json", "mcp.json"} {
			candidatePath := filepath.Join(sourceRoot, candidate)
			if _, err := os.Stat(candidatePath); err == nil {
				document, err := loadMCPServerConfig(candidatePath)
				if err != nil {
					return registration, fmt.Errorf("validate MCP server config for %s: %w", component.ID, err)
				}
				registration.Names = sortedMCPServerNames(document.MCPServers)
				if runtimeName == "codex" {
					if err := validatePortableMCPServers(document.MCPServers); err != nil {
						return registration, fmt.Errorf("validate Codex MCP server config for %s: %w", component.ID, err)
					}
					registration.Path = "./mcp.json"
				} else {
					registration.Path = registration.Path + "/" + candidate
				}
				registration.Enforcement = "native"
				registration.Reason = ""
				break
			}
		}
	}
	return registration, nil
}

func sortedMCPServerNames(servers map[string]json.RawMessage) []string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func loadMCPServerConfig(path string) (mcpServerDocument, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return mcpServerDocument{}, err
	}
	var document mcpServerDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		return mcpServerDocument{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	if len(document.MCPServers) == 0 {
		return mcpServerDocument{}, fmt.Errorf("%s declares no MCP servers", filepath.Base(path))
	}
	for name, raw := range document.MCPServers {
		if strings.TrimSpace(name) == "" {
			return mcpServerDocument{}, fmt.Errorf("%s contains an unnamed MCP server", filepath.Base(path))
		}
		var server map[string]any
		if err := json.Unmarshal(raw, &server); err != nil || len(server) == 0 {
			return mcpServerDocument{}, fmt.Errorf("MCP server %s has no configuration", name)
		}
		command, _ := server["command"].(string)
		urlValue, _ := server["url"].(string)
		if strings.TrimSpace(command) == "" && strings.TrimSpace(urlValue) == "" {
			return mcpServerDocument{}, fmt.Errorf("MCP server %s has neither command nor URL", name)
		}
	}
	return document, nil
}

func validatePortableMCPServers(servers map[string]json.RawMessage) error {
	for name, raw := range servers {
		var discriminator struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &discriminator); err != nil {
			return fmt.Errorf("MCP server %s is not an object", name)
		}
		switch discriminator.Type {
		case "stdio":
			var server portableMCPStdioServer
			if err := decodeStrictJSON(raw, &server); err != nil || strings.TrimSpace(server.Command) == "" {
				return fmt.Errorf("stdio MCP server %s must contain only portable fields and a non-empty command", name)
			}
			if !isPortableMCPCommand(server.Command) {
				return fmt.Errorf("stdio MCP server %s has a non-portable command", name)
			}
			if server.CWD != "" && !isPortableMCPWorkingDirectory(server.CWD) {
				return fmt.Errorf("stdio MCP server %s has a non-portable cwd", name)
			}
			if _, reserved := server.Env["PLUGIN_ROOT"]; reserved {
				return fmt.Errorf("stdio MCP server %s overrides reserved PLUGIN_ROOT", name)
			}
			if _, reserved := server.Env["PLUGIN_DATA"]; reserved {
				return fmt.Errorf("stdio MCP server %s overrides reserved PLUGIN_DATA", name)
			}
		case "streamable-http", "sse":
			var server struct {
				Type    string            `json:"type"`
				URL     string            `json:"url"`
				Headers map[string]string `json:"headers,omitempty"`
			}
			if err := decodeStrictJSON(raw, &server); err != nil || strings.TrimSpace(server.URL) == "" {
				return fmt.Errorf("%s MCP server %s must contain only portable fields and a non-empty URL", discriminator.Type, name)
			}
			if err := validatePortableMCPRemoteEndpoint(server.URL, server.Headers); err != nil {
				return fmt.Errorf("%s MCP server %s: %w", discriminator.Type, name, err)
			}
		default:
			return fmt.Errorf("MCP server %s requires portable transport type stdio, streamable-http, or sse", name)
		}
	}
	return nil
}

func validatePortableMCPRemoteEndpoint(rawURL string, headers map[string]string) error {
	endpoint, err := url.Parse(rawURL)
	if err != nil || endpoint.Host == "" || !endpoint.IsAbs() {
		return fmt.Errorf("URL must be an absolute HTTP or HTTPS URL")
	}
	if !strings.EqualFold(endpoint.Scheme, "http") && !strings.EqualFold(endpoint.Scheme, "https") {
		return fmt.Errorf("URL must use HTTP or HTTPS")
	}
	if endpoint.User != nil {
		return fmt.Errorf("URL must not contain user information")
	}
	if endpoint.Fragment != "" {
		return fmt.Errorf("URL must not contain a fragment")
	}
	if strings.EqualFold(endpoint.Scheme, "http") && !isPortableMCPLoopbackHost(endpoint.Hostname()) {
		return fmt.Errorf("non-loopback URLs must use HTTPS")
	}
	seenHeaders := make(map[string]string, len(headers))
	for name, value := range headers {
		if isCredentialBearingMCPHeader(name, value) {
			return fmt.Errorf("remote MCP headers must not contain credentials; credentials must be supplied by the runtime")
		}
		if !isHTTPHeaderName(name) {
			return fmt.Errorf("header name %q is invalid", name)
		}
		canonical := strings.ToLower(name)
		if previous, exists := seenHeaders[canonical]; exists {
			return fmt.Errorf("header names %q and %q differ only by case", previous, name)
		}
		seenHeaders[canonical] = name
		if !isHTTPHeaderValue(value) {
			return fmt.Errorf("header %q has an invalid value", name)
		}
	}
	return nil
}

func isCredentialBearingMCPHeader(name, value string) bool {
	return isCredentialBearingHTTPHeaderName(name) || registry.ContainsCredentialShapedValue(name) || registry.ContainsCredentialShapedValue(value) || registry.ContainsCredentialShapedValue(name+"="+value)
}

func isCredentialBearingHTTPHeaderName(name string) bool {
	normalized := strings.ToLower(name)
	switch normalized {
	case "authorization", "proxy-authorization", "cookie", "set-cookie", "api-key", "x-api-key", "access-token", "x-access-token", "auth-token", "x-auth-token":
		return true
	}
	for _, suffix := range []string{"-api-key", "-access-key", "-client-secret", "-refresh-token", "-session-token"} {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	return false
}

func isPortableMCPLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func isHTTPHeaderName(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			continue
		}
		if !strings.ContainsRune("!#$%&'*+-.^_`|~", rune(character)) {
			return false
		}
	}
	return true
}

func isHTTPHeaderValue(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < ' ' && character != '\t') || character == 0x7f {
			return false
		}
	}
	return true
}

func decodeStrictJSON(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func isPortableMCPWorkingDirectory(value string) bool {
	for _, prefix := range []string{"${PLUGIN_ROOT}", "${PLUGIN_DATA}"} {
		if value == prefix {
			return true
		}
		if strings.HasPrefix(value, prefix+"/") {
			return validateCompiledRelativePath("./"+strings.TrimPrefix(value, prefix+"/")) == nil
		}
	}
	return validateCompiledRelativePath(value) == nil
}

func isPortableMCPCommand(value string) bool {
	if strings.HasPrefix(value, "./") {
		return validateCompiledRelativePath(value) == nil
	}
	return value != "" && !strings.ContainsAny(value, "/\\\x00 \t\r\n")
}

func compileCodexMCPConfig(packageDir string, components []runtimeDependencyLockComponent, registrations []compiledRuntimeRegistration) error {
	nativeIDs := make(map[string]struct{})
	for _, registration := range registrations {
		if registration.Enforcement == "native" {
			nativeIDs[registration.ID] = struct{}{}
		}
	}
	if len(nativeIDs) == 0 {
		return nil
	}
	merged := mcpServerDocument{Schema: agentPluginMCPConfigSchema, MCPServers: make(map[string]json.RawMessage)}
	for _, component := range components {
		if component.Module != "tools-mcp" {
			continue
		}
		if _, ok := nativeIDs[component.ID]; !ok {
			continue
		}
		sourceRoot := filepath.Dir(filepath.Join(packageDir, filepath.FromSlash(component.ManifestPath)))
		configPath, err := findMCPServerConfig(sourceRoot)
		if err != nil {
			return fmt.Errorf("compile Codex MCP component %s: %w", component.ID, err)
		}
		document, err := loadMCPServerConfig(configPath)
		if err != nil {
			return fmt.Errorf("compile Codex MCP component %s: %w", component.ID, err)
		}
		if err := validatePortableMCPServers(document.MCPServers); err != nil {
			return fmt.Errorf("compile Codex MCP component %s: %w", component.ID, err)
		}
		for name, server := range document.MCPServers {
			if _, exists := merged.MCPServers[name]; exists {
				return fmt.Errorf("Codex MCP server name %s is declared by more than one locked component", name)
			}
			rebased, err := rebaseCodexMCPServer(packageDir, sourceRoot, name, server)
			if err != nil {
				return fmt.Errorf("compile Codex MCP component %s: %w", component.ID, err)
			}
			merged.MCPServers[name] = rebased
		}
	}
	return writeDeterministicJSON(filepath.Join(packageDir, "mcp.json"), merged)
}

func rebaseCodexMCPServer(packageDir, componentRoot, name string, raw json.RawMessage) (json.RawMessage, error) {
	var discriminator struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &discriminator); err != nil || discriminator.Type != "stdio" {
		return raw, nil
	}
	var server portableMCPStdioServer
	if err := decodeStrictJSON(raw, &server); err != nil {
		return nil, fmt.Errorf("decode stdio MCP server %s: %w", name, err)
	}
	relativeRoot, err := filepath.Rel(packageDir, componentRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve MCP component root: %w", err)
	}
	componentPath := "./" + filepath.ToSlash(relativeRoot)
	if err := validateCompiledRelativePath(componentPath); err != nil {
		return nil, fmt.Errorf("MCP component root: %w", err)
	}
	if strings.HasPrefix(server.Command, "./") {
		server.Command = componentPath + "/" + strings.TrimPrefix(server.Command, "./")
	}
	if server.CWD == "" {
		server.CWD = componentPath
	} else {
		server.CWD = rebaseMCPWorkingDirectory(server.CWD, componentPath)
	}
	for index := range server.Args {
		server.Args[index] = rebaseMCPPluginRootPlaceholder(server.Args[index], componentPath)
	}
	for key, value := range server.Env {
		server.Env[key] = rebaseMCPPluginRootPlaceholder(value, componentPath)
	}
	rebased, err := json.Marshal(server)
	if err != nil {
		return nil, fmt.Errorf("encode stdio MCP server %s: %w", name, err)
	}
	if err := validatePortableMCPServers(map[string]json.RawMessage{name: rebased}); err != nil {
		return nil, err
	}
	return rebased, nil
}

func rebaseMCPWorkingDirectory(value, componentPath string) string {
	if strings.HasPrefix(value, "./") {
		return componentPath + "/" + strings.TrimPrefix(value, "./")
	}
	return rebaseMCPPluginRootPlaceholder(value, componentPath)
}

func rebaseMCPPluginRootPlaceholder(value, componentPath string) string {
	componentSuffix := strings.TrimPrefix(componentPath, ".")
	return strings.ReplaceAll(value, "${PLUGIN_ROOT}", "${PLUGIN_ROOT}"+componentSuffix)
}

func findMCPServerConfig(sourceRoot string) (string, error) {
	for _, candidate := range []string{".mcp.json", "mcp.json"} {
		path := filepath.Join(sourceRoot, candidate)
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("no MCP server configuration")
}

func replaceCompiledTree(source, destination string) error {
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	return copyTree(source, destination)
}

func compileHookRegistration(packageDir, runtimeName, hookName string) (compiledRuntimeRegistration, error) {
	hookPath, err := resolveHookPath(filepath.Join(packageDir, "hooks"), hookName)
	if err != nil {
		return compiledRuntimeRegistration{}, err
	}
	relative, err := filepath.Rel(packageDir, hookPath)
	if err != nil {
		return compiledRuntimeRegistration{}, err
	}
	registration := compiledRuntimeRegistration{ID: hookName, Path: "./" + filepath.ToSlash(relative)}
	if strings.EqualFold(filepath.Ext(hookPath), ".json") && (runtimeName == "codex" || runtimeName == "claude") {
		native, reason, err := validateNativeHookDocument(packageDir, runtimeName, hookPath)
		if err != nil {
			return compiledRuntimeRegistration{}, fmt.Errorf("validate %s hook %s: %w", runtimeName, hookName, err)
		}
		if native {
			registration.Enforcement = "native"
		} else {
			registration.Enforcement = "advisory"
			registration.Reason = reason
		}
	} else {
		registration.Enforcement = "advisory"
		registration.Reason = "the source hook is guidance, not an executable native hook definition"
	}
	if registration.Enforcement == "advisory" && (runtimeName == "codex" || runtimeName == "claude") && registration.Path == "./hooks/hooks.json" {
		quarantined := filepath.Join(packageDir, ".runtime", "advisory", "hooks", "hooks.json")
		if err := copyFile(hookPath, quarantined); err != nil {
			return compiledRuntimeRegistration{}, fmt.Errorf("quarantine advisory default hook: %w", err)
		}
		if err := os.Remove(hookPath); err != nil {
			return compiledRuntimeRegistration{}, fmt.Errorf("remove advisory hook from native discovery: %w", err)
		}
		registration.Path = "./.runtime/advisory/hooks/hooks.json"
	}
	return registration, nil
}

var (
	hookPluginPathPattern         = regexp.MustCompile(`\$\{(?:PLUGIN_ROOT|CLAUDE_PLUGIN_ROOT)\}["']?/([A-Za-z0-9._/-]+)`)
	unrootedHookPathPattern       = regexp.MustCompile(`(?:^|[[:space:]"'])(\./[A-Za-z0-9._/-]+)`)
	hookEnvironmentReference      = regexp.MustCompile(`\$(?:([A-Za-z_][A-Za-z0-9_]*)|\{([A-Za-z_][A-Za-z0-9_]*)\})`)
	hookAuthorizationSchemePrefix = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*[[:space:]]*$`)
)

func validateNativeHookDocument(packageDir, runtimeName, hookPath string) (bool, string, error) {
	payload, err := os.ReadFile(hookPath)
	if err != nil {
		return false, "", err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil {
		return false, "", fmt.Errorf("parse hook JSON: %w", err)
	}
	for key := range document {
		if key != "hooks" && key != "description" {
			return false, "", fmt.Errorf("hook document contains a top-level field that is not defined by the %s runtime", runtimeName)
		}
	}
	if description, ok := document["description"]; ok {
		var value string
		if err := json.Unmarshal(description, &value); err != nil {
			return false, "", fmt.Errorf("hook description must be a string")
		}
	}
	hooksPayload, ok := document["hooks"]
	if !ok {
		return false, "", fmt.Errorf("hook document has no hooks object")
	}
	var events map[string]json.RawMessage
	if err := json.Unmarshal(hooksPayload, &events); err != nil || len(events) == 0 {
		return false, "", fmt.Errorf("hook document must contain at least one hook event")
	}
	sawNative := false
	advisoryReason := ""
	allowedEvents := supportedHookEvents(runtimeName)
	for event, groupsPayload := range events {
		eventAdvisoryReason := ""
		if _, supported := allowedEvents[event]; !supported {
			eventAdvisoryReason = fmt.Sprintf("hook event %s is not supported by the %s runtime", event, runtimeName)
		}
		var groups []map[string]json.RawMessage
		if err := json.Unmarshal(groupsPayload, &groups); err != nil || len(groups) == 0 {
			return false, "", fmt.Errorf("hook event %s must contain at least one matcher group", event)
		}
		for _, group := range groups {
			groupAdvisoryReason := eventAdvisoryReason
			for key := range group {
				if key != "matcher" && key != "hooks" {
					groupAdvisoryReason = fmt.Sprintf("hook event %s uses unsupported matcher field %s", event, key)
				}
			}
			if _, hasMatcher := group["matcher"]; hasMatcher && runtimeName == "claude" && !claudeHookEventSupportsMatcher(event) {
				groupAdvisoryReason = fmt.Sprintf("hook event %s ignores matcher fields in Claude plugin hook documents", event)
			}
			if matcher, ok := group["matcher"]; ok {
				var value string
				if err := json.Unmarshal(matcher, &value); err != nil {
					return false, "", fmt.Errorf("hook event %s matcher must be a string", event)
				}
			}
			handlersPayload, ok := group["hooks"]
			if !ok {
				return false, "", fmt.Errorf("hook event %s matcher group has no handlers", event)
			}
			var handlers []map[string]json.RawMessage
			if err := json.Unmarshal(handlersPayload, &handlers); err != nil || len(handlers) == 0 {
				return false, "", fmt.Errorf("hook event %s matcher group must contain at least one handler", event)
			}
			if groupAdvisoryReason != "" {
				if advisoryReason == "" {
					advisoryReason = groupAdvisoryReason
				}
				continue
			}
			for _, handler := range handlers {
				native, reason, err := validateNativeHookHandler(packageDir, runtimeName, event, handler)
				if err != nil {
					return false, "", err
				}
				if native {
					sawNative = true
				} else if advisoryReason == "" {
					advisoryReason = reason
				}
			}
		}
	}
	if sawNative && advisoryReason != "" {
		return false, "", fmt.Errorf("hook document mixes native and advisory handlers; split them into separate hook documents")
	}
	if advisoryReason != "" {
		return false, advisoryReason, nil
	}
	return sawNative, "", nil
}

func supportedHookEvents(runtimeName string) map[string]struct{} {
	events := []string{"SessionStart", "PreToolUse", "PermissionRequest", "PostToolUse", "PreCompact", "PostCompact", "UserPromptSubmit", "SubagentStart", "SubagentStop", "Stop", "SessionEnd"}
	if runtimeName == "codex" {
		events = append(events, "Interrupt")
	} else {
		events = append(events,
			"Setup", "UserPromptExpansion", "PermissionDenied", "PostToolUseFailure", "PostToolBatch", "Notification", "MessageDisplay",
			"TaskCreated", "TaskCompleted", "StopFailure", "TeammateIdle", "InstructionsLoaded", "ConfigChange", "CwdChanged", "DirectoryAdded",
			"FileChanged", "WorktreeCreate", "WorktreeRemove", "PreModelSwitch", "PostModelSwitch", "Elicitation", "ElicitationResult",
		)
	}
	result := make(map[string]struct{}, len(events))
	for _, event := range events {
		result[event] = struct{}{}
	}
	return result
}

func claudeHookEventSupportsMatcher(event string) bool {
	_, supported := map[string]struct{}{
		"PreToolUse": {}, "PostToolUse": {}, "PostToolUseFailure": {}, "PermissionRequest": {}, "PermissionDenied": {},
		"SessionStart": {}, "Setup": {}, "SessionEnd": {}, "Notification": {}, "SubagentStart": {}, "SubagentStop": {},
		"PreCompact": {}, "PostCompact": {}, "PreModelSwitch": {}, "PostModelSwitch": {}, "ConfigChange": {},
		"DirectoryAdded": {}, "FileChanged": {}, "StopFailure": {}, "InstructionsLoaded": {}, "UserPromptExpansion": {},
		"Elicitation": {}, "ElicitationResult": {},
	}[event]
	return supported
}

func claudeHookEventSupportsIf(event string) bool {
	_, supported := map[string]struct{}{
		"PreToolUse": {}, "PostToolUse": {}, "PostToolUseFailure": {}, "PermissionRequest": {}, "PermissionDenied": {},
	}[event]
	return supported
}

func validateNativeHookHandler(packageDir, runtimeName, event string, handler map[string]json.RawMessage) (bool, string, error) {
	var handlerType string
	if rawType, ok := handler["type"]; !ok || json.Unmarshal(rawType, &handlerType) != nil || strings.TrimSpace(handlerType) == "" {
		return false, "", fmt.Errorf("hook event %s handler has no valid type", event)
	}
	if handlerType != "command" {
		return validateNativeNonCommandHookHandler(runtimeName, event, handlerType, handler)
	}
	allowedFields := map[string]struct{}{"type": {}, "command": {}, "timeout": {}, "statusMessage": {}, "async": {}}
	if runtimeName == "claude" {
		allowedFields["if"] = struct{}{}
		allowedFields["args"] = struct{}{}
		allowedFields["asyncRewake"] = struct{}{}
		allowedFields["shell"] = struct{}{}
		if _, hasOnce := handler["once"]; hasOnce {
			return false, "the Claude runtime ignores once in plugin hook documents; it is only honored in skill frontmatter", nil
		}
		if _, hasIf := handler["if"]; hasIf && !claudeHookEventSupportsIf(event) {
			return false, fmt.Sprintf("hook event %s never evaluates if fields in Claude plugin hook documents", event), nil
		}
		if raw, ok := handler["args"]; ok {
			var value []string
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil {
				return false, "", fmt.Errorf("hook event %s command handler args must be an array of strings", event)
			}
		}
		if raw, ok := handler["asyncRewake"]; ok {
			var value bool
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil {
				return false, "", fmt.Errorf("hook event %s command handler asyncRewake must be a boolean", event)
			}
		}
		if raw, ok := handler["shell"]; ok {
			var value string
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil {
				return false, "", fmt.Errorf("hook event %s command handler shell must be a string", event)
			}
		}
	} else {
		allowedFields["commandWindows"] = struct{}{}
		allowedFields["additionalContextLimit"] = struct{}{}
	}
	for key := range handler {
		if _, ok := allowedFields[key]; !ok {
			return false, fmt.Sprintf("command hook field %s is not supported by the %s compiler", key, runtimeName), nil
		}
	}
	type commandHook struct {
		Type                   string   `json:"type"`
		Command                string   `json:"command"`
		CommandWindows         string   `json:"commandWindows,omitempty"`
		Timeout                float64  `json:"timeout,omitempty"`
		StatusMessage          string   `json:"statusMessage,omitempty"`
		Async                  bool     `json:"async,omitempty"`
		AdditionalContextLimit int      `json:"additionalContextLimit,omitempty"`
		If                     string   `json:"if,omitempty"`
		Once                   bool     `json:"once,omitempty"`
		Args                   []string `json:"args,omitempty"`
		AsyncRewake            bool     `json:"asyncRewake,omitempty"`
		Shell                  string   `json:"shell,omitempty"`
	}
	rawHandler, err := json.Marshal(handler)
	if err != nil {
		return false, "", fmt.Errorf("encode hook event %s handler: %w", event, err)
	}
	var commandHandler commandHook
	if err := decodeStrictJSON(rawHandler, &commandHandler); err != nil {
		return false, "", fmt.Errorf("hook event %s command handler has invalid field types: %w", event, err)
	}
	if strings.TrimSpace(commandHandler.Command) == "" {
		return false, "", fmt.Errorf("hook event %s command handler has no non-empty command", event)
	}
	if commandHandler.Timeout < 0 || commandHandler.AdditionalContextLimit < 0 {
		return false, "", fmt.Errorf("hook event %s command handler has a negative resource limit", event)
	}
	if runtimeName == "claude" && commandHandler.Shell != "" && commandHandler.Shell != "bash" && commandHandler.Shell != "powershell" {
		return false, "", fmt.Errorf("hook event %s command handler has unsupported shell %q", event, commandHandler.Shell)
	}
	if runtimeName == "claude" {
		_, hasArgs := handler["args"]
		_, hasShell := handler["shell"]
		if hasArgs && hasShell {
			return false, "the Claude runtime ignores shell when args selects exec-form command execution", nil
		}
		if commandHandler.Async {
			if _, hasTimeout := handler["timeout"]; hasTimeout {
				return false, "the Claude runtime does not enforce timeout on async command hooks", nil
			}
		}
	}
	if err := validateHookCommandContainment(packageDir, commandHandler.Command); err != nil {
		return false, "", fmt.Errorf("hook event %s command: %w", event, err)
	}
	if runtimeName == "claude" {
		for index, argument := range commandHandler.Args {
			if err := validateHookCommandContainment(packageDir, argument); err != nil {
				return false, "", fmt.Errorf("hook event %s argument %d: %w", event, index, err)
			}
		}
	}
	if commandHandler.CommandWindows != "" {
		if err := validateHookCommandContainment(packageDir, commandHandler.CommandWindows); err != nil {
			return false, "", fmt.Errorf("hook event %s Windows command: %w", event, err)
		}
	}
	return true, "", nil
}

func validateNativeNonCommandHookHandler(runtimeName, event, handlerType string, handler map[string]json.RawMessage) (bool, string, error) {
	if runtimeName == "codex" {
		if handlerType != "mcp_tool" {
			return false, fmt.Sprintf("hook handler type %s is not compiled as enforceable for %s", handlerType, runtimeName), nil
		}
		if event == "SessionEnd" {
			return false, "the Codex runtime does not execute mcp_tool handlers for SessionEnd", nil
		}
		return validateNativeMCPToolHookHandler(runtimeName, event, handler, false)
	}

	if _, hasOnce := handler["once"]; hasOnce {
		return false, "the Claude runtime ignores once in plugin hook documents; it is only honored in skill frontmatter", nil
	}
	if _, hasIf := handler["if"]; hasIf && !claudeHookEventSupportsIf(event) {
		return false, fmt.Sprintf("hook event %s never evaluates if fields in Claude plugin hook documents", event), nil
	}
	if !claudeHookEventSupportsHandler(event, handlerType) {
		return false, fmt.Sprintf("hook event %s does not execute %s handlers in Claude plugin hook documents", event, handlerType), nil
	}

	switch handlerType {
	case "http":
		return validateNativeClaudeHTTPHookHandler(event, handler)
	case "mcp_tool":
		return validateNativeMCPToolHookHandler(runtimeName, event, handler, true)
	case "prompt", "agent":
		return validateNativeClaudePromptHookHandler(event, handlerType, handler)
	default:
		return false, fmt.Sprintf("hook handler type %s is not compiled as enforceable for %s", handlerType, runtimeName), nil
	}
}

func claudeHookEventSupportsHandler(event, handlerType string) bool {
	switch handlerType {
	case "command":
		return true
	case "mcp_tool":
		// Setup always fires before the MCP client exists, so the runtime skips
		// these handlers even though it accepts their configuration.
		return event != "Setup"
	case "http":
		return event != "SessionStart" && event != "Setup"
	case "prompt", "agent":
		_, supported := map[string]struct{}{
			"PermissionDenied": {}, "PermissionRequest": {}, "PostToolBatch": {}, "PostToolUse": {}, "PostToolUseFailure": {},
			"PreToolUse": {}, "Stop": {}, "SubagentStop": {}, "TaskCompleted": {}, "TaskCreated": {}, "TeammateIdle": {},
			"UserPromptExpansion": {}, "UserPromptSubmit": {},
		}[event]
		return supported
	default:
		return false
	}
}

func validateNativeMCPToolHookHandler(runtimeName, event string, handler map[string]json.RawMessage, allowIf bool) (bool, string, error) {
	allowedFields := map[string]struct{}{
		"type": {}, "server": {}, "tool": {}, "input": {}, "timeout": {}, "statusMessage": {},
	}
	if allowIf {
		allowedFields["if"] = struct{}{}
	}
	if native, reason := validateNativeHookFields(runtimeName, "mcp_tool", handler, allowedFields); !native {
		return false, reason, nil
	}
	if err := validateNativeHookCommonFieldTypes(event, handler, allowIf); err != nil {
		return false, "", err
	}
	for _, field := range []string{"server", "tool"} {
		if err := validateRequiredHookString(event, "mcp_tool", field, handler[field]); err != nil {
			return false, "", err
		}
	}
	if raw, ok := handler["input"]; ok {
		var input map[string]any
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &input) != nil {
			return false, "", fmt.Errorf("hook event %s mcp_tool handler input must be an object", event)
		}
	}
	return true, "", nil
}

func validateNativeClaudeHTTPHookHandler(event string, handler map[string]json.RawMessage) (bool, string, error) {
	allowedFields := map[string]struct{}{
		"type": {}, "url": {}, "headers": {}, "allowedEnvVars": {}, "timeout": {}, "statusMessage": {}, "if": {},
	}
	if native, reason := validateNativeHookFields("claude", "http", handler, allowedFields); !native {
		return false, reason, nil
	}
	if err := validateNativeHookCommonFieldTypes(event, handler, true); err != nil {
		return false, "", err
	}
	if err := validateRequiredHookString(event, "http", "url", handler["url"]); err != nil {
		return false, "", err
	}
	var rawURL string
	_ = json.Unmarshal(handler["url"], &rawURL)
	endpoint, err := url.Parse(rawURL)
	if err != nil || !endpoint.IsAbs() || endpoint.Host == "" || (!strings.EqualFold(endpoint.Scheme, "http") && !strings.EqualFold(endpoint.Scheme, "https")) || endpoint.User != nil {
		return false, "", fmt.Errorf("hook event %s http handler url must be an absolute HTTP or HTTPS URL without user information", event)
	}
	if claudeHTTPHookURLContainsCredential(rawURL, endpoint) {
		return false, "", fmt.Errorf("hook event %s http handler url must not contain literal credentials", event)
	}
	if strings.EqualFold(endpoint.Scheme, "http") && !isPortableMCPLoopbackHost(endpoint.Hostname()) {
		return false, "", fmt.Errorf("hook event %s http handler must use HTTPS outside localhost or a loopback IP", event)
	}
	allowedVariables := make(map[string]struct{})
	if raw, ok := handler["allowedEnvVars"]; ok {
		var variables []string
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &variables) != nil {
			return false, "", fmt.Errorf("hook event %s http handler allowedEnvVars must be an array of strings", event)
		}
		for _, variable := range variables {
			if !isEnvironmentVariableName(variable) {
				return false, "", fmt.Errorf("hook event %s http handler contains an invalid allowed environment variable", event)
			}
			allowedVariables[variable] = struct{}{}
		}
	}
	if raw, ok := handler["headers"]; ok {
		var headers map[string]string
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &headers) != nil {
			return false, "", fmt.Errorf("hook event %s http handler headers must be an object of strings", event)
		}
		if err := validateClaudeHTTPHookHeaders(headers, allowedVariables); err != nil {
			return false, "", fmt.Errorf("hook event %s http handler: %w", event, err)
		}
	}
	return true, "", nil
}

func claudeHTTPHookURLContainsCredential(rawURL string, endpoint *url.URL) bool {
	for _, value := range []string{rawURL, endpoint.Host, endpoint.Path, endpoint.RawPath, endpoint.RawQuery, endpoint.Fragment, endpoint.RawFragment} {
		if containsCredentialAfterURLDecoding(value) {
			return true
		}
	}
	for name, values := range endpoint.Query() {
		if containsCredentialAfterURLDecoding(name) {
			return true
		}
		for _, value := range values {
			if containsCredentialAfterURLDecoding(value) || containsCredentialAfterURLDecoding(name+"="+value) {
				return true
			}
		}
	}
	return false
}

func containsCredentialAfterURLDecoding(value string) bool {
	for decodeCount := 0; decodeCount < 4; decodeCount++ {
		if registry.ContainsCredentialShapedValue(value) {
			return true
		}
		decoded, err := url.QueryUnescape(value)
		if err != nil || decoded == value {
			return false
		}
		value = decoded
	}
	return registry.ContainsCredentialShapedValue(value)
}

func validateClaudeHTTPHookHeaders(headers map[string]string, allowedVariables map[string]struct{}) error {
	for name, value := range headers {
		if !isHTTPHeaderName(name) || !isHTTPHeaderValue(value) {
			return fmt.Errorf("contains an invalid header")
		}
		if registry.ContainsCredentialShapedValue(name) {
			return fmt.Errorf("headers must not contain literal credentials; use allowedEnvVars references")
		}
		references := hookEnvironmentReference.FindAllStringSubmatch(value, -1)
		for _, reference := range references {
			variable := reference[1]
			if variable == "" {
				variable = reference[2]
			}
			if _, allowed := allowedVariables[variable]; !allowed {
				return fmt.Errorf("header references environment variable %s without declaring it in allowedEnvVars", variable)
			}
		}
		literal := hookEnvironmentReference.ReplaceAllString(value, "")
		if registry.ContainsCredentialShapedValue(literal) {
			return fmt.Errorf("headers must not contain literal credentials; use allowedEnvVars references")
		}
		if isCredentialBearingHTTPHeaderName(name) {
			if len(references) == 0 {
				return fmt.Errorf("headers must not contain literal credentials; use allowedEnvVars references")
			}
			remaining := strings.TrimSpace(literal)
			if remaining != "" && !hookAuthorizationSchemePrefix.MatchString(remaining) {
				return fmt.Errorf("credential-bearing headers may contain only an authorization scheme and allowedEnvVars references")
			}
		}
	}
	return nil
}

func validateNativeClaudePromptHookHandler(event, handlerType string, handler map[string]json.RawMessage) (bool, string, error) {
	allowedFields := map[string]struct{}{
		"type": {}, "prompt": {}, "model": {}, "timeout": {}, "statusMessage": {}, "if": {},
	}
	if native, reason := validateNativeHookFields("claude", handlerType, handler, allowedFields); !native {
		return false, reason, nil
	}
	if err := validateNativeHookCommonFieldTypes(event, handler, true); err != nil {
		return false, "", err
	}
	if err := validateRequiredHookString(event, handlerType, "prompt", handler["prompt"]); err != nil {
		return false, "", err
	}
	if raw, ok := handler["model"]; ok {
		if err := validateRequiredHookString(event, handlerType, "model", raw); err != nil {
			return false, "", err
		}
	}
	return true, "", nil
}

func validateNativeHookFields(runtimeName, handlerType string, handler map[string]json.RawMessage, allowed map[string]struct{}) (bool, string) {
	for field := range handler {
		if _, ok := allowed[field]; !ok {
			return false, fmt.Sprintf("%s hook field %s is not supported by the %s compiler", handlerType, field, runtimeName)
		}
	}
	return true, ""
}

func validateNativeHookCommonFieldTypes(event string, handler map[string]json.RawMessage, allowIf bool) error {
	if raw, ok := handler["timeout"]; ok {
		var timeout float64
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &timeout) != nil || timeout < 0 {
			return fmt.Errorf("hook event %s handler timeout must be a non-negative number", event)
		}
	}
	if raw, ok := handler["statusMessage"]; ok {
		var message string
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &message) != nil {
			return fmt.Errorf("hook event %s handler statusMessage must be a string", event)
		}
	}
	if raw, ok := handler["if"]; ok {
		var condition string
		if !allowIf || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &condition) != nil || strings.TrimSpace(condition) == "" {
			return fmt.Errorf("hook event %s handler if must be a non-empty string", event)
		}
	}
	return nil
}

func validateRequiredHookString(event, handlerType, field string, raw json.RawMessage) error {
	var value string
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" {
		return fmt.Errorf("hook event %s %s handler %s must be a non-empty string", event, handlerType, field)
	}
	return nil
}

func isEnvironmentVariableName(value string) bool {
	if value == "" || !((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z') || value[0] == '_') {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if !((character >= 'A' && character <= 'Z') || (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_') {
			return false
		}
	}
	return true
}

func validateHookCommandContainment(packageDir, command string) error {
	if strings.Contains(command, "../") || strings.Contains(command, `..\`) || strings.ContainsRune(command, '\x00') {
		return fmt.Errorf("contains a path escape")
	}
	if unrootedHookPathPattern.MatchString(command) {
		return fmt.Errorf("uses an unrooted relative path; bundled files must be referenced through PLUGIN_ROOT")
	}
	for _, match := range hookPluginPathPattern.FindAllStringSubmatch(command, -1) {
		relative := match[1]
		if err := validateCompiledRelativePath("./" + relative); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(packageDir, filepath.FromSlash(relative))); err != nil {
			return fmt.Errorf("references unavailable plugin path %s", relative)
		}
	}
	return nil
}

func writeNativePluginManifest(packageDir, runtimeName string, manifest registry.Manifest, compiled compiledRuntimePackage) (string, error) {
	nativeHooks := make([]string, 0)
	for _, hook := range compiled.Registrations.Hooks {
		if hook.Enforcement == "native" {
			nativeHooks = append(nativeHooks, hook.Path)
		}
	}
	if runtimeName == "codex" {
		portable := portablePluginManifest{
			Schema: agentPluginManifestSchema,
			Name:   manifest.ID[strings.LastIndex(manifest.ID, "/")+1:], Version: manifest.Version,
			Description: manifest.Description,
		}
		if len(nativeHooks) > 0 {
			var hookValue any = nativeHooks
			if len(nativeHooks) == 1 {
				hookValue = nativeHooks[0]
			}
			portable.Extensions = map[string]map[string]any{"com.openai": {"hooks": hookValue}}
		}
		manifestPath := filepath.Join(packageDir, "plugin.json")
		if err := writeDeterministicJSON(manifestPath, portable); err != nil {
			return "", err
		}
		return manifestPath, nil
	}

	native := nativePluginManifest{
		Name: manifest.ID[strings.LastIndex(manifest.ID, "/")+1:], Version: manifest.Version,
		Description: manifest.Description,
	}
	if len(compiled.Registrations.Skills) > 0 {
		native.Skills = "./skills/"
	}
	if runtimeName == "claude" && len(compiled.Registrations.Agents) > 0 {
		native.Agents = "./agents/"
	}
	if len(nativeHooks) == 1 {
		native.Hooks = nativeHooks[0]
	} else if len(nativeHooks) > 1 {
		native.Hooks = nativeHooks
	}
	nativeMCP := make([]string, 0)
	for _, server := range compiled.Registrations.MCPServers {
		if server.Enforcement == "native" {
			nativeMCP = append(nativeMCP, server.Path)
		}
	}
	if len(nativeMCP) == 1 {
		native.MCPServers = nativeMCP[0]
	} else if len(nativeMCP) > 1 {
		native.MCPServers = nativeMCP
	}

	runtimeDir := filepath.Join(packageDir, ".claude-plugin")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return "", fmt.Errorf("create native runtime manifest directory: %w", err)
	}
	manifestPath := filepath.Join(runtimeDir, "plugin.json")
	if err := writeDeterministicJSON(manifestPath, native); err != nil {
		return "", err
	}
	return manifestPath, nil
}

func writeDeterministicJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("render runtime artifact %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write runtime artifact %s: %w", path, err)
	}
	return nil
}

// ValidateCompiledRuntimePackage checks that the lock and compiled runtime
// contract have identical component sets and that every native registration
// points at a materialized path. It intentionally rejects legacy three-field
// plugin manifests as incomplete.
func ValidateCompiledRuntimePackage(packageDir, runtimeName string) error {
	runtimeName = strings.ToLower(strings.TrimSpace(runtimeName))
	if runtimeName != "codex" && runtimeName != "claude" && runtimeName != "generic" {
		return fmt.Errorf("unsupported runtime compiler %q", runtimeName)
	}
	contractPath := filepath.Join(packageDir, ".runtime", runtimeName+".json")
	payload, err := os.ReadFile(contractPath)
	if err != nil {
		return fmt.Errorf("read runtime contract: %w", err)
	}
	var compiled compiledRuntimePackage
	if err := json.Unmarshal(payload, &compiled); err != nil {
		return fmt.Errorf("parse runtime contract: %w", err)
	}
	if compiled.SchemaVersion != runtimePackageSchema || compiled.Runtime != runtimeName || compiled.Package.ID == "" || compiled.Package.Version == "" {
		return fmt.Errorf("runtime contract identity is incomplete")
	}
	if compiled.SourceLock != "dependencies.lock.json" {
		return fmt.Errorf("runtime contract source lock must be dependencies.lock.json")
	}
	lockPayload, err := os.ReadFile(filepath.Join(packageDir, filepath.FromSlash(compiled.SourceLock)))
	if err != nil {
		return fmt.Errorf("read runtime contract lock: %w", err)
	}
	var lock runtimeDependencyLock
	if err := json.Unmarshal(lockPayload, &lock); err != nil {
		return fmt.Errorf("parse runtime contract lock: %w", err)
	}
	if lock.Root.Module != "plugins" || lock.Root.ID != compiled.Package.ID || lock.Root.Version != compiled.Package.Version {
		return fmt.Errorf("runtime contract package identity does not match source lock root")
	}

	locked := make(map[string]runtimeDependencyLockComponent, len(lock.Components))
	for _, component := range lock.Components {
		key := component.Module + ":" + component.ID
		if _, exists := locked[key]; exists {
			return fmt.Errorf("runtime contract source lock repeats component %s", key)
		}
		locked[key] = component
	}
	seen := make(map[string]struct{}, len(lock.Components))
	groups := []struct {
		module        string
		registrations []compiledRuntimeRegistration
	}{
		{module: "skills", registrations: compiled.Registrations.Skills},
		{module: "agents", registrations: compiled.Registrations.Agents},
		{module: "tools-mcp", registrations: compiled.Registrations.MCPServers},
	}
	for _, group := range groups {
		for _, registration := range group.registrations {
			key := group.module + ":" + registration.ID
			component, exists := locked[key]
			if !exists {
				return fmt.Errorf("compiled %s registration %s is not present in the source lock", group.module, registration.ID)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("compiled runtime repeats locked component %s", key)
			}
			if registration.Version != component.Version {
				return fmt.Errorf("compiled registration %s version %q does not match locked version %q", registration.ID, registration.Version, component.Version)
			}
			if group.module == "tools-mcp" && registration.Enforcement == "native" && len(registration.Names) == 0 {
				return fmt.Errorf("native MCP registration %s has no declared server names", registration.ID)
			}
			if group.module != "tools-mcp" && len(registration.Names) != 0 {
				return fmt.Errorf("compiled %s registration %s declares MCP server names", group.module, registration.ID)
			}
			if err := validateCompiledRegistration(packageDir, registration); err != nil {
				return err
			}
			seen[key] = struct{}{}
		}
	}
	if len(seen) != len(lock.Components) {
		return fmt.Errorf("compiled component count %d does not match locked component count %d", len(seen), len(lock.Components))
	}
	for key := range locked {
		if _, ok := seen[key]; !ok {
			return fmt.Errorf("compiled runtime omits locked component %s", key)
		}
	}

	auxiliary := append([]compiledRuntimeRegistration{}, compiled.Registrations.Config...)
	auxiliary = append(auxiliary, compiled.Registrations.Hooks...)
	auxiliarySeen := make(map[string]struct{}, len(auxiliary))
	for _, registration := range auxiliary {
		key := registration.ID + "\x00" + registration.Path
		if _, duplicate := auxiliarySeen[key]; duplicate {
			return fmt.Errorf("compiled runtime repeats auxiliary registration %s", registration.ID)
		}
		if err := validateCompiledRegistration(packageDir, registration); err != nil {
			return err
		}
		auxiliarySeen[key] = struct{}{}
	}
	if runtimeName != "generic" {
		nativePath := filepath.Join(packageDir, ".claude-plugin", "plugin.json")
		if runtimeName == "codex" {
			nativePath = filepath.Join(packageDir, "plugin.json")
		}
		payload, err := os.ReadFile(nativePath)
		if err != nil {
			return fmt.Errorf("read native manifest: %w", err)
		}
		if runtimeName == "codex" {
			var native portablePluginManifest
			if err := json.Unmarshal(payload, &native); err != nil || native.Schema != agentPluginManifestSchema || native.Name == "" || native.Version == "" {
				return fmt.Errorf("Codex portable manifest is incomplete")
			}
			if native.Version != compiled.Package.Version {
				return fmt.Errorf("native manifest version does not match runtime contract")
			}
		} else {
			var native nativePluginManifest
			if err := json.Unmarshal(payload, &native); err != nil || native.Name == "" || native.Version == "" {
				return fmt.Errorf("native manifest is incomplete")
			}
			if native.Version != compiled.Package.Version {
				return fmt.Errorf("native manifest version does not match runtime contract")
			}
		}
	}
	if err := validateFinalRuntimeDiscovery(packageDir, runtimeName, compiled); err != nil {
		return fmt.Errorf("runtime discovery layout: %w", err)
	}
	return nil
}

func validateCompiledRegistration(packageDir string, registration compiledRuntimeRegistration) error {
	if registration.ID == "" || registration.Path == "" || (registration.Enforcement != "native" && registration.Enforcement != "advisory") {
		return fmt.Errorf("compiled registration is incomplete")
	}
	if registration.Enforcement == "advisory" && registration.Reason == "" {
		return fmt.Errorf("advisory registration %s has no reason", registration.ID)
	}
	if err := validateCompiledRelativePath(registration.Path); err != nil {
		return fmt.Errorf("compiled registration %s: %w", registration.ID, err)
	}
	path := filepath.Join(packageDir, filepath.FromSlash(strings.TrimPrefix(registration.Path, "./")))
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("compiled registration %s points at unavailable path %s", registration.ID, registration.Path)
	}
	return nil
}

func validateCompiledRelativePath(value string) error {
	if !strings.HasPrefix(value, "./") || strings.Contains(value, "\\") {
		return fmt.Errorf("path %q must be a ./-prefixed portable path", value)
	}
	trimmed := strings.TrimPrefix(value, "./")
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(trimmed)))
	if trimmed == "" || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != trimmed {
		return fmt.Errorf("path %q escapes or is not canonical", value)
	}
	return nil
}
