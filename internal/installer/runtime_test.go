package installer

import (
	"os"
	"path/filepath"
	"strings"
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

func TestResolveRuntimeTargetForModuleAgents(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")
	target, err := ResolveRuntimeTargetForModule("codex", "agents", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/codex-home", "agents")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetForModuleToolsMcp(t *testing.T) {
	t.Setenv("CLAUDE_HOME", "/tmp/claude-home")
	target, err := ResolveRuntimeTargetForModule("claude", "tools", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/claude-home", "tools-mcp")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetForModulePlugins(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")
	target, err := ResolveRuntimeTargetForModule("codex", "plugins", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/codex-home", "plugins")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestPreparePluginRuntimeArtifactsForCodex(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte("{\n  \"name\": \"demo-plugin\",\n  \"version\": \"0.1.0\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}

	artifacts, err := PreparePluginRuntimeArtifacts(dir, "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one runtime artifact, got %d", len(artifacts))
	}
	if !strings.HasSuffix(artifacts[0], filepath.Join(".codex-plugin", "plugin.json")) {
		t.Fatalf("unexpected artifact path: %s", artifacts[0])
	}
	rendered, err := os.ReadFile(artifacts[0])
	if err != nil {
		t.Fatalf("read generated artifact: %v", err)
	}
	if !strings.Contains(string(rendered), "\"runtime\": \"codex\"") {
		t.Fatalf("expected runtime field in generated artifact, got:\n%s", string(rendered))
	}
}

func TestPreparePluginRuntimeArtifactsForClaude(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte("{\n  \"name\": \"demo-plugin\",\n  \"version\": \"0.1.0\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}

	artifacts, err := PreparePluginRuntimeArtifacts(dir, "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one runtime artifact, got %d", len(artifacts))
	}
	if !strings.HasSuffix(artifacts[0], filepath.Join(".claude-plugin", "plugin.json")) {
		t.Fatalf("unexpected artifact path: %s", artifacts[0])
	}
	rendered, err := os.ReadFile(artifacts[0])
	if err != nil {
		t.Fatalf("read generated artifact: %v", err)
	}
	if !strings.Contains(string(rendered), "\"runtime\": \"claude\"") {
		t.Fatalf("expected runtime field in generated artifact, got:\n%s", string(rendered))
	}
}

func TestPreparePluginRuntimeArtifactsForGenericSkips(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte("{\"name\":\"demo-plugin\"}\n"), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}

	artifacts, err := PreparePluginRuntimeArtifacts(dir, "generic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("expected no artifacts for generic runtime, got %d", len(artifacts))
	}
}
