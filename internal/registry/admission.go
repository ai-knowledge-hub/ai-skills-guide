package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	instructionEvidenceLifetime = 180 * 24 * time.Hour
	executableEvidenceLifetime  = 90 * 24 * time.Hour
	authEvidenceLifetime        = 30 * 24 * time.Hour
)

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var packageIDPattern = regexp.MustCompile(`^[a-z0-9-]+/[a-z0-9-]+$`)

var authenticationDriverFlows = map[string]string{
	"api-key": "credential-binding", "bearer-token": "credential-binding",
	"oauth-authorization-code-pkce": "interactive-browser", "oauth-device-flow": "device-code",
	"oauth-client-credentials": "non-interactive-service", "service-account": "non-interactive-service",
	"workload-identity": "non-interactive-workload", "brokered": "brokered", "custom": "custom",
}

type authenticationDriverDocument struct {
	SchemaVersion  string                              `json:"schema_version"`
	CredentialMode string                              `json:"credential_mode"`
	Method         string                              `json:"method"`
	Flow           string                              `json:"flow"`
	Runtimes       []string                            `json:"runtimes"`
	Bootstrap      authenticationDriverCommandDocument `json:"bootstrap"`
	Credential     authenticationDriverCommandDocument `json:"credential"`
	Status         authenticationDriverCommandDocument `json:"status"`
}

type authenticationDriverCommandDocument struct {
	Command []string `json:"command"`
}

type moduleDefinition struct {
	directory string
	manifest  string
	kind      string
}

var moduleDefinitions = []moduleDefinition{
	{directory: "skills", manifest: "skill.yaml", kind: "skill"},
	{directory: "agents", manifest: "agent.yaml", kind: "agent"},
	{directory: "tools-mcp", manifest: "tool.yaml", kind: "tool"},
	{directory: "plugins", manifest: "plugin.yaml", kind: "plugin"},
}

type manifestRecord struct {
	kind     string
	path     string
	manifest Manifest
}

// ValidateRepository performs registry-admission checks that need repository
// context. Schema validation remains a separate CI gate; this function proves
// package-local paths and the internal dependency graph.
func ValidateRepository(root string) (int, error) {
	records := make([]manifestRecord, 0)
	now := time.Now().UTC()
	for _, definition := range moduleDefinitions {
		paths, err := findManifests(filepath.Join(root, definition.directory), definition.manifest)
		if err != nil {
			return 0, err
		}
		for _, path := range paths {
			manifest, err := ParseManifest(path)
			if err != nil {
				return 0, err
			}
			if err := validatePackage(path, manifest, now, true, false); err != nil {
				return 0, err
			}
			records = append(records, manifestRecord{kind: definition.kind, path: path, manifest: manifest})
		}
	}
	if err := validateDependencyGraph(root, records); err != nil {
		return 0, err
	}
	return len(records), nil
}

// ValidatePackageManifest applies the semantic and package-layout admission
// checks used by registry generation to one extracted package.
func ValidatePackageManifest(manifestPath string) (Manifest, error) {
	return validatePackageManifest(manifestPath, time.Now().UTC(), true)
}

// ValidateHistoricalPackageManifest verifies immutable package history against
// its canonical schema and package layout without treating elapsed wall-clock
// time as corruption. Consumers must still use ValidatePackageManifest when a
// selected release is installed so readiness evidence is current at use time.
func ValidateHistoricalPackageManifest(manifestPath string) (Manifest, error) {
	return validatePackageManifest(manifestPath, time.Time{}, false)
}

func validatePackageManifest(manifestPath string, now time.Time, requireCurrentEvidence bool) (Manifest, error) {
	if err := validateManifestSchema(manifestPath); err != nil {
		return Manifest{}, err
	}
	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		return Manifest{}, err
	}
	if err := validatePackage(manifestPath, manifest, now, requireCurrentEvidence, true); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validatePackage(manifestPath string, manifest Manifest, now time.Time, requireCurrentEvidence, requireArchiveClosure bool) error {
	if !validSemver(manifest.Version) {
		return fmt.Errorf("manifest %s has invalid semantic version %q", manifestPath, manifest.Version)
	}
	if !manifest.usabilitySet || manifest.Usability.Availability == "" || manifest.Usability.Execution == "" {
		return fmt.Errorf("manifest %s must explicitly declare usability.availability and usability.execution", manifestPath)
	}
	packageDir := filepath.Dir(manifestPath)
	keys := make([]string, 0, len(manifest.Entrypoints))
	for key := range manifest.Entrypoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		wantDirectory := strings.HasSuffix(key, "_dir")
		if err := validateContainedPath(packageDir, manifest.Entrypoints[key], wantDirectory); err != nil {
			return fmt.Errorf("manifest %s entrypoint %s: %w", manifestPath, key, err)
		}
	}
	for _, helper := range manifest.Usability.ExecutableHelpers {
		if err := validateContainedPath(packageDir, helper.Entrypoint, false); err != nil {
			return fmt.Errorf("manifest %s executable helper %s: %w", manifestPath, helper.Entrypoint, err)
		}
	}

	if strings.HasPrefix(manifest.SchemaVersion, "2.") {
		if filepath.Base(manifestPath) == "plugin.yaml" && manifest.Execution.Kind == "bundle" && !manifest.Artifact.SelfContained {
			return fmt.Errorf("manifest %s bundle artifact must declare a self-contained dependency closure", manifestPath)
		}
		if requireArchiveClosure && filepath.Base(manifestPath) == "plugin.yaml" && manifest.Artifact.SelfContained {
			if err := validatePluginArchiveClosure(packageDir, manifestPath, manifest, now, requireCurrentEvidence); err != nil {
				return err
			}
		}
		artifactPaths := []struct {
			label string
			path  *string
		}{
			{label: "artifact.dependency_lock", path: manifest.Artifact.DependencyLock},
			{label: "artifact.checksums", path: manifest.Artifact.Checksums},
			{label: "artifact.sbom", path: manifest.Artifact.SBOM},
			{label: "artifact.provenance", path: manifest.Artifact.Provenance},
		}
		for _, artifactPath := range artifactPaths {
			if artifactPath.path != nil {
				if err := validateContainedPath(packageDir, *artifactPath.path, false); err != nil {
					return fmt.Errorf("manifest %s %s: %w", manifestPath, artifactPath.label, err)
				}
			}
		}
		if err := validateExecutionPaths(packageDir, manifestPath, manifest.Execution); err != nil {
			return err
		}
		if err := validateAuthenticationDrivers(packageDir, manifestPath, manifest.Authentication, manifest.Runtimes); err != nil {
			return err
		}
		if authenticationIsRequired(manifest) && manifest.Authentication.Status == "none" {
			return fmt.Errorf("manifest %s requires authentication but declares authentication.status none", manifestPath)
		}
		if requireCurrentEvidence && manifest.Usability.Availability == "usable-now" {
			if err := validateFreshEvidence(manifestPath, manifest, now); err != nil {
				return err
			}
		}
	}

	// Legacy compatibility: an explicitly working local or remote executable
	// must at least point at packaged implementation files. Evidence-backed
	// readiness promotion itself is reserved for v2 manifests.
	if manifest.Usability.Availability == "usable-now" {
		switch manifest.Usability.Execution {
		case "local-tool", "remote-integration":
			if _, ok := manifest.Entrypoints["scripts_dir"]; !ok {
				return fmt.Errorf("manifest %s claims usable-now %s without a scripts_dir entrypoint", manifestPath, manifest.Usability.Execution)
			}
		}
	}
	return nil
}

func validateAuthenticationDrivers(packageDir, manifestPath string, authentication AuthenticationMetadata, runtimes []string) error {
	if authentication.Status == "none" {
		return nil
	}
	for _, method := range authentication.Methods {
		if method == "none" {
			continue
		}
		driverPath := filepath.Join(packageDir, "auth", method+".json")
		_, statErr := os.Stat(driverPath)
		required := method != "api-key" && method != "bearer-token"
		if os.IsNotExist(statErr) && !required {
			continue
		}
		if statErr != nil {
			return fmt.Errorf("manifest %s authentication method %s requires a packaged driver: %w", manifestPath, method, statErr)
		}
		if err := validateAuthDriverSchema(driverPath); err != nil {
			return err
		}
		payload, err := os.ReadFile(driverPath)
		if err != nil {
			return err
		}
		var driver authenticationDriverDocument
		if err := json.Unmarshal(payload, &driver); err != nil {
			return fmt.Errorf("decode authentication driver %s: %w", driverPath, err)
		}
		if driver.SchemaVersion != "skills-hub.auth-driver/v1" || driver.Method != method || driver.Flow != authenticationDriverFlows[method] {
			return fmt.Errorf("authentication driver %s does not match method %s and its required flow", driverPath, method)
		}
		if driver.CredentialMode != "" && driver.CredentialMode != "driver" && driver.CredentialMode != "external-env" {
			return fmt.Errorf("authentication driver %s has an unsupported credential mode", driverPath)
		}
		for _, runtimeName := range runtimes {
			if !containsString(driver.Runtimes, runtimeName) {
				return fmt.Errorf("authentication driver %s does not support declared runtime %s", driverPath, runtimeName)
			}
		}
		commands := []struct {
			name    string
			command []string
		}{{"bootstrap", driver.Bootstrap.Command}, {"credential", driver.Credential.Command}, {"status", driver.Status.Command}}
		for _, declared := range commands {
			if err := validateContainedPath(packageDir, declared.command[0], false); err != nil {
				return fmt.Errorf("authentication driver %s %s command: %w", driverPath, declared.name, err)
			}
			info, err := os.Stat(filepath.Join(packageDir, filepath.FromSlash(declared.command[0])))
			if err != nil || info.Mode()&0o111 == 0 {
				return fmt.Errorf("authentication driver %s %s command is not executable", driverPath, declared.name)
			}
			for _, argument := range declared.command[1:] {
				if strings.ContainsRune(argument, '\x00') || ContainsCredentialShapedValue(argument) {
					return fmt.Errorf("authentication driver %s %s command contains unsafe material", driverPath, declared.name)
				}
			}
		}
	}
	return nil
}

func validatePluginArchiveClosure(packageDir, manifestPath string, manifest Manifest, now time.Time, requireCurrentEvidence bool) error {
	sets := []struct {
		module       string
		manifestName string
		ids          []string
	}{
		{module: "skills", manifestName: "skill.yaml", ids: manifest.Includes.Skills},
		{module: "agents", manifestName: "agent.yaml", ids: manifest.Includes.Agents},
		{module: "tools-mcp", manifestName: "tool.yaml", ids: manifest.Includes.Tools},
	}
	bundledRoot := filepath.Join(packageDir, "bundled")
	if entries, err := os.ReadDir(bundledRoot); err == nil {
		allowedModules := map[string]struct{}{"skills": {}, "agents": {}, "tools-mcp": {}}
		for _, entry := range entries {
			if _, ok := allowedModules[entry.Name()]; !ok || !entry.IsDir() {
				return fmt.Errorf("manifest %s self-contained closure contains unexpected bundled member %s", manifestPath, entry.Name())
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("manifest %s inspect self-contained closure: %w", manifestPath, err)
	}
	for _, set := range sets {
		expected := make(map[string]struct{}, len(set.ids))
		for _, id := range set.ids {
			if !packageIDPattern.MatchString(id) {
				return fmt.Errorf("manifest %s self-contained %s dependency has invalid id %q", manifestPath, set.module, id)
			}
			expected[id] = struct{}{}
			dependencyManifest := filepath.Join(bundledRoot, set.module, filepath.FromSlash(id), set.manifestName)
			dependency, err := validatePackageManifest(dependencyManifest, now, requireCurrentEvidence)
			if err != nil {
				return fmt.Errorf("manifest %s self-contained dependency %s: %w", manifestPath, id, err)
			}
			if dependency.ID != id {
				return fmt.Errorf("manifest %s self-contained dependency path %s contains id %s", manifestPath, id, dependency.ID)
			}
		}
		moduleRoot := filepath.Join(bundledRoot, set.module)
		metadataLocked := manifest.Artifact.DependencyLock != nil && manifest.Artifact.Checksums != nil && manifest.Artifact.SBOM != nil && manifest.Artifact.Provenance != nil
		if !metadataLocked {
			if err := rejectUndeclaredBundledPackages(moduleRoot, expected); err != nil {
				return fmt.Errorf("manifest %s self-contained closure: %w", manifestPath, err)
			}
		}
	}
	for _, hook := range manifest.Includes.Hooks {
		found := false
		for _, extension := range []string{".md", ".json", ".yaml"} {
			candidate := filepath.Join(packageDir, "hooks", hook+extension)
			info, err := os.Lstat(candidate)
			if err == nil && info.Mode().IsRegular() {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("manifest %s self-contained closure is missing hook %s", manifestPath, hook)
		}
	}
	if manifest.SchemaVersion == "2.1" || manifest.Artifact.DependencyLock != nil || manifest.Artifact.Checksums != nil || manifest.Artifact.SBOM != nil || manifest.Artifact.Provenance != nil {
		if err := validateArtifactMetadata(packageDir, manifestPath, manifest, now, requireCurrentEvidence); err != nil {
			return err
		}
	}
	return nil
}

type artifactLock struct {
	LockVersion string                  `json:"lock_version"`
	Root        artifactLockRoot        `json:"root"`
	Components  []artifactLockComponent `json:"components"`
}

type artifactLockRoot struct {
	Module  string `json:"module"`
	ID      string `json:"id"`
	Version string `json:"version"`
}

type artifactLockComponent struct {
	Module        string   `json:"module"`
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	ContentSHA256 string   `json:"content_sha256"`
	ManifestPath  string   `json:"manifest_path"`
	Dependencies  []string `json:"dependencies"`
}

type cyclonedxSBOM struct {
	BOMFormat   string `json:"bomFormat"`
	SpecVersion string `json:"specVersion"`
	Metadata    struct {
		Component cyclonedxComponent `json:"component"`
	} `json:"metadata"`
	Components   []cyclonedxComponent  `json:"components"`
	Dependencies []cyclonedxDependency `json:"dependencies"`
}

type cyclonedxComponent struct {
	Type    string `json:"type"`
	BOMRef  string `json:"bom-ref"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Hashes  []struct {
		Algorithm string `json:"alg"`
		Content   string `json:"content"`
	} `json:"hashes"`
}

type cyclonedxDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

type artifactProvenance struct {
	ProvenanceVersion string `json:"provenance_version"`
	Subject           struct {
		Module        string `json:"module"`
		ID            string `json:"id"`
		Version       string `json:"version"`
		ClosureSHA256 string `json:"closure_sha256"`
	} `json:"subject"`
	Builder   string `json:"builder"`
	Materials []struct {
		Ref    string `json:"ref"`
		SHA256 string `json:"sha256"`
	} `json:"materials"`
}

func validateArtifactMetadata(packageDir, manifestPath string, manifest Manifest, now time.Time, requireCurrentEvidence bool) error {
	if manifest.Artifact.DependencyLock == nil || manifest.Artifact.Checksums == nil || manifest.Artifact.SBOM == nil || manifest.Artifact.Provenance == nil {
		return fmt.Errorf("manifest %s self-contained artifact metadata must declare dependency_lock, checksums, sbom, and provenance", manifestPath)
	}
	if err := validateArtifactChecksums(packageDir, *manifest.Artifact.Checksums); err != nil {
		return fmt.Errorf("manifest %s artifact checksums: %w", manifestPath, err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(packageDir, filepath.FromSlash(*manifest.Artifact.DependencyLock)))
	if err != nil {
		return fmt.Errorf("manifest %s read artifact dependency lock: %w", manifestPath, err)
	}
	var lock artifactLock
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		return fmt.Errorf("manifest %s parse artifact dependency lock: %w", manifestPath, err)
	}
	if lock.LockVersion != "1.0" || lock.Root.Module != "plugins" || lock.Root.ID != manifest.ID || lock.Root.Version != manifest.Version {
		return fmt.Errorf("manifest %s artifact dependency lock root does not match plugins:%s@%s", manifestPath, manifest.ID, manifest.Version)
	}
	locked := make(map[string]artifactLockComponent, len(lock.Components))
	for _, component := range lock.Components {
		key := component.Module + ":" + component.ID
		if _, exists := locked[key]; exists {
			return fmt.Errorf("manifest %s artifact dependency lock repeats component %s", manifestPath, key)
		}
		manifestName := map[string]string{"skills": "skill.yaml", "agents": "agent.yaml", "tools-mcp": "tool.yaml"}[component.Module]
		wantPath := "bundled/" + component.Module + "/" + component.ID + "/" + manifestName
		if manifestName == "" || component.ManifestPath != wantPath || !packageIDPattern.MatchString(component.ID) || !validSemver(component.Version) {
			return fmt.Errorf("manifest %s artifact dependency lock has invalid component %s", manifestPath, key)
		}
		dependencyManifest, err := validatePackageManifest(filepath.Join(packageDir, filepath.FromSlash(component.ManifestPath)), now, requireCurrentEvidence)
		if err != nil {
			return fmt.Errorf("manifest %s artifact dependency lock component %s: %w", manifestPath, key, err)
		}
		if dependencyManifest.ID != component.ID || dependencyManifest.Version != component.Version {
			return fmt.Errorf("manifest %s artifact dependency lock component %s identity does not match its manifest", manifestPath, key)
		}
		declaredDependencies := dependencyKeys(dependencyManifest)
		lockedDependencies := append([]string(nil), component.Dependencies...)
		sort.Strings(lockedDependencies)
		if strings.Join(declaredDependencies, "\n") != strings.Join(lockedDependencies, "\n") {
			return fmt.Errorf("manifest %s artifact dependency lock component %s dependency edges do not match its manifest", manifestPath, key)
		}
		componentRoot := filepath.Join(packageDir, "bundled", component.Module, filepath.FromSlash(component.ID))
		digest, err := digestPackageTree(componentRoot)
		if err != nil {
			return fmt.Errorf("manifest %s hash artifact dependency lock component %s: %w", manifestPath, key, err)
		}
		if component.ContentSHA256 != digest {
			return fmt.Errorf("manifest %s artifact dependency lock component %s digest mismatch", manifestPath, key)
		}
		locked[key] = component
	}
	if err := validateLockedBundledPackages(packageDir, locked); err != nil {
		return fmt.Errorf("manifest %s self-contained closure: %w", manifestPath, err)
	}

	expected := dependencyRefsForManifest(manifest)
	queue := make([]string, 0, len(expected))
	for _, reference := range expected {
		key := reference.module + ":" + reference.id
		if _, ok := locked[key]; !ok {
			return fmt.Errorf("manifest %s artifact dependency lock omits %s:%s", manifestPath, reference.module, reference.id)
		}
		queue = append(queue, key)
	}
	reachable := make(map[string]struct{}, len(locked))
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, seen := reachable[key]; seen {
			continue
		}
		component, ok := locked[key]
		if !ok {
			return fmt.Errorf("manifest %s artifact dependency lock references missing component %s", manifestPath, key)
		}
		reachable[key] = struct{}{}
		queue = append(queue, component.Dependencies...)
	}
	if len(reachable) != len(locked) {
		return fmt.Errorf("manifest %s artifact dependency lock contains unreachable components", manifestPath)
	}
	state := make(map[string]uint8, len(locked))
	var visit func(string) error
	visit = func(key string) error {
		if state[key] == 1 {
			return fmt.Errorf("artifact dependency lock contains a cycle through %s", key)
		}
		if state[key] == 2 {
			return nil
		}
		state[key] = 1
		for _, dependency := range locked[key].Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[key] = 2
		return nil
	}
	for key := range locked {
		if err := visit(key); err != nil {
			return fmt.Errorf("manifest %s %w", manifestPath, err)
		}
	}
	if err := validateArtifactSBOM(packageDir, *manifest.Artifact.SBOM, manifest, lock); err != nil {
		return fmt.Errorf("manifest %s artifact SBOM: %w", manifestPath, err)
	}
	if err := validateArtifactProvenance(packageDir, manifest, lock); err != nil {
		return fmt.Errorf("manifest %s artifact provenance: %w", manifestPath, err)
	}
	return nil
}

func dependencyKeys(manifest Manifest) []string {
	keys := make([]string, 0)
	add := func(module string, ids []string) {
		for _, id := range ids {
			if strings.Contains(id, "/") {
				keys = append(keys, module+":"+id)
			}
		}
	}
	add("skills", manifest.Dependencies.Skills)
	add("agents", manifest.Dependencies.Agents)
	add("tools-mcp", manifest.Dependencies.Tools)
	sort.Strings(keys)
	return keys
}

func dependencyRefsForManifest(manifest Manifest) []struct{ module, id string } {
	refs := make([]struct{ module, id string }, 0)
	seen := make(map[string]struct{})
	add := func(module string, ids []string) {
		for _, id := range ids {
			if strings.Contains(id, "/") {
				key := module + ":" + id
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					refs = append(refs, struct{ module, id string }{module: module, id: id})
				}
			}
		}
	}
	add("skills", manifest.Dependencies.Skills)
	add("agents", manifest.Dependencies.Agents)
	add("tools-mcp", manifest.Dependencies.Tools)
	add("skills", manifest.Includes.Skills)
	add("agents", manifest.Includes.Agents)
	add("tools-mcp", manifest.Includes.Tools)
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].module+":"+refs[i].id < refs[j].module+":"+refs[j].id
	})
	return refs
}

func validateArtifactChecksums(packageDir, checksumPath string) error {
	payload, err := os.ReadFile(filepath.Join(packageDir, filepath.FromSlash(checksumPath)))
	if err != nil {
		return err
	}
	declared := make(map[string]string)
	for lineNumber, line := range strings.Split(strings.TrimSpace(string(payload)), "\n") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || len(parts[0]) != sha256.Size*2 {
			return fmt.Errorf("line %d is not a SHA-256 checksum record", lineNumber+1)
		}
		if _, err := hex.DecodeString(parts[0]); err != nil {
			return fmt.Errorf("line %d has an invalid SHA-256 digest", lineNumber+1)
		}
		if _, exists := declared[parts[1]]; exists {
			return fmt.Errorf("duplicate checksum path %s", parts[1])
		}
		declared[parts[1]] = parts[0]
	}
	found := 0
	err = filepath.WalkDir(packageDir, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(packageDir, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == checksumPath {
			return nil
		}
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(bytes)
		if declared[relative] != hex.EncodeToString(digest[:]) {
			return fmt.Errorf("missing or incorrect checksum for %s", relative)
		}
		found++
		return nil
	})
	if err != nil {
		return err
	}
	if found != len(declared) {
		return fmt.Errorf("checksum manifest contains paths absent from the archive")
	}
	return nil
}

func digestPackageTree(root string) (string, error) {
	return digestPackageTreeExcluding(root, nil)
}

func digestPackageTreeExcluding(root string, excluded map[string]struct{}) (string, error) {
	lines := make([]string, 0)
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, skip := excluded[relative]; skip {
			return nil
		}
		mode := 0o644
		if info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		digest := sha256.Sum256(bytes)
		lines = append(lines, fmt.Sprintf("%s %o %d %s", hex.EncodeToString(digest[:]), mode, len(bytes), relative))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	digest := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(digest[:]), nil
}

func validateLockedBundledPackages(packageDir string, locked map[string]artifactLockComponent) error {
	for _, module := range []string{"skills", "agents", "tools-mcp"} {
		expected := make(map[string]struct{})
		for _, component := range locked {
			if component.Module == module {
				expected[component.ID] = struct{}{}
			}
		}
		if err := rejectUndeclaredBundledPackages(filepath.Join(packageDir, "bundled", module), expected); err != nil {
			return err
		}
	}
	return nil
}

func validateArtifactSBOM(packageDir, sbomPath string, manifest Manifest, lock artifactLock) error {
	payload, err := os.ReadFile(filepath.Join(packageDir, filepath.FromSlash(sbomPath)))
	if err != nil {
		return err
	}
	var sbom cyclonedxSBOM
	if err := json.Unmarshal(payload, &sbom); err != nil {
		return err
	}
	if sbom.BOMFormat != "CycloneDX" || sbom.SpecVersion != "1.5" || len(sbom.Components) != len(lock.Components) {
		return fmt.Errorf("SBOM does not describe the locked CycloneDX 1.5 component set")
	}
	rootRef := "plugins:" + lock.Root.ID + "@" + lock.Root.Version
	root := sbom.Metadata.Component
	if root.Type != "application" || root.BOMRef != rootRef || root.Name != lock.Root.ID || root.Version != lock.Root.Version {
		return fmt.Errorf("SBOM root component does not match the locked plugin %s", rootRef)
	}
	type componentContract struct {
		version string
		digest  string
	}
	want := make(map[string]componentContract, len(lock.Components))
	versionedRefs := make(map[string]string, len(lock.Components))
	for _, component := range lock.Components {
		ref := component.Module + ":" + component.ID + "@" + component.Version
		want[ref] = componentContract{version: component.Version, digest: component.ContentSHA256}
		versionedRefs[component.Module+":"+component.ID] = ref
	}
	for _, component := range sbom.Components {
		contract, ok := want[component.BOMRef]
		if !ok || component.Version != contract.version {
			return fmt.Errorf("SBOM contains an unlocked component %s", component.BOMRef)
		}
		if len(component.Hashes) != 1 || component.Hashes[0].Algorithm != "SHA-256" || component.Hashes[0].Content != contract.digest {
			return fmt.Errorf("SBOM component %s does not match its locked digest", component.BOMRef)
		}
		delete(want, component.BOMRef)
	}
	if len(want) != 0 {
		return fmt.Errorf("SBOM omits locked components")
	}
	expectedGraph := make(map[string][]string, len(lock.Components)+1)
	rootDependencies := make([]string, 0)
	for _, dependency := range dependencyRefsForManifest(manifest) {
		ref, ok := versionedRefs[dependency.module+":"+dependency.id]
		if !ok {
			return fmt.Errorf("SBOM root references unlocked dependency %s:%s", dependency.module, dependency.id)
		}
		rootDependencies = append(rootDependencies, ref)
	}
	sort.Strings(rootDependencies)
	expectedGraph[rootRef] = rootDependencies
	for _, component := range lock.Components {
		ref := versionedRefs[component.Module+":"+component.ID]
		dependencies := make([]string, 0, len(component.Dependencies))
		for _, dependency := range component.Dependencies {
			dependencyRef, ok := versionedRefs[dependency]
			if !ok {
				return fmt.Errorf("SBOM component %s references unlocked dependency %s", ref, dependency)
			}
			dependencies = append(dependencies, dependencyRef)
		}
		sort.Strings(dependencies)
		expectedGraph[ref] = dependencies
	}
	if len(sbom.Dependencies) != len(expectedGraph) {
		return fmt.Errorf("SBOM dependency graph does not describe the complete locked closure")
	}
	for _, node := range sbom.Dependencies {
		wantDependencies, ok := expectedGraph[node.Ref]
		if !ok {
			return fmt.Errorf("SBOM dependency graph contains unknown node %s", node.Ref)
		}
		gotDependencies := append([]string(nil), node.DependsOn...)
		sort.Strings(gotDependencies)
		if strings.Join(gotDependencies, "\n") != strings.Join(wantDependencies, "\n") {
			return fmt.Errorf("SBOM dependency graph for %s does not match the lock", node.Ref)
		}
		delete(expectedGraph, node.Ref)
	}
	if len(expectedGraph) != 0 {
		return fmt.Errorf("SBOM dependency graph omits locked nodes")
	}
	return nil
}

func validateArtifactProvenance(packageDir string, manifest Manifest, lock artifactLock) error {
	payload, err := os.ReadFile(filepath.Join(packageDir, filepath.FromSlash(*manifest.Artifact.Provenance)))
	if err != nil {
		return err
	}
	var provenance artifactProvenance
	if err := json.Unmarshal(payload, &provenance); err != nil {
		return err
	}
	if provenance.ProvenanceVersion != "1.0" || provenance.Builder != "ai-skills-guide/prepare-public-assets" ||
		provenance.Subject.Module != "plugins" || provenance.Subject.ID != manifest.ID || provenance.Subject.Version != manifest.Version {
		return fmt.Errorf("provenance subject or builder does not match plugins:%s@%s", manifest.ID, manifest.Version)
	}
	excluded := map[string]struct{}{
		*manifest.Artifact.Checksums:  {},
		*manifest.Artifact.SBOM:       {},
		*manifest.Artifact.Provenance: {},
	}
	closureDigest, err := digestPackageTreeExcluding(packageDir, excluded)
	if err != nil {
		return err
	}
	if provenance.Subject.ClosureSHA256 != closureDigest {
		return fmt.Errorf("provenance closure digest does not match the archive")
	}
	wantMaterials := make(map[string]string, len(lock.Components))
	for _, component := range lock.Components {
		wantMaterials[component.Module+":"+component.ID+"@"+component.Version] = component.ContentSHA256
	}
	if len(provenance.Materials) != len(wantMaterials) {
		return fmt.Errorf("provenance materials do not describe the complete locked closure")
	}
	for _, material := range provenance.Materials {
		digest, ok := wantMaterials[material.Ref]
		if !ok || material.SHA256 != digest {
			return fmt.Errorf("provenance material %s does not match the lock", material.Ref)
		}
		delete(wantMaterials, material.Ref)
	}
	if len(wantMaterials) != 0 {
		return fmt.Errorf("provenance materials omit locked components")
	}
	return nil
}

func rejectUndeclaredBundledPackages(moduleRoot string, expected map[string]struct{}) error {
	entries, err := os.ReadDir(moduleRoot)
	if os.IsNotExist(err) {
		if len(expected) == 0 {
			return nil
		}
		return fmt.Errorf("missing bundled module directory %s", moduleRoot)
	}
	if err != nil {
		return err
	}
	for _, category := range entries {
		if !category.IsDir() {
			return fmt.Errorf("unexpected file in bundled module directory: %s", filepath.Join(moduleRoot, category.Name()))
		}
		packages, err := os.ReadDir(filepath.Join(moduleRoot, category.Name()))
		if err != nil {
			return err
		}
		for _, packageEntry := range packages {
			id := category.Name() + "/" + packageEntry.Name()
			if !packageEntry.IsDir() {
				return fmt.Errorf("bundled dependency %s is not a directory", id)
			}
			if _, ok := expected[id]; !ok {
				return fmt.Errorf("archive contains undeclared bundled dependency %s", id)
			}
		}
	}
	return nil
}

func validSemver(version string) bool {
	if !semverPattern.MatchString(version) {
		return false
	}
	withoutBuild := strings.SplitN(version, "+", 2)[0]
	parts := strings.SplitN(withoutBuild, "-", 2)
	if len(parts) == 1 {
		return true
	}
	for _, identifier := range strings.Split(parts[1], ".") {
		allDigits := true
		for _, character := range identifier {
			if character < '0' || character > '9' {
				allDigits = false
				break
			}
		}
		if allDigits && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func authenticationIsRequired(manifest Manifest) bool {
	if len(manifest.Requires.Secrets) > 0 {
		return true
	}
	for _, requirement := range manifest.Operational.AuthRequired {
		normalized := strings.ToLower(strings.TrimSpace(requirement))
		if normalized != "" && normalized != "none" && !strings.HasPrefix(normalized, "none ") {
			return true
		}
	}
	return false
}

func validateContainedPath(packageDir, declared string, wantDirectory bool) error {
	if declared == "" || filepath.IsAbs(declared) || strings.Contains(declared, `\`) {
		return fmt.Errorf("path %q must be a non-empty package-relative path using '/' separators", declared)
	}
	clean := filepath.Clean(filepath.FromSlash(declared))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q escapes the package", declared)
	}
	target := filepath.Join(packageDir, clean)
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("declared path %q does not exist", declared)
		}
		return fmt.Errorf("stat declared path %q: %w", declared, err)
	}
	realPackage, err := filepath.EvalSymlinks(packageDir)
	if err != nil {
		return fmt.Errorf("resolve package directory: %w", err)
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return fmt.Errorf("resolve declared path %q: %w", declared, err)
	}
	rel, err := filepath.Rel(realPackage, realTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("declared path %q resolves outside the package", declared)
	}
	if wantDirectory && !info.IsDir() {
		return fmt.Errorf("declared path %q must be a directory", declared)
	}
	if !wantDirectory && !info.Mode().IsRegular() {
		return fmt.Errorf("declared path %q must be a regular file", declared)
	}
	return nil
}

func validateExecutionPaths(packageDir, manifestPath string, execution ExecutionMetadata) error {
	if !isExecutableKind(execution.Kind) {
		return nil
	}
	commands := []struct {
		label   string
		command []string
	}{
		{label: "execution.command", command: execution.Command},
		{label: "execution.healthcheck", command: execution.Healthcheck},
		{label: "execution.smoke_test", command: execution.SmokeTest},
	}
	for _, declaredCommand := range commands {
		for _, candidate := range commandLocalPaths(declaredCommand.command) {
			if err := validateContainedPath(packageDir, candidate, false); err != nil {
				return fmt.Errorf("manifest %s %s: %w", manifestPath, declaredCommand.label, err)
			}
		}
	}
	return nil
}

func commandLocalPaths(command []string) []string {
	paths := make([]string, 0, 1)
	for index, value := range command {
		if value == "" || strings.HasPrefix(value, "-") || strings.Contains(value, "://") {
			continue
		}
		if strings.HasPrefix(value, "./") || strings.Contains(value, "/") || (index > 0 && hasExecutableFileSuffix(value)) {
			paths = append(paths, strings.TrimPrefix(value, "./"))
			break
		}
	}
	return paths
}

func hasExecutableFileSuffix(value string) bool {
	switch strings.ToLower(filepath.Ext(value)) {
	case ".js", ".mjs", ".cjs", ".py", ".sh", ".rb", ".php", ".jar", ".wasm":
		return true
	default:
		return false
	}
}

func isExecutableKind(kind string) bool {
	switch kind {
	case "script", "cli", "mcp-server", "service", "orchestrator":
		return true
	default:
		return false
	}
}

func validateFreshEvidence(manifestPath string, manifest Manifest, now time.Time) error {
	if len(manifest.Verification.Evidence) == 0 {
		return fmt.Errorf("manifest %s claims usable-now without evidence references", manifestPath)
	}
	observedAt, err := time.Parse(time.RFC3339, manifest.Verification.LastVerifiedAt)
	if err != nil {
		return fmt.Errorf("manifest %s claims usable-now with invalid verification timestamp: %w", manifestPath, err)
	}
	if observedAt.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("manifest %s claims usable-now with a future verification timestamp", manifestPath)
	}
	lifetime := instructionEvidenceLifetime
	if isExecutableKind(manifest.Execution.Kind) {
		lifetime = executableEvidenceLifetime
	}
	if manifest.Authentication.Status != "none" {
		lifetime = authEvidenceLifetime
	}
	if now.Sub(observedAt) > lifetime {
		return fmt.Errorf("manifest %s claims usable-now with evidence older than %s", manifestPath, lifetime)
	}
	return nil
}

func validateDependencyGraph(root string, records []manifestRecord) error {
	byKind := map[string]map[string]manifestRecord{}
	all := map[string]manifestRecord{}
	for _, record := range records {
		if byKind[record.kind] == nil {
			byKind[record.kind] = map[string]manifestRecord{}
		}
		if previous, exists := byKind[record.kind][record.manifest.ID]; exists {
			return fmt.Errorf("duplicate %s id %q in %s and %s", record.kind, record.manifest.ID, previous.path, record.path)
		}
		byKind[record.kind][record.manifest.ID] = record
		all[record.kind+":"+record.manifest.ID] = record
	}

	edges := map[string][]string{}
	for _, record := range records {
		from := record.kind + ":" + record.manifest.ID
		for _, dependency := range internalDependencies(record) {
			targets := byKind[dependency.kind]
			target, exists := targets[dependency.id]
			if !exists {
				if dependency.kind == "tool" && record.kind == "skill" && isRepositoryFile(root, dependency.id) {
					continue
				}
				return fmt.Errorf("manifest %s references missing %s %q", record.path, dependency.kind, dependency.id)
			}
			if !sharesRuntime(record.manifest.Runtimes, target.manifest.Runtimes) {
				return fmt.Errorf("manifest %s and %s %q have no compatible runtime", record.path, dependency.kind, dependency.id)
			}
			edges[from] = append(edges[from], dependency.kind+":"+dependency.id)
		}
	}

	state := map[string]uint8{}
	stack := make([]string, 0)
	var visit func(string) error
	visit = func(node string) error {
		state[node] = 1
		stack = append(stack, node)
		for _, next := range edges[node] {
			if state[next] == 1 {
				return fmt.Errorf("dependency cycle detected: %s -> %s", strings.Join(stack, " -> "), next)
			}
			if state[next] == 0 {
				if err := visit(next); err != nil {
					return err
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[node] = 2
		return nil
	}
	keys := make([]string, 0, len(all))
	for key := range all {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if state[key] == 0 {
			if err := visit(key); err != nil {
				return err
			}
		}
	}
	return nil
}

func isRepositoryFile(root, reference string) bool {
	if filepath.IsAbs(reference) || strings.Contains(reference, `\`) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(reference))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	info, err := os.Stat(filepath.Join(root, clean))
	return err == nil && info.Mode().IsRegular()
}

type dependencyRef struct {
	kind string
	id   string
}

func internalDependencies(record manifestRecord) []dependencyRef {
	refs := make([]dependencyRef, 0)
	add := func(kind string, ids []string, requireQualified bool) {
		for _, id := range ids {
			if !requireQualified || strings.Contains(id, "/") {
				refs = append(refs, dependencyRef{kind: kind, id: id})
			}
		}
	}
	add("agent", record.manifest.Dependencies.Agents, false)
	add("skill", record.manifest.Dependencies.Skills, false)
	add("tool", record.manifest.Dependencies.Tools, true)
	if record.kind == "plugin" {
		add("skill", record.manifest.Includes.Skills, false)
		add("agent", record.manifest.Includes.Agents, false)
		add("tool", record.manifest.Includes.Tools, false)
	}
	return refs
}

func sharesRuntime(left, right []string) bool {
	for _, a := range left {
		for _, b := range right {
			if a == b || a == "generic" || b == "generic" {
				return true
			}
		}
	}
	return false
}
