package main

import (
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type catalogueReadinessFlags struct{ runtime, target, stateDir *string }
type catalogueStatus struct {
	Availability  string
	EvidenceScope string
	ExpiresAt     time.Time
	Reason        string
}

// Display a static catalogue declaration without granting runtime authority.
// Work on copies: the selected manifest remains authoritative for admission.
func catalogueDisplayEntry(entry registry.SkillEntry) registry.SkillEntry {
	if entry.Usability.Availability == "usable-now" {
		entry.Usability.Availability = "not-verified"
		entry.Usability.Source = "inferred"
	}
	entry.CapabilityReadiness = append([]registry.CapabilityReadinessMetadata(nil), entry.CapabilityReadiness...)
	for i := range entry.CapabilityReadiness {
		capability := &entry.CapabilityReadiness[i]
		if capability.Availability == "usable-now" || capability.Availability == "setup-required" {
			capability.Availability = "not-verified"
			entry.Usability.Source = "inferred"
		}
	}
	entry.Usability.ExecutableHelpers = append([]registry.ExecutableHelperMetadata(nil), entry.Usability.ExecutableHelpers...)
	for i := range entry.Usability.ExecutableHelpers {
		helper := &entry.Usability.ExecutableHelpers[i]
		if helper.Availability == "usable-now" || helper.Availability == "setup-required" {
			helper.Availability = "not-verified"
			entry.Usability.Source = "inferred"
		}
	}
	return registry.ApplyLifecycleProjection(entry)
}
