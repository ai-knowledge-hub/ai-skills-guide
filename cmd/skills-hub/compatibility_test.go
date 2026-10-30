package main

import (
	"bytes"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	"strings"
	"testing"
)

func TestCompatibilitySummaryDoesNotPromoteOrSubstituteVersions(t *testing.T) {
	entry := registry.SkillEntry{Latest: "2.0.0", Compatibility: []registry.CompatibilityRequirement{{Kind: "api", Name: "Example API", Versions: []string{"v2"}}}}
	var out bytes.Buffer
	printCompatibilitySummary(&out, entry, "1.0.0")
	if !strings.Contains(out.String(), "unknown") || strings.Contains(out.String(), "v2") {
		t.Fatal("latest declarations leaked into historical selection")
	}
	out.Reset()
	printCompatibilitySummary(&out, entry, "2.0.0")
	if out.String() != "declared_compatibility: api Example API; exact_versions=v2; capabilities=\n" {
		t.Fatalf("declaration misrepresented: %s", out.String())
	}
	out.Reset()
	entry.Compatibility = nil
	printCompatibilitySummary(&out, entry, "2.0.0")
	if !strings.Contains(out.String(), "unknown (not declared)") {
		t.Fatal("legacy absence was not unknown")
	}
}
