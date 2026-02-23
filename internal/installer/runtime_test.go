package installer

import (
	"path/filepath"
	"testing"
)

func TestResolveRuntimeTargetExplicitTargetWins(t *testing.T) {
	target, err := ResolveRuntimeTarget("codex", "./tmp-target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(target.TargetPath) {
		t.Fatalf("expected absolute target path, got: %s", target.TargetPath)
	}
	if target.Runtime != "codex" {
		t.Fatalf("unexpected runtime: %s", target.Runtime)
	}
}

func TestResolveRuntimeTargetCodexFromEnv(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")
	target, err := ResolveRuntimeTarget("codex", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/codex-home", "skills")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetClaudeFromEnv(t *testing.T) {
	t.Setenv("CLAUDE_HOME", "/tmp/claude-home")
	target, err := ResolveRuntimeTarget("claude", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/claude-home", "skills")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetGenericNeedsTarget(t *testing.T) {
	_, err := ResolveRuntimeTarget("generic", "")
	if err == nil {
		t.Fatalf("expected error for missing target")
	}
}
