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
	if entry.Dependencies == nil || len(entry.Dependencies.Skills) != 1 {
		t.Fatalf("expected dependencies in index, got %#v", entry.Dependencies)
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
