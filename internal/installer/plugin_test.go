package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestInstallPluginDependenciesInstallsReferencedModulesAndHooks(t *testing.T) {
	root := t.TempDir()

	skillsRoot := filepath.Join(root, "skills")
	agentsRoot := filepath.Join(root, "agents")
	toolsRoot := filepath.Join(root, "tools-mcp")
	pluginTarget := filepath.Join(root, "runtime", "plugins")

	mustMkdirAll(t, filepath.Join(skillsRoot, "marketing", "demo-skill"))
	mustMkdirAll(t, filepath.Join(agentsRoot, "marketing", "demo-agent"))
	mustMkdirAll(t, filepath.Join(toolsRoot, "analytics", "demo-tool"))
	mustMkdirAll(t, filepath.Join(pluginTarget, "marketing", "demo-plugin", "hooks"))

	mustWriteFile(t, filepath.Join(skillsRoot, "marketing", "demo-skill", "SKILL.md"), "# Demo skill\n")
	mustWriteFile(t, filepath.Join(agentsRoot, "marketing", "demo-agent", "AGENT.md"), "# Demo agent\n")
	mustWriteFile(t, filepath.Join(toolsRoot, "analytics", "demo-tool", "TOOL.md"), "# Demo tool\n")
	mustWriteFile(t, filepath.Join(pluginTarget, "marketing", "demo-plugin", "hooks", "notify.md"), "# Hook\n")

	skillsIndexPath := filepath.Join(root, "registry", "skills-index.json")
	agentsIndexPath := filepath.Join(root, "registry", "agents-index.json")
	toolsIndexPath := filepath.Join(root, "registry", "tools-index.json")
	mustWriteIndex(t, skillsIndexPath, registry.Index{Skills: []registry.SkillEntry{{ID: "marketing/demo-skill"}}})
	mustWriteIndex(t, agentsIndexPath, registry.Index{Skills: []registry.SkillEntry{{ID: "marketing/demo-agent"}}})
	mustWriteIndex(t, toolsIndexPath, registry.Index{Skills: []registry.SkillEntry{{ID: "analytics/demo-tool"}}})

	entry := registry.SkillEntry{
		ID: "marketing/demo-plugin",
		Includes: &registry.IncludeSet{
			Skills: []string{"marketing/demo-skill"},
			Agents: []string{"marketing/demo-agent"},
			Tools:  []string{"analytics/demo-tool"},
			Hooks:  []string{"notify"},
		},
	}

	result, err := InstallPluginDependencies(
		entry,
		"generic",
		pluginTarget,
		map[string]string{
			"skills": skillsRoot,
			"agents": agentsRoot,
			"tools":  toolsRoot,
		},
		map[string]string{
			"skills": skillsIndexPath,
			"agents": agentsIndexPath,
			"tools":  toolsIndexPath,
		},
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.InstalledSkills) != 1 || len(result.InstalledAgents) != 1 || len(result.InstalledTools) != 1 {
		t.Fatalf("expected dependencies to be installed, got %#v", result)
	}
	if len(result.HookPaths) != 1 {
		t.Fatalf("expected one hook path, got %#v", result.HookPaths)
	}
	if _, err := os.Stat(filepath.Join(root, "runtime", "skills", "marketing", "demo-skill", "SKILL.md")); err != nil {
		t.Fatalf("expected installed skill, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "runtime", "agents", "marketing", "demo-agent", "AGENT.md")); err != nil {
		t.Fatalf("expected installed agent, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "runtime", "tools-mcp", "analytics", "demo-tool", "TOOL.md")); err != nil {
		t.Fatalf("expected installed tool, got %v", err)
	}
}

func TestInstallPluginDependenciesSkipsExistingWithoutForce(t *testing.T) {
	root := t.TempDir()
	skillsRoot := filepath.Join(root, "skills")
	pluginTarget := filepath.Join(root, "runtime", "plugins")
	mustMkdirAll(t, filepath.Join(skillsRoot, "marketing", "demo-skill"))
	mustWriteFile(t, filepath.Join(skillsRoot, "marketing", "demo-skill", "SKILL.md"), "# Demo skill\n")
	skillsIndexPath := filepath.Join(root, "registry", "skills-index.json")
	mustWriteIndex(t, skillsIndexPath, registry.Index{Skills: []registry.SkillEntry{{ID: "marketing/demo-skill"}}})
	existingTarget := filepath.Join(root, "runtime", "skills", "marketing", "demo-skill")
	mustMkdirAll(t, existingTarget)

	entry := registry.SkillEntry{
		ID: "marketing/demo-plugin",
		Includes: &registry.IncludeSet{
			Skills: []string{"marketing/demo-skill"},
		},
	}
	result, err := InstallPluginDependencies(
		entry,
		"generic",
		pluginTarget,
		map[string]string{"skills": skillsRoot, "agents": filepath.Join(root, "agents"), "tools": filepath.Join(root, "tools-mcp")},
		map[string]string{"skills": skillsIndexPath, "agents": filepath.Join(root, "registry", "agents-index.json"), "tools": filepath.Join(root, "registry", "tools-index.json")},
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.SkippedSkills) != 1 {
		t.Fatalf("expected one skipped skill, got %#v", result)
	}
}

func mustWriteIndex(t *testing.T, path string, idx registry.Index) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir index dir: %v", err)
	}
	if err := registry.WriteIndex(path, idx); err != nil {
		t.Fatalf("write index: %v", err)
	}
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
