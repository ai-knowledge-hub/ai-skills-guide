package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateRepositoryAcceptsCompleteClosure(t *testing.T) {
	root := t.TempDir()
	writeTestModule(t, root, "skill", "shared/base-skill", "codex", "")
	writeTestModule(t, root, "agent", "shared/demo-agent", "codex", `dependencies:
  skills: [shared/base-skill]
`)
	writeTestModule(t, root, "plugin", "shared/demo-plugin", "codex", `includes:
  agents: [shared/demo-agent]
`)

	count, err := ValidateRepository(root)
	if err != nil {
		t.Fatalf("validate complete closure: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 manifests, got %d", count)
	}
}

func TestProviderDependencyDeclarationsRequireExactIncludedToolsAndAuthentication(t *testing.T) {
	base := Manifest{
		SchemaVersion:  "2.1",
		Authentication: AuthenticationMetadata{Status: "required"},
		Includes:       IncludeSet{Tools: []string{"analytics/ga4-mcp-connector", "ads/meta-ads-mcp-connector"}},
		ProviderDependencies: []ProviderDependencyMetadata{
			{Tool: "analytics/ga4-mcp-connector", Requirement: "required", Access: "read-only"},
			{Tool: "ads/meta-ads-mcp-connector", Requirement: "optional", Access: "read-only"},
		},
	}
	if err := validateProviderDependencyDeclarations("plugin.yaml", base); err != nil {
		t.Fatalf("valid provider dependency contract rejected: %v", err)
	}

	tests := map[string]func(*Manifest){
		"tool not included": func(manifest *Manifest) {
			manifest.ProviderDependencies[0].Tool = "warehouse/bigquery-mcp-query-runner"
		},
		"duplicate tool": func(manifest *Manifest) {
			manifest.ProviderDependencies[1].Tool = manifest.ProviderDependencies[0].Tool
		},
		"no required provider": func(manifest *Manifest) {
			manifest.ProviderDependencies[0].Requirement = "optional"
		},
		"flat authentication disabled": func(manifest *Manifest) {
			manifest.Authentication.Status = "none"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := base
			candidate.ProviderDependencies = append([]ProviderDependencyMetadata(nil), base.ProviderDependencies...)
			mutate(&candidate)
			if err := validateProviderDependencyDeclarations("plugin.yaml", candidate); err == nil {
				t.Fatal("invalid provider dependency contract was accepted")
			}
		})
	}
}

func TestProviderDependencyAuthorityRequiresExactToolAccess(t *testing.T) {
	authenticatedReadOnlyTool := Manifest{
		Authentication: AuthenticationMetadata{Status: "required"},
		Operational:    OperationalMetadata{AccessLevel: "read-only"},
	}
	readOnly := ProviderDependencyMetadata{Tool: "analytics/provider", Requirement: "required", Access: "read-only"}
	if err := validateProviderDependencyAuthority("plugin.yaml", readOnly, authenticatedReadOnlyTool); err != nil {
		t.Fatalf("matching read-only authority rejected: %v", err)
	}

	mutations := map[string]struct {
		dependency ProviderDependencyMetadata
		tool       Manifest
	}{
		"read-write dependency with read-only tool": {
			dependency: ProviderDependencyMetadata{Tool: "analytics/provider", Requirement: "required", Access: "read-write"},
			tool:       authenticatedReadOnlyTool,
		},
		"read-only dependency with read-write tool": {
			dependency: readOnly,
			tool: Manifest{
				Authentication: AuthenticationMetadata{Status: "required"},
				Operational:    OperationalMetadata{AccessLevel: "read-write"},
			},
		},
		"provider without authentication": {
			dependency: readOnly,
			tool: Manifest{
				Authentication: AuthenticationMetadata{Status: "none"},
				Operational:    OperationalMetadata{AccessLevel: "read-only"},
			},
		},
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			if err := validateProviderDependencyAuthority("plugin.yaml", mutation.dependency, mutation.tool); err == nil {
				t.Fatal("incompatible provider authority was accepted")
			}
		})
	}
}

func TestValidateRepositoryRejectsMissingUsabilityDeclaration(t *testing.T) {
	root := t.TempDir()
	path := writeTestModule(t, root, "skill", "shared/undeclared-usability", "codex", "")
	replaceTestManifest(t, path, "usability:\n  availability: documentation-only\n  execution: instructions\n", "")

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, "must explicitly declare usability.availability and usability.execution")
}

func TestValidateRepositoryRejectsMissingDeclaredEntrypoint(t *testing.T) {
	root := t.TempDir()
	path := writeTestModule(t, root, "skill", "shared/missing-script", "codex", "")
	appendTestManifest(t, path, "  scripts_dir: scripts\n", "  spec: SKILL.md\n")

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, `entrypoint scripts_dir: declared path "scripts" does not exist`)
}

func TestValidateRepositoryRejectsCoordinatedMissingClosure(t *testing.T) {
	root := t.TempDir()
	// The plugin and agent are internally consistent and individually parse;
	// only repository closure reveals that the agent's implementation dependency
	// is absent.
	writeTestModule(t, root, "agent", "shared/demo-agent", "codex", `dependencies:
  skills: [shared/absent-skill]
`)
	writeTestModule(t, root, "plugin", "shared/demo-plugin", "codex", `includes:
  agents: [shared/demo-agent]
`)

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, `references missing skill "shared/absent-skill"`)
}

func TestValidateRepositoryRejectsRuntimeIncompatibleClosure(t *testing.T) {
	root := t.TempDir()
	writeTestModule(t, root, "skill", "shared/claude-only", "claude", "")
	writeTestModule(t, root, "agent", "shared/codex-agent", "codex", `dependencies:
  skills: [shared/claude-only]
`)

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, "have no compatible runtime")
}

func TestValidateRepositoryRejectsDependencyCycle(t *testing.T) {
	root := t.TempDir()
	writeTestModule(t, root, "agent", "shared/agent-a", "codex", `dependencies:
  agents: [shared/agent-b]
`)
	writeTestModule(t, root, "agent", "shared/agent-b", "codex", `dependencies:
  agents: [shared/agent-a]
`)

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, "dependency cycle detected")
}

func TestValidateRepositoryRejectsRemoteIntegrationWithoutImplementation(t *testing.T) {
	root := t.TempDir()
	writeTestModule(t, root, "tool", "shared/docs-only-connector", "codex", `usability:
  availability: usable-now
  execution: remote-integration
  quickstart: Follow the examples.
`)

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, "usable-now remote-integration without a scripts_dir entrypoint")
}

func TestValidateRepositoryRejectsInvalidPackageVersion(t *testing.T) {
	for _, version := range []string{"latest", "1.0.0-01"} {
		t.Run(version, func(t *testing.T) {
			root := t.TempDir()
			path := writeTestModule(t, root, "skill", "shared/bad-version", "codex", "")
			replaceTestManifest(t, path, "version: 1.0.0", "version: "+version)

			_, err := ValidateRepository(root)
			assertAdmissionError(t, err, fmt.Sprintf("invalid semantic version %q", version))
		})
	}
}

func TestBuildIndexRejectsUnlaunchableV2Executable(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "shared", "missing-command")
	if err := os.MkdirAll(filepath.Join(entryDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := v2ExecutableManifest("shared/missing-command", time.Now().UTC())
	writeTestFile(t, filepath.Join(entryDir, "tool.yaml"), manifest, 0o644)
	writeTestFile(t, filepath.Join(entryDir, "TOOL.md"), "# Tool\n", 0o644)
	writeTestFile(t, filepath.Join(entryDir, "checksums.txt"), "fixture\n", 0o644)
	writeTestFile(t, filepath.Join(entryDir, "sbom.cdx.json"), "{}\n", 0o644)

	_, err := BuildToolsIndex(root)
	assertAdmissionError(t, err, `execution.command: declared path "bin/tool" does not exist`)
}

func TestBuildIndexRejectsStaleEvidenceForDeclaredUsableNow(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "shared", "stale-tool")
	if err := os.MkdirAll(filepath.Join(entryDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := v2ExecutableManifest("shared/stale-tool", time.Now().UTC().Add(-executableEvidenceLifetime-time.Hour))
	writeTestFile(t, filepath.Join(entryDir, "tool.yaml"), manifest, 0o644)
	for _, file := range []string{"TOOL.md", "checksums.txt", "sbom.cdx.json", "bin/tool"} {
		writeTestFile(t, filepath.Join(entryDir, filepath.FromSlash(file)), "fixture\n", 0o755)
	}

	_, err := BuildToolsIndex(root)
	assertAdmissionError(t, err, "claims usable-now with evidence older than")
}

func TestRunnableBundleUsesExecutableEvidenceLifetime(t *testing.T) {
	now := time.Now().UTC()
	manifest := Manifest{
		Execution:      ExecutionMetadata{Kind: "bundle"},
		Authentication: AuthenticationMetadata{Status: "none"},
		Verification: VerificationMetadata{
			Evidence:       []string{"evidence://tests/plugin/clean-client"},
			LastVerifiedAt: now.Add(-executableEvidenceLifetime - time.Hour).Format(time.RFC3339),
		},
	}
	if err := validateFreshEvidence("runnable-bundle", manifest, now); err == nil || !strings.Contains(err.Error(), "evidence older than") {
		t.Fatalf("expected runnable bundle evidence to expire after %s, got %v", executableEvidenceLifetime, err)
	}
	manifest.Verification.LastVerifiedAt = now.Add(-executableEvidenceLifetime + time.Hour).Format(time.RFC3339)
	if err := validateFreshEvidence("runnable-bundle", manifest, now); err != nil {
		t.Fatalf("fresh runnable bundle evidence was rejected: %v", err)
	}
}

func TestBuildIndexSuppressesDeprecatedRootAndHelperUsability(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "shared", "deprecated-tool")
	if err := os.MkdirAll(filepath.Join(entryDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := v2ExecutableManifest("shared/deprecated-tool", time.Now().UTC())
	manifest = strings.Replace(manifest, `schema_version: "2.0"`, `schema_version: "2.1"`, 1)
	manifest = strings.Replace(
		manifest,
		"  quickstart: bin/tool\n",
		"  quickstart: bin/tool\n  executable_helpers:\n    - entrypoint: bin/tool\n      availability: usable-now\n      execution: local-tool\n      limitations: [Deprecated helper.]\n",
		1,
	)
	manifest = strings.Replace(
		manifest,
		"deprecated: false\n",
		"deprecated: true\nreplaced_by: shared/replacement-tool\n",
		1,
	)
	writeTestFile(t, filepath.Join(entryDir, "tool.yaml"), manifest, 0o644)
	for _, file := range []string{"TOOL.md", "checksums.txt", "sbom.cdx.json", "bin/tool"} {
		writeTestFile(t, filepath.Join(entryDir, filepath.FromSlash(file)), "fixture\n", 0o755)
	}

	index, err := BuildToolsIndex(root)
	if err != nil {
		t.Fatalf("build deprecated tool index: %v", err)
	}
	entry := index.Skills[0]
	if entry.Readiness != "deprecated" || entry.Usability.Availability != "not-verified" || entry.Usability.Source != "inferred" {
		t.Fatalf("deprecated catalog entry retained positive usability: %#v", entry)
	}
	if len(entry.Usability.ExecutableHelpers) != 1 || entry.Usability.ExecutableHelpers[0].Availability != "not-verified" {
		t.Fatalf("deprecated catalog helper retained positive usability: %#v", entry.Usability.ExecutableHelpers)
	}
}

func TestValidatePackageManifestRejectsCanonicalSchemaViolations(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "unknown property",
			mutate: func(manifest string) string {
				return manifest + "unknown_contract_field: true\n"
			},
		},
		{
			name: "unknown availability",
			mutate: func(manifest string) string {
				return strings.Replace(manifest, "availability: usable-now", "availability: future-ready", 1)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			entryDir := filepath.Join(root, "tools-mcp", "shared", "schema-bypass")
			manifestPath := filepath.Join(entryDir, "tool.yaml")
			writeTestFile(t, manifestPath, test.mutate(v2ExecutableManifest("shared/schema-bypass", time.Now().UTC())), 0o644)
			for _, file := range []string{"TOOL.md", "checksums.txt", "sbom.cdx.json", "bin/tool"} {
				writeTestFile(t, filepath.Join(entryDir, filepath.FromSlash(file)), "fixture\n", 0o755)
			}

			_, err := ValidatePackageManifest(manifestPath)
			assertAdmissionError(t, err, "does not satisfy tool.schema.json")
		})
	}
}

func TestValidateAuthenticationDriversBeforeAdmission(t *testing.T) {
	authentication := AuthenticationMetadata{Status: "required", Methods: []string{"oauth-device-flow"}}
	root := t.TempDir()
	manifestPath := filepath.Join(root, "tool.yaml")
	if err := validateAuthenticationDrivers(root, manifestPath, authentication, []string{"codex"}); err == nil || !strings.Contains(err.Error(), "requires a packaged driver") {
		t.Fatalf("missing typed authentication driver was accepted: %v", err)
	}

	writeTestFile(t, filepath.Join(root, "bin", "auth-driver"), "fixture\n", 0o755)
	driverPath := filepath.Join(root, "auth", "oauth-device-flow.json")
	validDriver := `{
  "schema_version": "skills-hub.auth-driver/v1",
  "method": "oauth-device-flow",
  "flow": "device-code",
  "runtimes": ["codex"],
  "bootstrap": {"command": ["bin/auth-driver", "bootstrap"]},
  "credential": {"command": ["bin/auth-driver", "credential"]},
  "status": {"command": ["bin/auth-driver", "status"]}
}`
	writeTestFile(t, driverPath, validDriver, 0o644)
	if err := validateAuthenticationDrivers(root, manifestPath, authentication, []string{"codex"}); err != nil {
		t.Fatalf("valid typed authentication driver rejected: %v", err)
	}

	writeTestFile(t, driverPath, strings.Replace(validDriver, `"runtimes": ["codex"]`, `"runtimes": ["claude"]`, 1), 0o644)
	if err := validateAuthenticationDrivers(root, manifestPath, authentication, []string{"codex"}); err == nil || !strings.Contains(err.Error(), "does not support declared runtime") {
		t.Fatalf("runtime-incompatible authentication driver was accepted: %v", err)
	}

	writeTestFile(t, driverPath, strings.Replace(validDriver, `"flow": "device-code"`, `"flow": "interactive-browser"`, 1), 0o644)
	if err := validateAuthenticationDrivers(root, manifestPath, authentication, []string{"codex"}); err == nil || !strings.Contains(err.Error(), "does not match method") {
		t.Fatalf("wrong-flow authentication driver was accepted: %v", err)
	}
}

func TestValidatePackageManifestRejectsMissingSelfContainedPluginDependency(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "plugins", "engineering", "closure-plugin")
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	writeTestFile(t, manifestPath, v2PluginManifest("engineering/closure-plugin", time.Now().UTC()), 0o644)
	writeTestFile(t, filepath.Join(pluginDir, "plugin.json"), "{}\n", 0o644)

	_, err := ValidatePackageManifest(manifestPath)
	assertAdmissionError(t, err, "self-contained dependency engineering/closure-skill")
}

func TestValidatePackageManifestRejectsV21PluginWithoutIntegrityMetadata(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "plugins", "engineering", "closure-plugin")
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	writeTestFile(t, manifestPath, v2PluginManifest("engineering/closure-plugin", time.Now().UTC()), 0o644)
	writeTestFile(t, filepath.Join(pluginDir, "plugin.json"), "{}\n", 0o644)
	dependencyDir := filepath.Join(pluginDir, "bundled", "skills", "engineering", "closure-skill")
	writeTestFile(t, filepath.Join(dependencyDir, "skill.yaml"), `schema_version: "2.1"
id: engineering/closure-skill
name: Closure Skill
description: Bundled dependency used to verify metadata admission.
version: 1.0.0
released_at: "2026-09-14T00:00:00Z"
category: engineering/testing-quality
tags: [testing]
license: MIT
author:
  name: Test Maintainer
runtimes: [generic]
entrypoints:
  skill_md: SKILL.md
usability:
  availability: documentation-only
  execution: instructions
execution:
  kind: instructions
  supported_platforms: [linux, macos, windows]
  supported_runtimes: [generic]
artifact:
  self_contained: true
  dependency_lock: null
  checksums: null
  sbom: null
authentication:
  status: none
  methods: [none]
  credential_bindings: []
  scopes: []
  credential_storage: No credentials.
  validation: Confirm no authentication challenge.
  revocation: Not applicable.
verification:
  evidence: [evidence://tests/closure-skill]
  last_verified_at: "2026-09-14T00:00:00Z"
deprecated: false
`, 0o644)
	writeTestFile(t, filepath.Join(dependencyDir, "SKILL.md"), "# Closure skill\n", 0o644)

	_, err := ValidatePackageManifest(manifestPath)
	assertAdmissionError(t, err, "must declare dependency_lock, checksums, sbom, and provenance")
}

func TestProjectManifestForCurrentCatalogDemotesExpiredReadiness(t *testing.T) {
	verifiedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	manifestPath := filepath.Join(t.TempDir(), "tool.yaml")
	writeTestFile(t, manifestPath, v2ExecutableManifest("shared/stale-projection", verifiedAt), 0o644)
	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("parse historical manifest: %v", err)
	}

	entry := ProjectManifestForCurrentCatalog(manifest, verifiedAt.Add(executableEvidenceLifetime+time.Second))
	if entry.Usability.Availability != "not-verified" || entry.Usability.Source != "inferred" {
		t.Fatalf("expired readiness was still advertised: %#v", entry.Usability)
	}
	if entry.Verification == nil || entry.Verification.LastVerifiedAt != verifiedAt.Format(time.RFC3339) {
		t.Fatalf("historical verification was not preserved: %#v", entry.Verification)
	}
}

func TestBuildIndexRejectsUsableNowWithoutEvidence(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "shared", "unevidenced-tool")
	if err := os.MkdirAll(filepath.Join(entryDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := v2ExecutableManifest("shared/unevidenced-tool", time.Now().UTC())
	manifest = strings.Replace(manifest, "  evidence: [evidence://tests/v2-tool/smoke]\n", "  evidence: []\n", 1)
	writeTestFile(t, filepath.Join(entryDir, "tool.yaml"), manifest, 0o644)
	for _, file := range []string{"TOOL.md", "checksums.txt", "sbom.cdx.json", "bin/tool"} {
		writeTestFile(t, filepath.Join(entryDir, filepath.FromSlash(file)), "fixture\n", 0o755)
	}

	_, err := BuildToolsIndex(root)
	assertAdmissionError(t, err, "incomplete verification contract")
}

func TestBuildIndexRejectsMissingV2ArtifactMember(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "shared", "incomplete-artifact")
	if err := os.MkdirAll(filepath.Join(entryDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := v2ExecutableManifest("shared/incomplete-artifact", time.Now().UTC())
	writeTestFile(t, filepath.Join(entryDir, "tool.yaml"), manifest, 0o644)
	for _, file := range []string{"TOOL.md", "checksums.txt", "bin/tool"} {
		writeTestFile(t, filepath.Join(entryDir, filepath.FromSlash(file)), "fixture\n", 0o755)
	}

	_, err := BuildToolsIndex(root)
	assertAdmissionError(t, err, `artifact.sbom: declared path "sbom.cdx.json" does not exist`)
}

func TestBuildIndexRejectsUndeclaredRequiredAuthentication(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "shared", "auth-tool")
	if err := os.MkdirAll(filepath.Join(entryDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := v2ExecutableManifest("shared/auth-tool", time.Now().UTC())
	manifest = strings.Replace(manifest, "authentication:\n", "operational:\n  auth_required: [Provider account credential]\nauthentication:\n", 1)
	writeTestFile(t, filepath.Join(entryDir, "tool.yaml"), manifest, 0o644)
	for _, file := range []string{"TOOL.md", "checksums.txt", "sbom.cdx.json", "bin/tool"} {
		writeTestFile(t, filepath.Join(entryDir, filepath.FromSlash(file)), "fixture\n", 0o755)
	}

	_, err := BuildToolsIndex(root)
	assertAdmissionError(t, err, "requires authentication but declares authentication.status none")
}

func TestValidateRepositoryRejectsNonSelfContainedV2Bundle(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "plugins", "shared", "incomplete-bundle")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := strings.Replace(
		v2ExecutableManifest("shared/incomplete-bundle", time.Now().UTC()),
		"entrypoints:\n  spec: TOOL.md\n  scripts_dir: bin\nusability:\n  availability: usable-now\n  execution: local-tool\n  quickstart: bin/tool\nexecution:\n  kind: cli\n  command: [bin/tool]\n  smoke_test: [bin/tool, --smoke]\n",
		"entrypoints:\n  spec: plugin.json\nincludes: {}\nusability:\n  availability: template-only\n  execution: bundle\nexecution:\n  kind: bundle\n",
		1,
	)
	manifest = strings.Replace(manifest, "self_contained: true", "self_contained: false", 1)
	writeTestFile(t, filepath.Join(entryDir, "plugin.yaml"), manifest, 0o644)
	writeTestFile(t, filepath.Join(entryDir, "plugin.json"), "{}\n", 0o644)

	_, err := ValidateRepository(root)
	assertAdmissionError(t, err, "bundle artifact must declare a self-contained dependency closure")
}

func writeTestModule(t *testing.T, root, kind, id, runtime, extra string) string {
	t.Helper()
	directory, manifestName, specName := "skills", "skill.yaml", "SKILL.md"
	switch kind {
	case "agent":
		directory, manifestName, specName = "agents", "agent.yaml", "AGENT.md"
	case "tool":
		directory, manifestName, specName = "tools-mcp", "tool.yaml", "TOOL.md"
	case "plugin":
		directory, manifestName, specName = "plugins", "plugin.yaml", "plugin.json"
	}
	availability, execution := "documentation-only", "instructions"
	if kind == "agent" {
		availability, execution = "template-only", "orchestrator"
	} else if kind == "plugin" {
		availability, execution = "template-only", "bundle"
	} else if kind == "tool" {
		availability, execution = "template-only", "integration-template"
	}
	usability := fmt.Sprintf("usability:\n  availability: %s\n  execution: %s\n", availability, execution)
	if strings.Contains(extra, "usability:") {
		usability = ""
	}
	entryDir := filepath.Join(root, directory, filepath.FromSlash(id))
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir module: %v", err)
	}
	manifest := fmt.Sprintf(`id: %s
name: Test Module
description: Complete test module used for repository admission tests.
version: 1.0.0
released_at: "2026-09-13T00:00:00Z"
category: test/modules
tags: [test]
runtimes: [%s]
entrypoints:
  spec: %s
%s%sdeprecated: false
`, id, runtime, specName, extra, usability)
	path := filepath.Join(entryDir, manifestName)
	writeTestFile(t, path, manifest, 0o644)
	writeTestFile(t, filepath.Join(entryDir, specName), "# Test module\n", 0o644)
	return path
}

func v2ExecutableManifest(id string, verifiedAt time.Time) string {
	return fmt.Sprintf(`schema_version: "2.0"
id: %s
name: V2 Test Tool
description: V2 executable used for repository admission tests.
version: 1.0.0
released_at: "2026-09-13T00:00:00Z"
category: tools-mcp/test
tags: [test]
license: MIT
author:
  name: Test Maintainer
runtimes: [codex]
entrypoints:
  spec: TOOL.md
  scripts_dir: bin
usability:
  availability: usable-now
  execution: local-tool
  quickstart: bin/tool
execution:
  kind: cli
  command: [bin/tool]
  smoke_test: [bin/tool, --smoke]
  supported_platforms: [linux]
  supported_runtimes: [native]
artifact:
  self_contained: true
  dependency_lock: null
  checksums: checksums.txt
  sbom: sbom.cdx.json
authentication:
  status: none
  methods: [none]
  credential_bindings: []
  scopes: []
  credential_storage: No credentials.
  validation: Confirm no authentication challenge.
  revocation: Not applicable.
verification:
  evidence: [evidence://tests/v2-tool/smoke]
  last_verified_at: %q
deprecated: false
`, id, verifiedAt.Format(time.RFC3339))
}

func v2PluginManifest(id string, verifiedAt time.Time) string {
	return fmt.Sprintf(`schema_version: "2.1"
id: %s
name: Closure Plugin
description: Self-contained plugin used to verify exact archived dependency closure.
version: 1.0.0
released_at: "2026-09-14T00:00:00Z"
category: engineering-plugins/maintenance
tags: [testing]
license: MIT
author:
  name: Test Maintainer
runtimes: [generic]
entrypoints:
  spec: plugin.json
includes:
  skills: [engineering/closure-skill]
usability:
  availability: usable-now
  execution: bundle
execution:
  kind: bundle
  supported_platforms: [linux]
  supported_runtimes: [generic]
artifact:
  self_contained: true
  dependency_lock: null
  checksums: null
  sbom: null
authentication:
  status: none
  methods: [none]
  credential_bindings: []
  scopes: []
  credential_storage: No credentials.
  validation: Confirm no authentication challenge.
  revocation: Not applicable.
verification:
  evidence: [evidence://tests/closure-plugin]
  last_verified_at: %q
deprecated: false
`, id, verifiedAt.Format(time.RFC3339))
}

func appendTestManifest(t *testing.T, path, replacement, old string) {
	t.Helper()
	replaceTestManifest(t, path, old, old+replacement)
}

func replaceTestManifest(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	updated := strings.Replace(string(data), old, replacement, 1)
	if updated == string(data) {
		t.Fatalf("manifest replacement did not match %q", old)
	}
	writeTestFile(t, path, updated, 0o644)
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir file parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertAdmissionError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}
