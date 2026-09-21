package registry

import (
	"path/filepath"
	"testing"
)

func TestCatalogClassificationsAreDeclaredAndTruthful(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	tests := []struct {
		name         string
		build        func(string) (Index, error)
		wantCount    int
		availability string
		execution    string
	}{
		{name: "skills", build: BuildSkillsIndex, wantCount: 42, availability: "documentation-only", execution: "instructions"},
		{name: "agents", build: BuildAgentsIndex, wantCount: 7, availability: "template-only", execution: "orchestrator"},
		{name: "plugins", build: BuildPluginsIndex, wantCount: 11, availability: "template-only", execution: "bundle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index, err := test.build(root)
			if err != nil {
				t.Fatalf("build index: %v", err)
			}
			if len(index.Skills) != test.wantCount {
				t.Fatalf("got %d entries, want %d", len(index.Skills), test.wantCount)
			}
			for _, entry := range index.Skills {
				if entry.ID == "marketing/content-repurposing-plugin" {
					if entry.SchemaVersion != "2.1" || entry.Usability.Availability != "not-verified" || entry.Usability.Execution != "bundle" {
						t.Errorf("content repurposing release classification = %#v", entry)
					}
					continue
				}
				assertClassification(t, entry, test.availability, test.execution)
			}
		})
	}

	skills, err := BuildSkillsIndex(root)
	if err != nil {
		t.Fatalf("build skills index for helper audit: %v", err)
	}
	wantHelpers := map[string]string{
		"adtech/analyst-copilot-bigquery-redshift":        "scripts/query_safety_check.py",
		"adtech/playwright-vscode-loop-codex":             "scripts/install-global-codex.sh",
		"adtech/policy-brand-compliance-checker":          "scripts/utm_lint.py",
		"marketing/creative-workshop-pmax-reels":          "scripts/validate_lengths.py",
		"marketing/lifecycle-experiment-planner":          "scripts/sample_size.py",
		"marketing/meta-google-weekly-performance-review": "scripts/compute_metrics.py",
		"marketing/seo-paid-search-synergy":               "scripts/extract_seo_winners.py",
	}
	for _, entry := range skills.Skills {
		want, scriptBearing := wantHelpers[entry.ID]
		if !scriptBearing {
			if len(entry.Usability.ExecutableHelpers) != 0 {
				t.Errorf("%s unexpectedly declares executable helpers", entry.ID)
			}
			continue
		}
		if len(entry.Usability.ExecutableHelpers) != 1 {
			t.Errorf("%s helpers = %#v, want one", entry.ID, entry.Usability.ExecutableHelpers)
			continue
		}
		helper := entry.Usability.ExecutableHelpers[0]
		if helper.Entrypoint != want || helper.Availability != "not-verified" || helper.Execution != "local-tool" {
			t.Errorf("%s helper = %#v, want %s not-verified/local-tool", entry.ID, helper, want)
		}
	}

	tools, err := BuildToolsIndex(root)
	if err != nil {
		t.Fatalf("build tools index: %v", err)
	}
	wantTools := map[string][2]string{
		"ads/meta-ads-mcp-connector":           {"setup-required", "remote-integration"},
		"adtech/ad-platform-executor-template": {"setup-required", "remote-integration"},
		"adtech/conversion-event-reconciler":   {"not-verified", "local-tool"},
		"adtech/openai-ads-adapter-template":   {"template-only", "integration-template"},
		"adtech/openai-ads-api-client":         {"setup-required", "local-tool"},
		"agentops/agent-control-plane-server":  {"setup-required", "local-tool"},
		"analytics/ga4-mcp-connector":          {"setup-required", "remote-integration"},
		"warehouse/bigquery-mcp-query-runner":  {"setup-required", "remote-integration"},
	}
	if len(tools.Skills) != len(wantTools) {
		t.Fatalf("got %d tools, want %d", len(tools.Skills), len(wantTools))
	}
	for _, entry := range tools.Skills {
		want, ok := wantTools[entry.ID]
		if !ok {
			t.Fatalf("unexpected tool %q", entry.ID)
		}
		if entry.ID == "adtech/openai-ads-adapter-template" {
			if !entry.Deprecated || entry.ReplacedBy != "adtech/openai-ads-api-client" || entry.Readiness != "deprecated" || entry.Usability.Source != "declared" || entry.Usability.Availability != "template-only" {
				t.Errorf("deprecated OpenAI Ads adapter projection = %#v", entry)
			}
			continue
		}
		if entry.ID == "adtech/openai-ads-api-client" {
			if entry.SchemaVersion != "2.1" || entry.Deprecated || entry.Usability.Source != "declared" || entry.Usability.Availability != "setup-required" || entry.Usability.Execution != "local-tool" || entry.Execution == nil || entry.Execution.Kind != "cli" || entry.Authentication == nil || entry.Authentication.Status != "optional" {
				t.Errorf("preferred OpenAI Ads client classification = %#v", entry)
			}
			continue
		}
		if entry.ID == "ads/meta-ads-mcp-connector" || entry.ID == "adtech/ad-platform-executor-template" || entry.ID == "analytics/ga4-mcp-connector" || entry.ID == "warehouse/bigquery-mcp-query-runner" || entry.ID == "agentops/agent-control-plane-server" {
			if entry.SchemaVersion != "2.0" || entry.Usability.Source != "declared" || entry.Usability.Availability != "setup-required" || entry.Usability.Execution != want[1] || len(entry.Usability.Limitations) == 0 {
				t.Errorf("executable integration classification = %#v", entry)
			}
			continue
		}
		assertClassification(t, entry, want[0], want[1])
	}
}

func assertClassification(t *testing.T, entry SkillEntry, availability, execution string) {
	t.Helper()
	if entry.SchemaVersion != "1.1" {
		t.Errorf("%s schema_version = %q, want 1.1", entry.ID, entry.SchemaVersion)
	}
	if entry.Usability.Source != "declared" || entry.Usability.Availability != availability || entry.Usability.Execution != execution {
		t.Errorf("%s usability = %#v, want declared %s/%s", entry.ID, entry.Usability, availability, execution)
	}
	if len(entry.Usability.Limitations) == 0 {
		t.Errorf("%s must disclose at least one limitation", entry.ID)
	}
}
