package main

import (
	"bytes"
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
