package registry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadIndexVersionCompatibility(t *testing.T) {
	for _, version := range []string{"1.1", "1.2"} {
		t.Run(version, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "index.json")
			if err := WriteIndex(path, Index{RegistryVersion: version, Skills: []SkillEntry{}}); err != nil {
				t.Fatalf("write index: %v", err)
			}
			if _, err := LoadIndex(path); err != nil {
				t.Fatalf("load supported registry: %v", err)
			}
		})
	}

	path := filepath.Join(t.TempDir(), "index.json")
	if err := WriteIndex(path, Index{RegistryVersion: "1.3", Skills: []SkillEntry{}}); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if _, err := LoadIndex(path); err == nil || !strings.Contains(err.Error(), "unsupported registry_version") {
		t.Fatalf("expected fail-closed version error, got %v", err)
	}
}

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
