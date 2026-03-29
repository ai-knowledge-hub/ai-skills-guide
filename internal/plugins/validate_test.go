package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePluginSuccess(t *testing.T) {
	root := t.TempDir()
	pluginsRoot := filepath.Join(root, "plugins")
	skillsRoot := filepath.Join(root, "skills")
	agentsRoot := filepath.Join(root, "agents")
	toolsRoot := filepath.Join(root, "tools-mcp")

	mustMkdirAll(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "examples"))
	mustMkdirAll(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "tests"))
	mustMkdirAll(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "hooks"))
	mustMkdirAll(t, filepath.Join(skillsRoot, "marketing", "demo-skill"))
	mustMkdirAll(t, filepath.Join(agentsRoot, "marketing", "demo-agent"))
	mustMkdirAll(t, filepath.Join(toolsRoot, "analytics", "demo-tool"))

	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "README.md"), "# Demo\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "plugin.json"), "{\"name\":\"demo\"}\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "examples", "overview.md"), "# Example\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "tests", "test-prompts.md"), "1. One\n2. Two\n3. Three\n4. Four\n5. Five\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "hooks", "notify.md"), "# Hook\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "plugin.yaml"), `id: marketing/demo-plugin
name: Demo Plugin
description: Demo plugin manifest used for plugin validation tests.
version: 0.1.0
released_at: "2026-03-29T00:00:00Z"
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
    - marketing/demo-skill
  agents:
    - marketing/demo-agent
  tools:
    - analytics/demo-tool
  hooks:
    - notify
deprecated: false
`)

	issues, err := Validate(ValidateOptions{
		PluginsRoot: pluginsRoot,
		SkillsRoot:  skillsRoot,
		AgentsRoot:  agentsRoot,
		ToolsRoot:   toolsRoot,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %#v", issues)
	}
}

func TestValidatePluginReportsMissingDependenciesAndHooks(t *testing.T) {
	root := t.TempDir()
	pluginsRoot := filepath.Join(root, "plugins")
	mustMkdirAll(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "examples"))
	mustMkdirAll(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "tests"))
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "README.md"), "# Demo\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "plugin.json"), "{\"name\":\"demo\"}\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "examples", "overview.md"), "# Example\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "tests", "test-prompts.md"), "1. One\n2. Two\n3. Three\n4. Four\n5. Five\n")
	mustWriteFile(t, filepath.Join(pluginsRoot, "marketing", "demo-plugin", "plugin.yaml"), `id: marketing/demo-plugin
name: Demo Plugin
description: Demo plugin manifest used for plugin validation tests.
version: 0.1.0
released_at: "2026-03-29T00:00:00Z"
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
    - marketing/missing-skill
  agents:
    - marketing/missing-agent
  tools:
    - analytics/missing-tool
  hooks:
    - missing-hook
deprecated: false
`)

	issues, err := Validate(ValidateOptions{
		PluginsRoot: pluginsRoot,
		SkillsRoot:  filepath.Join(root, "skills"),
		AgentsRoot:  filepath.Join(root, "agents"),
		ToolsRoot:   filepath.Join(root, "tools-mcp"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rendered := make([]string, 0, len(issues))
	for _, issue := range issues {
		rendered = append(rendered, issue.Message)
	}
	expected := []string{
		"missing bundled skill: marketing/missing-skill",
		"missing bundled agent: marketing/missing-agent",
		"missing bundled tool: analytics/missing-tool",
		"missing hooks/ directory",
	}
	for _, needle := range expected {
		if !contains(rendered, needle) {
			t.Fatalf("expected issue %q, got %#v", needle, rendered)
		}
	}
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
