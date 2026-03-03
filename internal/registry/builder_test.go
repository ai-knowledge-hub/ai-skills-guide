package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if !strings.Contains(entry.Versions[0].ManifestURL, "/tools-mcp/analytics/demo-tool/tool.yaml") {
		t.Fatalf("unexpected manifest url: %s", entry.Versions[0].ManifestURL)
	}
	if !strings.Contains(entry.Versions[0].ArtifactURL, "/artifacts/analytics/demo-tool/0.1.0.tar.gz") {
		t.Fatalf("unexpected artifact url: %s", entry.Versions[0].ArtifactURL)
	}
}
