package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSkillsIndex(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "skills", "engineering", "demo-skill")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mf := `id: engineering/demo-skill
name: Demo Skill
description: Demo skill manifest used for index generation tests.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: engineering/code-maintenance
tags:
  - planning
license: MIT
author:
  name: Tests
runtimes:
  - codex
entrypoints:
  skill_md: SKILL.md
dependencies:
  tools:
    - rg
operational:
  use_when: Use when repo planning is needed.
  execution_mode: read-only-local-inspection
  outputs:
    - Change strategy
  approval_boundary: Safe for planning before edits.
deprecated: false
`
	if err := os.WriteFile(filepath.Join(entryDir, "skill.yaml"), []byte(mf), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "SKILL.md"), []byte("---\n# Demo\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildSkillsIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(idx.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Skills))
	}
	entry := idx.Skills[0]
	if entry.Operational == nil || entry.Operational.UseWhen != "Use when repo planning is needed." {
		t.Fatalf("expected operational metadata in index, got %#v", entry.Operational)
	}
	if entry.Dependencies == nil || len(entry.Dependencies.Tools) != 1 {
		t.Fatalf("expected dependencies in index, got %#v", entry.Dependencies)
	}
	if entry.Usability.Availability != "usable-now" || entry.Usability.Execution != "instructions" || entry.Usability.Source != "inferred" {
		t.Fatalf("unexpected default usability: %#v", entry.Usability)
	}
}

func TestBuildAgentsIndex(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "agents", "marketing", "demo-agent")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mf := `id: marketing/demo-agent
name: Demo Agent
description: Demo agent manifest used for index generation tests.
version: 0.1.0
released_at: "2026-03-02T00:00:00Z"
category: marketing-agents/performance
tags:
  - demo
license: MIT
author:
  name: Tests
runtimes:
  - codex
entrypoints:
  spec: AGENT.md
dependencies:
  agents:
    - marketing/creative-operating-system-supervisor
  skills:
    - adtech/dashboard-generator
  tools:
    - ga4_query
operational:
  role: Demo supervisor.
  coordinates:
    - Reporting
  autonomy_level: semi-autonomous
  approval_boundary: Approval required before publish.
  outputs:
    - Publish decision
deprecated: false
`
	if err := os.WriteFile(filepath.Join(entryDir, "agent.yaml"), []byte(mf), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "AGENT.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildAgentsIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(idx.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Skills))
	}
	entry := idx.Skills[0]
	if entry.ID != "marketing/demo-agent" {
		t.Fatalf("unexpected id: %s", entry.ID)
	}
	if entry.Operational == nil || entry.Operational.Role != "Demo supervisor." {
		t.Fatalf("expected operational metadata in index, got %#v", entry.Operational)
	}
	if entry.Dependencies == nil || len(entry.Dependencies.Agents) != 1 || len(entry.Dependencies.Skills) != 1 {
		t.Fatalf("expected agent and skill dependencies in index, got %#v", entry.Dependencies)
	}
	if entry.Usability.Availability != "setup-required" || entry.Usability.Execution != "orchestrator" {
		t.Fatalf("unexpected agent usability: %#v", entry.Usability)
	}
	if !strings.Contains(entry.Versions[0].ManifestURL, "/agents/marketing/demo-agent/agent.yaml") {
		t.Fatalf("unexpected manifest url: %s", entry.Versions[0].ManifestURL)
	}
	if !strings.Contains(entry.Versions[0].ArtifactURL, "/artifacts/marketing/demo-agent/0.1.0.tar.gz") {
		t.Fatalf("unexpected artifact url: %s", entry.Versions[0].ArtifactURL)
	}
}

func TestBuildToolsIndex(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "analytics", "demo-tool")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mf := `id: analytics/demo-tool
name: Demo Tool
description: Demo tool manifest used for index generation tests.
version: 0.1.0
released_at: "2026-03-02T00:00:00Z"
category: tools-mcp/analytics
tags:
  - demo
license: MIT
author:
  name: Tests
runtimes:
  - codex
entrypoints:
  spec: TOOL.md
dependencies:
  mcp_servers:
    - demo-mcp
operational:
  connected_system: Demo Analytics
  capabilities:
    - Pull demo rows.
  auth_required:
    - Demo auth
  access_level: read-only
  trust_boundary: remote-mcp-server
  approval_boundary: Safe for demo read-only queries.
deprecated: false
`
	if err := os.WriteFile(filepath.Join(entryDir, "tool.yaml"), []byte(mf), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "TOOL.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildToolsIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(idx.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Skills))
	}
	entry := idx.Skills[0]
	if entry.ID != "analytics/demo-tool" {
		t.Fatalf("unexpected id: %s", entry.ID)
	}
	if entry.Operational == nil || entry.Operational.ConnectedSystem != "Demo Analytics" {
		t.Fatalf("expected operational metadata in index, got %#v", entry.Operational)
	}
	if entry.Dependencies == nil || len(entry.Dependencies.MCPServers) != 1 {
		t.Fatalf("expected dependencies in index, got %#v", entry.Dependencies)
	}
	if !strings.Contains(entry.Versions[0].ManifestURL, "/tools-mcp/analytics/demo-tool/tool.yaml") {
		t.Fatalf("unexpected manifest url: %s", entry.Versions[0].ManifestURL)
	}
	if !strings.Contains(entry.Versions[0].ArtifactURL, "/artifacts/analytics/demo-tool/0.1.0.tar.gz") {
		t.Fatalf("unexpected artifact url: %s", entry.Versions[0].ArtifactURL)
	}
}

func TestBuildPluginsIndex(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "plugins", "marketing", "demo-plugin")
	if err := os.MkdirAll(filepath.Join(entryDir, "examples"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mf := `id: marketing/demo-plugin
name: Demo Plugin
description: Demo plugin manifest used for index generation tests.
version: 0.1.0
released_at: "2026-03-28T00:00:00Z"
category: marketing-plugins/reporting
tags:
  - demo
license: MIT
author:
  name: Tests
runtimes:
  - codex
entrypoints:
  spec: plugin.json
includes:
  skills:
    - marketing/meta-google-weekly-performance-review
requires:
  secrets:
    - GA4_PROPERTY_ID
deprecated: false
`
	if err := os.WriteFile(filepath.Join(entryDir, "plugin.yaml"), []byte(mf), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "plugin.json"), []byte("{\"name\":\"demo\"}\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildPluginsIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(idx.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Skills))
	}
	entry := idx.Skills[0]
	if entry.ID != "marketing/demo-plugin" {
		t.Fatalf("unexpected id: %s", entry.ID)
	}
	if entry.Includes == nil || len(entry.Includes.Skills) != 1 {
		t.Fatalf("expected included skills in index, got %#v", entry.Includes)
	}
	if entry.Requires == nil || len(entry.Requires.Secrets) != 1 {
		t.Fatalf("expected required secrets in index, got %#v", entry.Requires)
	}
	if !strings.Contains(entry.Versions[0].ManifestURL, "/plugins/marketing/demo-plugin/plugin.yaml") {
		t.Fatalf("unexpected manifest url: %s", entry.Versions[0].ManifestURL)
	}
}

func TestDigestSkillDirIgnoresPythonCache(t *testing.T) {
	root := t.TempDir()
	scriptDir := filepath.Join(root, "scripts")
	cacheDir := filepath.Join(scriptDir, "__pycache__")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	sourcePath := filepath.Join(scriptDir, "reconcile.py")
	if err := os.WriteFile(sourcePath, []byte("print('source')\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	before, err := digestSkillDir(root)
	if err != nil {
		t.Fatalf("digest before cache: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(cacheDir, "reconcile.cpython-314.pyc"),
		[]byte("runtime-specific bytecode"),
		0o644,
	); err != nil {
		t.Fatalf("write pycache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "temporary.pyo"), []byte("optimized bytecode"), 0o644); err != nil {
		t.Fatalf("write pyo: %v", err)
	}

	afterCache, err := digestSkillDir(root)
	if err != nil {
		t.Fatalf("digest after cache: %v", err)
	}
	if before != afterCache {
		t.Fatalf("python cache changed digest: before=%s after=%s", before, afterCache)
	}

	if err := os.WriteFile(sourcePath, []byte("print('changed source')\n"), 0o644); err != nil {
		t.Fatalf("update source: %v", err)
	}
	afterSource, err := digestSkillDir(root)
	if err != nil {
		t.Fatalf("digest after source change: %v", err)
	}
	if afterSource == before {
		t.Fatal("tracked source change did not alter digest")
	}
}

func TestBuildIndexPreservesV2Contracts(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "adtech", "v2-tool")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mf := `schema_version: "2.0"
id: adtech/v2-tool
name: V2 Tool
description: V2 tool manifest used for registry preservation testing.
version: 1.0.0
released_at: "2026-09-13T00:00:00Z"
category: tools-mcp/adtech
tags:
  - testing
license: MIT
author:
  name: Tests
runtimes:
  - codex
entrypoints:
  spec: TOOL.md
execution:
  kind: cli
  command:
    - bin/v2-tool
  smoke_test:
    - bin/v2-tool
    - --smoke
  supported_platforms:
    - linux
  supported_runtimes:
    - native
artifact:
  self_contained: true
  dependency_lock: null
  checksums: checksums.txt
  sbom: sbom.cdx.json
authentication:
  status: none
  methods:
    - none
  credential_bindings: []
  scopes: []
  credential_storage: No credentials are used.
  validation: Confirm no credentials are requested.
  revocation: No revocation applies.
verification:
  evidence:
    - evidence://tests/v2-tool/smoke
  last_verified_at: "2026-09-13T00:00:00Z"
deprecated: false
`
	if err := os.WriteFile(filepath.Join(entryDir, "tool.yaml"), []byte(mf), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "TOOL.md"), []byte("# V2 Tool\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildToolsIndex(root)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	entry := idx.Skills[0]
	if idx.RegistryVersion != "1.1" || entry.SchemaVersion != "2.0" {
		t.Fatalf("unexpected registry versions: %#v", entry)
	}
	if entry.Execution == nil || entry.Execution.Kind != "cli" || entry.Artifact == nil {
		t.Fatalf("execution and artifact contracts were not preserved: %#v", entry)
	}
	if entry.Authentication == nil || entry.Authentication.Status != "none" || entry.Verification == nil {
		t.Fatalf("authentication and verification contracts were not preserved: %#v", entry)
	}
	if entry.Authentication.CredentialBindings == nil || entry.Authentication.Scopes == nil {
		t.Fatalf("empty authentication arrays must remain arrays: %#v", entry.Authentication)
	}
}

func TestSchemaValidFlowSequencesRoundTripToRegistry(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "adtech", "flow-sequence-fixture")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fixturePath := filepath.Join(
		"..", "..", "shared", "schemas", "fixtures", "manifest-v2",
		"tool-flow-sequences.valid.yaml",
	)
	manifest, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read schema-valid fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "tool.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "TOOL.md"), []byte("# Flow fixture\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildToolsIndex(root)
	if err != nil {
		t.Fatalf("build index from schema-valid flow YAML: %v", err)
	}
	entry := idx.Skills[0]
	if len(entry.Runtimes) != 2 || entry.Execution == nil {
		t.Fatalf("flow sequences were not preserved: %#v", entry)
	}
	if len(entry.Execution.Command) != 2 || len(entry.Execution.SupportedPlatforms) != 3 {
		t.Fatalf("execution flow sequences were not preserved: %#v", entry.Execution)
	}
	if entry.Authentication == nil || len(entry.Authentication.Methods) != 1 || entry.Verification == nil || len(entry.Verification.Evidence) != 1 {
		t.Fatalf("contract flow sequences were not preserved: %#v", entry)
	}
}

func TestBuildIndexRejectsUnversionedV2Contract(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "adtech", "unversioned-v2-tool")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "tool.yaml"), []byte(unversionedV2ToolManifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	idx, err := BuildToolsIndex(root)
	if err == nil {
		t.Fatalf("expected builder rejection, got index %#v", idx)
	}
	if !strings.Contains(err.Error(), "declares v2 contract field $.execution without schema_version") {
		t.Fatalf("unexpected builder error: %v", err)
	}
}

func TestBuildIndexRejectsCommonAPIKeyFormat(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "adtech", "credential-bearing-tool")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	credential := "sk_" + "live_51ReviewFixtureAbCdEf123456789"
	manifest := "schema_version: \"2.0\"\n" + unversionedV2ToolManifest
	manifest = strings.Replace(
		manifest,
		"validation: Validate the configured identity.",
		"validation: "+credential,
		1,
	)
	if err := os.WriteFile(filepath.Join(entryDir, "tool.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	idx, err := BuildToolsIndex(root)
	if err == nil {
		t.Fatalf("expected builder rejection, got index %#v", idx)
	}
	if !strings.Contains(err.Error(), "$.authentication.validation") {
		t.Fatalf("unexpected builder error: %v", err)
	}
	if strings.Contains(err.Error(), credential) {
		t.Fatal("credential-shaped value was echoed in the builder error")
	}
}

func TestBuildIndexAllowsBenignCredentialSentinel(t *testing.T) {
	root := t.TempDir()
	entryDir := filepath.Join(root, "tools-mcp", "adtech", "sentinel-tool")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := "schema_version: \"2.0\"\n" + unversionedV2ToolManifest
	manifest = strings.Replace(
		manifest,
		"validation: Validate the configured identity.",
		"validation: token=not-applicable",
		1,
	)
	if err := os.WriteFile(filepath.Join(entryDir, "tool.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(entryDir, "TOOL.md"), []byte("# Sentinel Tool\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	idx, err := BuildToolsIndex(root)
	if err != nil {
		t.Fatalf("build index with benign credential sentinel: %v", err)
	}
	if len(idx.Skills) != 1 || idx.Skills[0].Authentication == nil {
		t.Fatalf("expected one v2 registry entry, got %#v", idx.Skills)
	}
	if got := idx.Skills[0].Authentication.Validation; got != "token=not-applicable" {
		t.Fatalf("benign guidance was not preserved in registry: %q", got)
	}
}
