package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://skills.ai-knowledge-hub.org"

func BuildIndex(root string) (Index, error) {
	return BuildSkillsIndex(root)
}

func BuildSkillsIndex(root string) (Index, error) {
	return buildIndexFor(root, "skills", "skill.yaml")
}

func BuildAgentsIndex(root string) (Index, error) {
	return buildIndexFor(root, "agents", "agent.yaml")
}

func BuildToolsIndex(root string) (Index, error) {
	return buildIndexFor(root, "tools-mcp", "tool.yaml")
}

func BuildPluginsIndex(root string) (Index, error) {
	return buildIndexFor(root, "plugins", "plugin.yaml")
}

func buildIndexFor(root, moduleDir, manifestName string) (Index, error) {
	manifests, err := findManifests(filepath.Join(root, moduleDir), manifestName)
	if err != nil {
		return Index{}, err
	}

	skills := make([]SkillEntry, 0, len(manifests))
	for _, manifestPath := range manifests {
		m, err := ParseManifest(manifestPath)
		if err != nil {
			return Index{}, err
		}
		skillDir := filepath.Dir(manifestPath)
		if err := validatePackage(manifestPath, m, time.Now().UTC(), true, false); err != nil {
			return Index{}, err
		}
		sha, err := digestPublishedArtifact(root, moduleDir, m, skillDir)
		if err != nil {
			return Index{}, err
		}
		manifestSHA, err := digestFile(manifestPath)
		if err != nil {
			return Index{}, err
		}

		entry := ProjectManifest(m)
		entry.Latest = m.Version
		entry.Versions = []VersionEntry{{
			Version:        m.Version,
			ReleasedAt:     m.ReleasedAt,
			ManifestURL:    fmt.Sprintf("%s/release-manifests/%s/%s/%s.yaml", baseURL, moduleDir, m.ID, m.Version),
			ManifestSHA256: manifestSHA,
			ArtifactURL:    fmt.Sprintf("%s/artifacts/%s/%s.tar.gz", baseURL, m.ID, m.Version),
			SHA256:         sha,
		}}
		skills = append(skills, entry)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].ID < skills[j].ID
	})

	generatedAt := "1970-01-01T00:00:00Z"
	for _, skill := range skills {
		for _, version := range skill.Versions {
			if version.ReleasedAt > generatedAt {
				generatedAt = version.ReleasedAt
			}
		}
	}

	return Index{
		RegistryVersion: "1.3",
		GeneratedAt:     generatedAt,
		Skills:          skills,
	}, nil
}

// digestPublishedArtifact binds a registry release to the exact bytes a
// remote installer downloads. During initial release creation the retained
// archive does not exist yet, so the source-tree digest remains a bootstrap
// value; publication must rebuild the tracked registry after persisting the
// final immutable archive.
func digestPublishedArtifact(root, moduleDir string, manifest Manifest, sourceDir string) (string, error) {
	archivePath := filepath.Join(root, "releases", moduleDir, filepath.FromSlash(manifest.ID), manifest.Version, "package.tar.gz")
	info, err := os.Stat(archivePath)
	if err == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("retained release is not a regular file: %s", archivePath)
		}
		return digestFile(archivePath)
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect retained release %s: %w", archivePath, err)
	}
	return digestSkillDir(sourceDir)
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s for digest: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("digest %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func declaredUsability(m Manifest) UsabilityMetadata {
	result := m.Usability
	result.RequiresSetup = cloneNonEmptyStrings(m.Usability.RequiresSetup)
	result.Limitations = cloneNonEmptyStrings(m.Usability.Limitations)
	result.ExecutableHelpers = nil
	if len(m.Usability.ExecutableHelpers) > 0 {
		result.ExecutableHelpers = make([]ExecutableHelperMetadata, len(m.Usability.ExecutableHelpers))
		for index, helper := range m.Usability.ExecutableHelpers {
			result.ExecutableHelpers[index] = helper
			result.ExecutableHelpers[index].Limitations = cloneNonEmptyStrings(helper.Limitations)
		}
	}
	result.Source = "declared"
	return result
}

func cloneNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

// ProjectManifest returns every registry entry field derived from a manifest.
// Release catalog fields (latest, versions, URLs, and checksums) are populated
// separately by the publisher.
func ProjectManifest(m Manifest) SkillEntry {
	entry := SkillEntry{
		SchemaVersion:    m.SchemaVersion,
		ID:               m.ID,
		Name:             m.Name,
		Description:      m.Description,
		Category:         m.Category,
		Runtimes:         append([]string{}, m.Runtimes...),
		Tags:             append([]string{}, m.Tags...),
		Readiness:        readinessFor(m),
		SecurityReviewed: m.SecurityReviewed,
		Deprecated:       m.Deprecated,
		ReplacedBy:       m.ReplacedBy,
		Usability:        declaredUsability(m),
	}
	if strings.HasPrefix(m.SchemaVersion, "2.") {
		execution := m.Execution
		artifact := m.Artifact
		authentication := m.Authentication
		verification := m.Verification
		entry.Execution = &execution
		entry.Artifact = &artifact
		entry.Authentication = &authentication
		entry.Verification = &verification
		entry.ProviderDependencies = append([]ProviderDependencyMetadata(nil), m.ProviderDependencies...)
		entry.CapabilityReadiness = cloneCapabilityReadiness(m.CapabilityReadiness)
	}
	if hasOperational(m.Operational) {
		operational := m.Operational
		entry.Operational = &operational
	}
	if hasDependencies(m.Dependencies) {
		dependencies := m.Dependencies
		entry.Dependencies = &dependencies
	}
	if hasIncludes(m.Includes) {
		includes := m.Includes
		entry.Includes = &includes
	}
	if hasRequirements(m.Requires) {
		requires := m.Requires
		entry.Requires = &requires
	}
	return ApplyLifecycleProjection(entry)
}

// ApplyLifecycleProjection suppresses readiness claims whenever entry-level
// deprecation is effective. The manifest and verification fields remain intact
// as audit evidence, while every user-facing usability scope becomes
// conservative and deterministic.
func ApplyLifecycleProjection(entry SkillEntry) SkillEntry {
	if !entry.Deprecated {
		return entry
	}
	entry.Readiness = "deprecated"
	demoted := false
	if isPositiveUsability(entry.Usability.Availability) {
		entry.Usability.Availability = "not-verified"
		demoted = true
	}
	for index := range entry.Usability.ExecutableHelpers {
		if isPositiveUsability(entry.Usability.ExecutableHelpers[index].Availability) {
			entry.Usability.ExecutableHelpers[index].Availability = "not-verified"
			demoted = true
		}
	}
	for index := range entry.CapabilityReadiness {
		if isPositiveUsability(entry.CapabilityReadiness[index].Availability) {
			entry.CapabilityReadiness[index].Availability = "not-verified"
			demoted = true
		}
	}
	if demoted {
		entry.Usability.Source = "inferred"
	}
	return entry
}

func isPositiveUsability(availability string) bool {
	return availability == "usable-now" || availability == "setup-required"
}

func cloneCapabilityReadiness(values []CapabilityReadinessMetadata) []CapabilityReadinessMetadata {
	if len(values) == 0 {
		return nil
	}
	result := make([]CapabilityReadinessMetadata, len(values))
	for index, value := range values {
		result[index] = value
		result[index].RequiresSetup = cloneNonEmptyStrings(value.RequiresSetup)
		result[index].Limitations = cloneNonEmptyStrings(value.Limitations)
	}
	return result
}

func hasPositiveCapabilityReadiness(values []CapabilityReadinessMetadata) bool {
	for _, capability := range values {
		if isPositiveUsability(capability.Availability) {
			return true
		}
	}
	return false
}

// ProjectManifestForCurrentCatalog derives the operator-visible projection at
// publication time without changing the immutable manifest or release
// sidecar. An expired usable-now claim is retained for audit in Verification,
// but it cannot remain advertised as current readiness.
func ProjectManifestForCurrentCatalog(m Manifest, now time.Time) SkillEntry {
	entry := ProjectManifest(m)
	if m.Usability.Availability == "usable-now" || hasPositiveCapabilityReadiness(m.CapabilityReadiness) {
		if err := validateFreshEvidence(m.ID, m, now.UTC()); err != nil {
			if entry.Usability.Availability == "usable-now" {
				entry.Usability.Availability = "not-verified"
			}
			entry.Usability.Source = "inferred"
			for index := range entry.CapabilityReadiness {
				if isPositiveUsability(entry.CapabilityReadiness[index].Availability) {
					entry.CapabilityReadiness[index].Availability = "not-verified"
				}
			}
		}
	}
	return ApplyLifecycleProjection(entry)
}

func hasIncludes(includes IncludeSet) bool {
	return len(includes.Skills) > 0 || len(includes.Agents) > 0 || len(includes.Tools) > 0 || len(includes.Hooks) > 0
}

func hasOperational(operational OperationalMetadata) bool {
	return operational.ConnectedSystem != "" ||
		len(operational.Capabilities) > 0 ||
		len(operational.AuthRequired) > 0 ||
		operational.AccessLevel != "" ||
		operational.TrustBoundary != "" ||
		operational.ApprovalBoundary != "" ||
		operational.Role != "" ||
		len(operational.Coordinates) > 0 ||
		operational.AutonomyLevel != "" ||
		len(operational.Outputs) > 0 ||
		operational.UseWhen != "" ||
		operational.ExecutionMode != ""
}

func hasDependencies(dependencies DependencySet) bool {
	return len(dependencies.Agents) > 0 || len(dependencies.Skills) > 0 || len(dependencies.Tools) > 0 || len(dependencies.APIs) > 0 || len(dependencies.MCPServers) > 0
}

func hasRequirements(requires RequirementSet) bool {
	return len(requires.Secrets) > 0 || len(requires.Approvals) > 0
}

func readinessFor(m Manifest) string {
	if m.Deprecated {
		return "deprecated"
	}
	if m.SecurityReviewed {
		return "reviewed"
	}
	return "experimental"
}

func WriteIndex(path string, index Index) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write index %s: %w", path, err)
	}
	return nil
}

func findManifests(moduleRoot, manifestName string) ([]string, error) {
	entries := make([]string, 0)
	if _, err := os.Stat(moduleRoot); err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, fmt.Errorf("stat module root %s: %w", moduleRoot, err)
	}

	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == manifestName {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk module root %s: %w", moduleRoot, err)
	}
	sort.Strings(entries)
	return entries, nil
}

func digestSkillDir(skillDir string) (string, error) {
	type fileEntry struct {
		rel string
		abs string
	}
	files := make([]fileEntry, 0)
	err := filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if containsIgnoredDigestPart(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, fileEntry{rel: filepath.ToSlash(rel), abs: path})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk skill dir %s: %w", skillDir, err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].rel < files[j].rel
	})

	h := sha256.New()
	for _, f := range files {
		if _, err := io.WriteString(h, f.rel); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte{0}); err != nil {
			return "", err
		}
		data, err := os.ReadFile(f.abs)
		if err != nil {
			return "", fmt.Errorf("read skill file %s: %w", f.abs, err)
		}
		if _, err := h.Write(data); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte{0}); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func containsIgnoredDigestPart(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
		if part == "__pycache__" || strings.HasSuffix(part, ".pyc") || strings.HasSuffix(part, ".pyo") {
			return true
		}
	}
	return false
}
