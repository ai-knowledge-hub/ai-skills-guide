package registry

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

// Named regression anchors come from the package roles documented in
// docs/catalogue-test-assurance.md. They are a required subset, not a census.
var skillHelpers = map[string]string{
	"adtech/analyst-copilot-bigquery-redshift":        "scripts/query_safety_check.py",
	"adtech/playwright-vscode-loop-codex":             "scripts/install-global-codex.sh",
	"adtech/policy-brand-compliance-checker":          "scripts/utm_lint.py",
	"marketing/creative-workshop-pmax-reels":          "scripts/validate_lengths.py",
	"marketing/lifecycle-experiment-planner":          "scripts/sample_size.py",
	"marketing/meta-google-weekly-performance-review": "scripts/compute_metrics.py",
	"marketing/seo-paid-search-synergy":               "scripts/extract_seo_winners.py",
}

func TestCatalogClassificationsAreDeclaredAndTruthful(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	tests := []struct {
		name         string
		build        func(string) (Index, error)
		required     []string
		availability string
		execution    string
	}{
		{name: "skills", build: BuildSkillsIndex, required: helperSkillIDs(), availability: "documentation-only", execution: "instructions"},
		{name: "agents", build: BuildAgentsIndex, required: []string{
			"adtech/ad-platform-control-plane-supervisor", "adtech/bi-insights-orchestrator",
			"adtech/chatgpt-ads-experiment-supervisor", "agentops/agent-control-plane-supervisor",
			"marketing/campaign-qa-supervisor", "marketing/creative-operating-system-supervisor",
			"marketing/weekly-performance-supervisor",
		}, availability: "setup-required", execution: "orchestrator"},
		{name: "plugins", build: BuildPluginsIndex, required: []string{
			"agentops/harness-governance-plugin", "engineering/code-maintenance-plugin",
			"marketing/ad-creative-plugin", "marketing/campaign-audit-plugin",
			"marketing/chatgpt-advertising-experiment-plugin", "marketing/competitive-intelligence-plugin",
			"marketing/content-repurposing-plugin", "marketing/creative-operating-system-plugin",
			"marketing/page-speed-technical-seo-plugin", "marketing/performance-reporting-plugin",
			"security/runtime-safety-plugin",
		}, availability: "template-only", execution: "bundle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index, err := test.build(root)
			if err != nil {
				t.Fatalf("build index: %v", err)
			}
			if err := requireCatalogueIdentities(index, test.required); err != nil {
				t.Fatal(err)
			}
			for _, entry := range index.Skills {
				// Future packages can have different valid classifications. Every
				// entry must still disclose its declared usability and limitations.
				if entry.Usability.Source != "declared" || len(entry.Usability.Limitations) == 0 {
					t.Errorf("%s must disclose declared usability and limitations", entry.ID)
				}
				if !slices.Contains(test.required, entry.ID) {
					continue
				}
				if test.name == "plugins" && localPluginWave[entry.ID] {
					if entry.Usability.Availability != "usable-now" || entry.Usability.Execution != "bundle" || entry.Artifact == nil || !entry.Artifact.SelfContained {
						t.Errorf("local plugin wave classification = %#v", entry)
					}
					continue
				}
				if test.name == "agents" && entry.ID == "marketing/campaign-qa-supervisor" {
					if entry.Usability.Availability != "not-verified" || entry.Usability.Execution != "orchestrator" || entry.Orchestration == nil {
						t.Errorf("unavailable-provider agent classification = %#v", entry)
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
	for _, entry := range skills.Skills {
		want, scriptBearing := skillHelpers[entry.ID]
		if !scriptBearing {
			continue
		}
		found := false
		for _, helper := range entry.Usability.ExecutableHelpers {
			if helper.Entrypoint == want {
				found = true
				if helper.Availability != "not-verified" || helper.Execution != "local-tool" {
					t.Errorf("%s helper = %#v, want %s not-verified/local-tool", entry.ID, helper, want)
				}
			}
		}
		if !found {
			t.Errorf("%s is missing required helper %s", entry.ID, want)
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
	var requiredTools []string
	for id := range wantTools {
		requiredTools = append(requiredTools, id)
	}
	if err := requireCatalogueIdentities(tools, requiredTools); err != nil {
		t.Fatal(err)
	}
	for _, entry := range tools.Skills {
		if entry.Usability.Source != "declared" || len(entry.Usability.Limitations) == 0 {
			t.Errorf("%s must disclose declared usability and limitations", entry.ID)
		}
		want, ok := wantTools[entry.ID]
		if !ok {
			continue // A valid additional tool does not change existing promises.
		}
		if entry.ID == "adtech/openai-ads-adapter-template" {
			if !entry.Deprecated || entry.ReplacedBy != "adtech/openai-ads-api-client" || entry.Readiness != "deprecated" || entry.Usability.Source != "declared" || entry.Usability.Availability != "template-only" {
				t.Errorf("deprecated OpenAI Ads adapter projection = %#v", entry)
			}
			continue
		}
		if entry.ID == "adtech/openai-ads-api-client" {
			if entry.Deprecated || entry.Usability.Source != "declared" || entry.Usability.Availability != "setup-required" || entry.Usability.Execution != "local-tool" || entry.Execution == nil || entry.Execution.Kind != "cli" || entry.Authentication == nil || entry.Authentication.Status != "optional" {
				t.Errorf("preferred OpenAI Ads client classification = %#v", entry)
			}
			continue
		}
		if entry.ID == "ads/meta-ads-mcp-connector" || entry.ID == "adtech/ad-platform-executor-template" || entry.ID == "analytics/ga4-mcp-connector" || entry.ID == "warehouse/bigquery-mcp-query-runner" || entry.ID == "agentops/agent-control-plane-server" {
			if entry.Usability.Source != "declared" || entry.Usability.Availability != "setup-required" || entry.Usability.Execution != want[1] || len(entry.Usability.Limitations) == 0 {
				t.Errorf("executable integration classification = %#v", entry)
			}
			continue
		}
		assertClassification(t, entry, want[0], want[1])
	}
}

func helperSkillIDs() []string {
	var ids []string
	for id := range skillHelpers {
		ids = append(ids, id)
	}
	return ids
}

func requireCatalogueIdentities(index Index, required []string) error {
	seen := make(map[string]bool)
	for _, entry := range index.Skills {
		if seen[entry.ID] {
			return fmt.Errorf("duplicate catalogue identity %q", entry.ID)
		}
		seen[entry.ID] = true
	}
	for _, id := range required {
		if !seen[id] {
			return fmt.Errorf("missing required catalogue identity %q", id)
		}
	}
	return nil
}

func TestCatalogueMembershipAllowsGrowthButRejectsIdentityLoss(t *testing.T) {
	// Independent fixture requirement: the reporting workflow needs its named
	// skill. A valid same-shape replacement cannot satisfy that promise.
	required := []string{"marketing/required-report"}
	for _, test := range []struct {
		name  string
		ids   []string
		valid bool
	}{
		{"baseline", []string{"marketing/required-report"}, true},
		{"addition", []string{"marketing/required-report", "engineering/new-skill"}, true},
		{"reordering", []string{"engineering/new-skill", "marketing/required-report"}, true},
		{"deletion", nil, false},
		{"same-count substitution", []string{"marketing/other-valid-report"}, false},
		{"duplicate", []string{"marketing/required-report", "marketing/required-report"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var index Index
			for _, id := range test.ids {
				index.Skills = append(index.Skills, SkillEntry{ID: id})
			}
			if err := requireCatalogueIdentities(index, required); (err == nil) != test.valid {
				t.Fatalf("valid=%v: %v", test.valid, err)
			}
		})
	}
}

var localPluginWave = map[string]bool{
	"agentops/harness-governance-plugin":              true,
	"engineering/code-maintenance-plugin":             true,
	"marketing/chatgpt-advertising-experiment-plugin": true,
	"marketing/competitive-intelligence-plugin":       true,
	"marketing/content-repurposing-plugin":            true,
	"marketing/creative-operating-system-plugin":      true,
	"security/runtime-safety-plugin":                  true,
}

func assertClassification(t *testing.T, entry SkillEntry, availability, execution string) {
	t.Helper()
	if entry.Usability.Source != "declared" || entry.Usability.Availability != availability || entry.Usability.Execution != execution {
		t.Errorf("%s usability = %#v, want declared %s/%s", entry.ID, entry.Usability, availability, execution)
	}
	if len(entry.Usability.Limitations) == 0 {
		t.Errorf("%s must disclose at least one limitation", entry.ID)
	}
}
