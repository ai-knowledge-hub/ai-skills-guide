package registry

import (
	"fmt"
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
			if err := validatePackage(path, manifest, now); err != nil {
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

func validatePackage(manifestPath string, manifest Manifest, now time.Time) error {
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
		artifactPaths := []struct {
			label string
			path  *string
		}{
			{label: "artifact.dependency_lock", path: manifest.Artifact.DependencyLock},
			{label: "artifact.checksums", path: manifest.Artifact.Checksums},
			{label: "artifact.sbom", path: manifest.Artifact.SBOM},
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
		if authenticationIsRequired(manifest) && manifest.Authentication.Status == "none" {
			return fmt.Errorf("manifest %s requires authentication but declares authentication.status none", manifestPath)
		}
		if manifest.Usability.Availability == "usable-now" {
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
