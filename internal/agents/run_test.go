package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPreflightReady(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agents", "marketing", "demo")
	skillDir := filepath.Join(root, "skills", "adtech", "dashboard-generator")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte("## Workflow\n1. Step A\n2. Step B\n"), 0o644); err != nil {
		t.Fatalf("write AGENT.md: %v", err)
	}
	manifest := `id: marketing/demo
name: Demo
description: Demo.
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
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	bindingsPath := filepath.Join(root, "bindings.json")
	if err := os.WriteFile(bindingsPath, []byte(`{"tools":{"ga4_query":{"endpoint":"mcp://ga4","mode":"read_only"}}}`), 0o644); err != nil {
		t.Fatalf("write bindings: %v", err)
	}
	report, err := RunPreflight(RunOptions{
		AgentsRoot:     filepath.Join(root, "agents"),
		SkillsRoot:     filepath.Join(root, "skills"),
		ToolsRoot:      filepath.Join(root, "tools-mcp"),
		AgentID:        "marketing/demo",
		BindingsPath:   bindingsPath,
		ApproveLive:    true,
		GovernancePath: "",
	})
	if err != nil {
		t.Fatalf("run preflight: %v", err)
	}
	if report.Status != "ready" {
		t.Fatalf("expected ready, got %s (%v)", report.Status, report.BlockingReasons)
	}
	if len(report.WorkflowSteps) != 2 {
		t.Fatalf("expected workflow steps, got %#v", report.WorkflowSteps)
	}
}
