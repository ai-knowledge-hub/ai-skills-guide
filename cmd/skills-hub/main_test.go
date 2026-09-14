package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestPrintPluginSummary(t *testing.T) {
	entry := registry.SkillEntry{
		ID: "marketing/performance-reporting-plugin",
		Includes: &registry.IncludeSet{
			Skills: []string{"marketing/meta-google-weekly-performance-review", "adtech/dashboard-generator"},
			Agents: []string{"marketing/weekly-performance-supervisor"},
			Tools:  []string{"analytics/ga4-mcp-connector"},
			Hooks:  []string{"post-analysis-slack-summary"},
		},
		Requires: &registry.RequirementSet{
			Secrets:   []string{"GA4_PROPERTY_ID", "SLACK_WEBHOOK_URL"},
			Approvals: []string{"human-review-for-write-actions"},
		},
	}

	var out bytes.Buffer
	printPluginSummary(&out, entry)
	rendered := out.String()

	expected := []string{
		"includes.skills: 2",
		"includes.skills.list: marketing/meta-google-weekly-performance-review, adtech/dashboard-generator",
		"includes.agents: 1",
		"includes.agents.list: marketing/weekly-performance-supervisor",
		"includes.tools: 1",
		"includes.tools.list: analytics/ga4-mcp-connector",
		"includes.hooks: 1",
		"includes.hooks.list: post-analysis-slack-summary",
		"requires.secrets: GA4_PROPERTY_ID, SLACK_WEBHOOK_URL",
		"requires.approvals: human-review-for-write-actions",
	}
	for _, needle := range expected {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected output to contain %q, got:\n%s", needle, rendered)
		}
	}
}

func TestPrintInstallUsabilityWarning(t *testing.T) {
	tests := []struct {
		availability string
		want         string
	}{
		{availability: "template-only", want: "reference template"},
		{availability: "setup-required", want: "requires configuration"},
		{availability: "not-verified", want: "no current target-scoped operational evidence"},
		{availability: "documentation-only", want: "not executable runtime capability"},
	}
	for _, test := range tests {
		t.Run(test.availability, func(t *testing.T) {
			entry := registry.SkillEntry{ID: "shared/example", Usability: registry.UsabilityMetadata{Availability: test.availability}}
			var out bytes.Buffer
			printInstallUsabilityWarning(&out, entry)
			if !strings.Contains(out.String(), test.want) {
				t.Fatalf("warning %q does not contain %q", out.String(), test.want)
			}
		})
	}
}

func TestPrintUsabilitySummaryIncludesExecutableHelper(t *testing.T) {
	entry := registry.SkillEntry{Usability: registry.UsabilityMetadata{
		Availability: "documentation-only",
		Execution:    "instructions",
		Source:       "declared",
		ExecutableHelpers: []registry.ExecutableHelperMetadata{{
			Entrypoint:   "scripts/check.py",
			Availability: "not-verified",
			Execution:    "local-tool",
			Limitations:  []string{"No current executable evidence."},
			Quickstart:   "python3 scripts/check.py",
		}},
	}}
	var out bytes.Buffer
	printUsabilitySummary(&out, entry)
	for _, want := range []string{
		"usability.executable_helper: scripts/check.py (not-verified, local-tool)",
		"usability.executable_helper.quickstart: python3 scripts/check.py",
		"usability.executable_helper.limitations: No current executable evidence.",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("summary missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunInstallRejectsTemplateBeforeFilesystemWrite(t *testing.T) {
	root := t.TempDir()
	registryPath := filepath.Join(root, "plugins-index.json")
	entry := registry.SkillEntry{
		ID:        "marketing/template-plugin",
		Latest:    "0.1.0",
		Versions:  []registry.VersionEntry{{Version: "0.1.0"}},
		Usability: registry.UsabilityMetadata{Availability: "template-only", Execution: "bundle"},
	}
	if err := registry.WriteIndex(registryPath, registry.Index{RegistryVersion: "1.2", Skills: []registry.SkillEntry{entry}}); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	target := filepath.Join(root, "runtime", "plugins")
	err := runInstall([]string{
		"--module", "plugins",
		"--registry", registryPath,
		"--root", filepath.Join(root, "source", "plugins"),
		"--runtime", "generic",
		"--target", target,
		"--entry", "marketing/template-plugin@0.1.0",
	})
	if err == nil || !strings.Contains(err.Error(), "template-only") {
		t.Fatalf("expected template-only install rejection, got %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("template install created runtime state: %v", statErr)
	}
}

func TestRunInstallPreflightsPluginClosureBeforeFilesystemWrite(t *testing.T) {
	root := t.TempDir()
	mustCreateDir := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	mustCreateFile := func(path, content string) {
		t.Helper()
		mustCreateDir(filepath.Dir(path))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	mustWriteRegistry := func(path string, entries []registry.SkillEntry) {
		t.Helper()
		mustCreateDir(filepath.Dir(path))
		if err := registry.WriteIndex(path, registry.Index{RegistryVersion: "1.2", Skills: entries}); err != nil {
			t.Fatalf("write registry %s: %v", path, err)
		}
	}

	plugin := registry.SkillEntry{
		ID:       "marketing/demo-plugin",
		Latest:   "1.0.0",
		Versions: []registry.VersionEntry{{Version: "1.0.0"}},
		Includes: &registry.IncludeSet{
			Skills: []string{"marketing/implemented"},
			Agents: []string{"marketing/template"},
		},
	}
	mustCreateFile(filepath.Join(root, "plugins", "marketing", "demo-plugin", "README.md"), "# Demo plugin\n")
	mustCreateFile(filepath.Join(root, "skills", "marketing", "implemented", "SKILL.md"), "# Implemented\n")
	mustCreateFile(filepath.Join(root, "agents", "marketing", "template", "AGENT.md"), "# Template\n")
	mustWriteRegistry(filepath.Join(root, "registry", "plugins-index.json"), []registry.SkillEntry{plugin})
	mustWriteRegistry(filepath.Join(root, "registry", "skills-index.json"), []registry.SkillEntry{{ID: "marketing/implemented"}})
	mustWriteRegistry(filepath.Join(root, "registry", "agents-index.json"), []registry.SkillEntry{{
		ID:        "marketing/template",
		Usability: registry.UsabilityMetadata{Availability: "template-only"},
	}})

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	err = runInstall([]string{
		"--module", "plugins",
		"--registry", filepath.Join("registry", "plugins-index.json"),
		"--root", "plugins",
		"--runtime", "generic",
		"--target", filepath.Join("runtime", "plugins"),
		"--entry", "marketing/demo-plugin@1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "template-only") {
		t.Fatalf("expected dependency preflight rejection, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "runtime")); !os.IsNotExist(statErr) {
		t.Fatalf("plugin preflight left partial runtime state: %v", statErr)
	}
}

func TestPrintPluginInstallNotesWarnsWhenNotSecurityReviewed(t *testing.T) {
	entry := registry.SkillEntry{
		ID:               "marketing/performance-reporting-plugin",
		SecurityReviewed: false,
		Includes:         &registry.IncludeSet{},
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	printPluginInstallNotes(
		&out,
		&errOut,
		entry,
		"codex",
		"/tmp/plugins/marketing/performance-reporting-plugin",
		[]string{"/tmp/plugins/marketing/performance-reporting-plugin/.codex-plugin/plugin.json"},
		installer.DependencyInstallResult{
			InstalledSkills: []string{"/tmp/codex/skills/marketing/meta-google-weekly-performance-review"},
			HookPaths:       []string{"/tmp/plugins/marketing/performance-reporting-plugin/hooks/post-analysis-slack-summary.md"},
		},
	)

	if !strings.Contains(errOut.String(), "is not security reviewed") {
		t.Fatalf("expected security review warning, got: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "includes.skills: 0") {
		t.Fatalf("expected plugin summary in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "install.root: /tmp/plugins/marketing/performance-reporting-plugin") {
		t.Fatalf("expected install root in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "install.runtime_artifacts: /tmp/plugins/marketing/performance-reporting-plugin/.codex-plugin/plugin.json") {
		t.Fatalf("expected runtime artifact path in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "install.next_step: review the generated codex runtime manifest before enabling the plugin") {
		t.Fatalf("expected next step guidance in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "deps.skills.installed: 1") {
		t.Fatalf("expected dependency summary in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "deps.hooks.packaged.list: /tmp/plugins/marketing/performance-reporting-plugin/hooks/post-analysis-slack-summary.md") {
		t.Fatalf("expected packaged hook path in stdout, got: %s", out.String())
	}
}
