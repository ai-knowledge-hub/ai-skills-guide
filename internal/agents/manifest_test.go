package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAgentManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yaml")
	content := `id: marketing/demo-supervisor
name: Demo Supervisor
description: Demo agent.
version: 0.1.0
released_at: "2026-03-03T00:00:00Z"
category: marketing-agents/performance
tags:
  - demo
runtimes:
  - codex
dependencies:
  skills:
    - adtech/dashboard-generator
  tools:
    - ga4_query
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	m, err := ParseAgentManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.ID != "marketing/demo-supervisor" {
		t.Fatalf("unexpected id: %s", m.ID)
	}
	if len(m.DependencySkills) != 1 || m.DependencySkills[0] != "adtech/dashboard-generator" {
		t.Fatalf("unexpected skills deps: %#v", m.DependencySkills)
	}
	if len(m.DependencyTools) != 1 || m.DependencyTools[0] != "ga4_query" {
		t.Fatalf("unexpected tools deps: %#v", m.DependencyTools)
	}
}
