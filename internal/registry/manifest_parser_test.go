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
