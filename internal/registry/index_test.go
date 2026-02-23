package registry

import "testing"

func TestResolveVersion(t *testing.T) {
	skill := SkillEntry{
		ID:     "marketing/demo",
		Latest: "0.2.0",
		Versions: []VersionEntry{
			{Version: "0.1.0"},
			{Version: "0.2.0"},
		},
	}
	v, err := ResolveVersion(skill, "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Version != "0.2.0" {
		t.Fatalf("expected latest version, got %s", v.Version)
	}
}

func TestSearchFilters(t *testing.T) {
	idx := Index{Skills: []SkillEntry{
		{
			ID:          "marketing/meta-weekly",
			Name:        "Meta Weekly",
			Description: "weekly report",
			Category:    "marketing-tools/ads-ops",
			Tags:        []string{"paid-media", "weekly"},
			Runtimes:    []string{"codex", "generic"},
		},
		{
			ID:          "adtech/sql-safety",
			Name:        "SQL Safety",
			Description: "query checks",
			Category:    "adtech/analytics-engineering",
			Tags:        []string{"sql"},
			Runtimes:    []string{"claude"},
		},
	}}

	results := Search(idx, SearchQuery{Tag: "paid-media", Runtime: "codex"})
	if len(results) != 1 || results[0].ID != "marketing/meta-weekly" {
		t.Fatalf("unexpected search results: %#v", results)
	}
}
