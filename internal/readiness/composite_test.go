package readiness

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestValidateCompositeReadinessEvidenceAcceptsCompleteAndPartialCoverage(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	installation := compositeInstallationExpectation()
	dependencies := compositeDependencies()
	expectations := compositeExpectations(now)

	complete := compositeEvidenceFixture(now)
	validateCompositeFixture(t, complete, installation, dependencies, expectations, now)

	partial := compositeEvidenceFixture(now)
	partial.ProviderObservations[2] = CompositeProviderObservation{
		ToolID: "warehouse/bigquery-mcp-query-runner", ToolVersion: "0.2.0",
		Requirement: "optional", Access: "read-only", State: "unavailable", FailureReason: "not-configured",
	}
	partial.ProviderObservations = append([]CompositeProviderObservation(nil), partial.ProviderObservations...)
	ready := partial.ProviderObservations[:2]
	partial.CoverageState = "partial"
	partial.PromotionEligible = false
	partial.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(ready)
	validateCompositeFixture(t, partial, installation, dependencies, expectations, now)
}

func TestValidateCompositeReadinessEvidenceRejectsAuthorityAndRelationshipMutations(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	installation := compositeInstallationExpectation()
	dependencies := compositeDependencies()
	expectations := compositeExpectations(now)
	tests := map[string]func(*CompositeReadinessEvidence){
		"coordinated other installation": func(value *CompositeReadinessEvidence) {
			value.EntryID = "marketing/other-plugin"
			value.EntryVersion = "0.3.0"
			value.ArtifactSHA256 = strings.Repeat("7", 64)
			value.DependencyLockSHA256 = strings.Repeat("8", 64)
			value.Runtime = "claude"
			value.Platform = "linux"
			value.EnvironmentID = strings.Repeat("9", 32)
		},
		"unpublished contract version": func(value *CompositeReadinessEvidence) {
			value.ContractVersion = "1.1.0"
		},
		"other entry": func(value *CompositeReadinessEvidence) {
			value.EntryID = "marketing/other-plugin"
		},
		"other entry version": func(value *CompositeReadinessEvidence) {
			value.EntryVersion = "0.3.0"
		},
		"other artifact": func(value *CompositeReadinessEvidence) {
			value.ArtifactSHA256 = strings.Repeat("7", 64)
		},
		"other dependency lock": func(value *CompositeReadinessEvidence) {
			value.DependencyLockSHA256 = strings.Repeat("8", 64)
		},
		"other runtime": func(value *CompositeReadinessEvidence) {
			value.Runtime = "claude"
		},
		"other platform": func(value *CompositeReadinessEvidence) {
			value.Platform = "linux"
		},
		"other environment": func(value *CompositeReadinessEvidence) {
			value.EnvironmentID = strings.Repeat("9", 32)
		},
		"substituted provider target": func(value *CompositeReadinessEvidence) {
			value.ProviderObservations[0].TargetFingerprint = strings.Repeat("d", 64)
			value.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(value.ProviderObservations)
		},
		"changed required authority": func(value *CompositeReadinessEvidence) {
			value.ProviderObservations[0].Requirement = "optional"
			value.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(value.ProviderObservations)
		},
		"omitted required provider": func(value *CompositeReadinessEvidence) {
			value.ProviderObservations = value.ProviderObservations[1:]
			value.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(value.ProviderObservations)
		},
		"duplicated provider": func(value *CompositeReadinessEvidence) {
			value.ProviderObservations[1] = value.ProviderObservations[0]
			value.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(value.ProviderObservations)
		},
		"forged aggregate fingerprint": func(value *CompositeReadinessEvidence) {
			value.WorkflowTargetFingerprint = strings.Repeat("e", 64)
		},
		"expired child evidence": func(value *CompositeReadinessEvidence) {
			expired := now.Add(-time.Minute)
			value.ProviderObservations[0].ExpiresAt = &expired
			value.ExpiresAt = now.Add(-time.Minute)
			value.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(value.ProviderObservations)
		},
		"extended child evidence lifetime": func(value *CompositeReadinessEvidence) {
			extended := now.Add(365 * 24 * time.Hour)
			for index := range value.ProviderObservations {
				value.ProviderObservations[index].ExpiresAt = timePointer(extended)
			}
			value.ExpiresAt = extended
		},
		"changed child observation time": func(value *CompositeReadinessEvidence) {
			changed := now.Add(-2 * time.Minute)
			value.ProviderObservations[0].ObservedAt = timePointer(changed)
		},
		"partial promoted": func(value *CompositeReadinessEvidence) {
			value.ProviderObservations[2] = CompositeProviderObservation{
				ToolID: "warehouse/bigquery-mcp-query-runner", ToolVersion: "0.2.0",
				Requirement: "optional", Access: "read-only", State: "unavailable", FailureReason: "not-configured",
			}
			value.CoverageState = "partial"
			value.WorkflowTargetFingerprint = CompositeWorkflowTargetFingerprint(value.ProviderObservations)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := compositeEvidenceFixture(now)
			mutate(&candidate)
			payload, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateCompositeReadinessEvidence(payload, installation, dependencies, expectations, now); err == nil {
				t.Fatal("harmful mutation was accepted")
			}
		})
	}
}

func TestCompositeWorkflowTargetFingerprintIsOrderIndependentAndEvidenceBound(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	observations := compositeEvidenceFixture(now).ProviderObservations
	want := CompositeWorkflowTargetFingerprint(observations)
	reversed := []CompositeProviderObservation{observations[2], observations[1], observations[0]}
	if got := CompositeWorkflowTargetFingerprint(reversed); got != want {
		t.Fatalf("fingerprint changed with ordering: got %s want %s", got, want)
	}
	reversed[0].EvidenceSHA256 = strings.Repeat("f", 64)
	if got := CompositeWorkflowTargetFingerprint(reversed); got == want {
		t.Fatal("fingerprint did not change with child evidence")
	}
}

func compositeDependencies() []registry.ProviderDependencyMetadata {
	return []registry.ProviderDependencyMetadata{
		{Tool: "analytics/ga4-mcp-connector", Requirement: "required", Access: "read-only"},
		{Tool: "ads/meta-ads-mcp-connector", Requirement: "required", Access: "read-only"},
		{Tool: "warehouse/bigquery-mcp-query-runner", Requirement: "optional", Access: "read-only"},
	}
}

func compositeExpectations(now time.Time) map[string]ProviderEvidenceExpectation {
	observed := now.Add(-time.Minute)
	expires := now.Add(time.Hour)
	return map[string]ProviderEvidenceExpectation{
		"analytics/ga4-mcp-connector": {
			ToolVersion: "0.2.0", TargetFingerprint: strings.Repeat("1", 64), EvidenceSHA256: strings.Repeat("a", 64), ObservedAt: observed, ExpiresAt: expires,
		},
		"ads/meta-ads-mcp-connector": {
			ToolVersion: "0.2.0", TargetFingerprint: strings.Repeat("2", 64), EvidenceSHA256: strings.Repeat("b", 64), ObservedAt: observed, ExpiresAt: expires,
		},
		"warehouse/bigquery-mcp-query-runner": {
			ToolVersion: "0.2.0", TargetFingerprint: strings.Repeat("3", 64), EvidenceSHA256: strings.Repeat("c", 64), ObservedAt: observed, ExpiresAt: expires,
		},
	}
}

func compositeInstallationExpectation() CompositeInstallationExpectation {
	return CompositeInstallationExpectation{
		EntryID: "marketing/performance-reporting-plugin", EntryVersion: "0.2.0",
		ArtifactSHA256: strings.Repeat("4", 64), DependencyLockSHA256: strings.Repeat("5", 64),
		Runtime: "codex", Platform: "macos", EnvironmentID: strings.Repeat("6", 32),
	}
}

func compositeEvidenceFixture(now time.Time) CompositeReadinessEvidence {
	observed := now.Add(-time.Minute)
	expires := now.Add(time.Hour)
	expectations := compositeExpectations(now)
	observations := make([]CompositeProviderObservation, 0, 3)
	for _, dependency := range compositeDependencies() {
		expectation := expectations[dependency.Tool]
		observations = append(observations, CompositeProviderObservation{
			ToolID: dependency.Tool, ToolVersion: expectation.ToolVersion,
			Requirement: dependency.Requirement, Access: dependency.Access, State: "ready",
			TargetFingerprint: expectation.TargetFingerprint, EvidenceSHA256: expectation.EvidenceSHA256,
			ObservedAt: timePointer(observed), ExpiresAt: timePointer(expires),
		})
	}
	return CompositeReadinessEvidence{
		SchemaVersion: compositeEvidenceSchema, ContractID: "operational-readiness", ContractVersion: "1.0.3",
		EntryID: "marketing/performance-reporting-plugin", EntryVersion: "0.2.0",
		ArtifactSHA256: strings.Repeat("4", 64), DependencyLockSHA256: strings.Repeat("5", 64),
		Runtime: "codex", Platform: "macos", EnvironmentID: strings.Repeat("6", 32),
		ConfigurationClass: "authenticated-composite", CoverageState: "complete",
		WorkflowTargetFingerprint: CompositeWorkflowTargetFingerprint(observations),
		ProviderObservations:      observations, ObservedAt: observed, ExpiresAt: expires,
		Result: "passed", PromotionEligible: true,
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func validateCompositeFixture(t *testing.T, evidence CompositeReadinessEvidence, installation CompositeInstallationExpectation, dependencies []registry.ProviderDependencyMetadata, expectations map[string]ProviderEvidenceExpectation, now time.Time) {
	t.Helper()
	payload, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateCompositeReadinessEvidence(payload, installation, dependencies, expectations, now); err != nil {
		t.Fatalf("validate composite evidence: %v", err)
	}
}
