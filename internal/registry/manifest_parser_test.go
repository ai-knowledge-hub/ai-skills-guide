package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	content := `id: marketing/example-skill
name: Example Skill
description: First sentence,
  second sentence.
version: 0.1.0
released_at: "2026-02-23T00:00:00Z"
category: marketing-tools/ads-ops
tags:
  - one
  - two
license: MIT
author:
  name: Example
runtimes:
  - codex
  - generic
verification:
  security_reviewed: true
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.ID != "marketing/example-skill" {
		t.Fatalf("unexpected id: %s", m.ID)
	}
	if m.Description != "First sentence, second sentence." {
		t.Fatalf("unexpected description: %s", m.Description)
	}
	if m.Category != "marketing-tools/ads-ops" {
		t.Fatalf("unexpected category: %s", m.Category)
	}
	if len(m.Tags) != 2 || m.Tags[0] != "one" || m.Tags[1] != "two" {
		t.Fatalf("unexpected tags: %#v", m.Tags)
	}
	if len(m.Runtimes) != 2 || m.Runtimes[0] != "codex" {
		t.Fatalf("unexpected runtimes: %#v", m.Runtimes)
	}
	if !m.SecurityReviewed {
		t.Fatal("expected security reviewed to be true")
	}
}

func TestParseToolManifestOperationalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.yaml")
	content := `id: analytics/example-tool
name: Example Tool
description: Example tool manifest for operational metadata parsing.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: tools-mcp/analytics
tags:
  - analytics
license: MIT
author:
  name: Example
runtimes:
  - codex
dependencies:
  mcp_servers:
    - ga4
operational:
  connected_system: Google Analytics 4
  capabilities:
    - Pull sessions by dimension.
    - Return normalized rows for dashboard input.
  auth_required:
    - GA4 property access via authenticated MCP runtime
  access_level: read-only
  trust_boundary: remote-mcp-server
  approval_boundary: Safe for read-only analytics retrieval only.
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Operational.ConnectedSystem != "Google Analytics 4" {
		t.Fatalf("unexpected connected system: %s", m.Operational.ConnectedSystem)
	}
	if len(m.Operational.Capabilities) != 2 {
		t.Fatalf("unexpected capabilities: %#v", m.Operational.Capabilities)
	}
	if len(m.Operational.AuthRequired) != 1 {
		t.Fatalf("unexpected auth requirements: %#v", m.Operational.AuthRequired)
	}
	if m.Operational.AccessLevel != "read-only" {
		t.Fatalf("unexpected access level: %s", m.Operational.AccessLevel)
	}
	if m.Operational.TrustBoundary != "remote-mcp-server" {
		t.Fatalf("unexpected trust boundary: %s", m.Operational.TrustBoundary)
	}
	if len(m.Dependencies.MCPServers) != 1 || m.Dependencies.MCPServers[0] != "ga4" {
		t.Fatalf("unexpected dependencies: %#v", m.Dependencies)
	}
}

func TestParseAgentManifestOperationalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yaml")
	content := `id: marketing/example-agent
name: Example Agent
description: Example agent manifest for operational metadata parsing.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: marketing-agents/performance
tags:
  - reporting
license: MIT
author:
  name: Example
runtimes:
  - codex
dependencies:
  agents:
    - marketing/creative-operating-system-supervisor
  skills:
    - adtech/dashboard-generator
  tools:
    - ga4_query
operational:
  role: Weekly reporting supervisor.
  coordinates:
    - Dashboard generation
    - QA gating
  autonomy_level: semi-autonomous
  approval_boundary: Require approval before external distribution.
  outputs:
    - Publish decision
    - Executive narrative
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Operational.Role != "Weekly reporting supervisor." {
		t.Fatalf("unexpected role: %s", m.Operational.Role)
	}
	if m.Operational.AutonomyLevel != "semi-autonomous" {
		t.Fatalf("unexpected autonomy level: %s", m.Operational.AutonomyLevel)
	}
	if len(m.Operational.Coordinates) != 2 {
		t.Fatalf("unexpected coordinates: %#v", m.Operational.Coordinates)
	}
	if len(m.Operational.Outputs) != 2 {
		t.Fatalf("unexpected outputs: %#v", m.Operational.Outputs)
	}
	if len(m.Dependencies.Agents) != 1 || m.Dependencies.Agents[0] != "marketing/creative-operating-system-supervisor" {
		t.Fatalf("unexpected agent dependencies: %#v", m.Dependencies)
	}
	if len(m.Dependencies.Skills) != 1 || m.Dependencies.Skills[0] != "adtech/dashboard-generator" {
		t.Fatalf("unexpected skill dependencies: %#v", m.Dependencies)
	}
}

func TestParseSkillManifestOperationalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	content := `id: engineering/example-skill
name: Example Skill
description: Example skill manifest for operational metadata parsing.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: engineering/code-maintenance
tags:
  - planning
license: MIT
author:
  name: Example
runtimes:
  - codex
dependencies:
  tools:
    - rg
operational:
  use_when: Use when repo scope analysis is needed before edits.
  execution_mode: read-only-local-inspection
  outputs:
    - Change strategy
    - Verification command list
  approval_boundary: Safe for inspection and planning before edits.
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Operational.UseWhen != "Use when repo scope analysis is needed before edits." {
		t.Fatalf("unexpected use_when: %s", m.Operational.UseWhen)
	}
	if m.Operational.ExecutionMode != "read-only-local-inspection" {
		t.Fatalf("unexpected execution mode: %s", m.Operational.ExecutionMode)
	}
	if len(m.Operational.Outputs) != 2 {
		t.Fatalf("unexpected outputs: %#v", m.Operational.Outputs)
	}
	if len(m.Dependencies.Tools) != 1 || m.Dependencies.Tools[0] != "rg" {
		t.Fatalf("unexpected tool dependencies: %#v", m.Dependencies)
	}
}

func TestParsePluginManifestIncludesAndRequires(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `id: marketing/example-plugin
name: Example Plugin
description: Example plugin package for manifest parsing tests.
version: 0.1.0
released_at: "2026-03-28T00:00:00Z"
category: marketing-plugins/reporting
tags:
  - reporting
license: MIT
author:
  name: Example
runtimes:
  - codex
  - claude
entrypoints:
  spec: plugin.json
includes:
  skills:
    - marketing/meta-google-weekly-performance-review
  agents:
    - marketing/weekly-performance-supervisor
  tools:
    - analytics/ga4-mcp-connector
  hooks:
    - post-analysis-slack-summary
requires:
  secrets:
    - GA4_PROPERTY_ID
  approvals:
    - human-review-for-write-actions
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(m.Includes.Skills) != 1 || m.Includes.Skills[0] != "marketing/meta-google-weekly-performance-review" {
		t.Fatalf("unexpected included skills: %#v", m.Includes.Skills)
	}
	if len(m.Includes.Agents) != 1 || m.Includes.Agents[0] != "marketing/weekly-performance-supervisor" {
		t.Fatalf("unexpected included agents: %#v", m.Includes.Agents)
	}
	if len(m.Includes.Tools) != 1 || m.Includes.Tools[0] != "analytics/ga4-mcp-connector" {
		t.Fatalf("unexpected included tools: %#v", m.Includes.Tools)
	}
	if len(m.Includes.Hooks) != 1 || m.Includes.Hooks[0] != "post-analysis-slack-summary" {
		t.Fatalf("unexpected included hooks: %#v", m.Includes.Hooks)
	}
	if len(m.Requires.Secrets) != 1 || m.Requires.Secrets[0] != "GA4_PROPERTY_ID" {
		t.Fatalf("unexpected required secrets: %#v", m.Requires.Secrets)
	}
	if len(m.Requires.Approvals) != 1 || m.Requires.Approvals[0] != "human-review-for-write-actions" {
		t.Fatalf("unexpected required approvals: %#v", m.Requires.Approvals)
	}
}
