package agents

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CapabilityExpectation is the authoritative input contract for one
// composition. Returned results cannot add, remove, or reclassify requirements.
type CapabilityExpectation struct {
	Capability string `json:"capability"`
	Source     string `json:"source"`
	Required   bool   `json:"required"`
}

type CapabilityResult struct {
	Capability string    `json:"capability"`
	Source     string    `json:"source"`
	Status     string    `json:"status"` // success | unavailable | failed | cancelled
	Required   bool      `json:"required"`
	ObservedAt time.Time `json:"observed_at"`
	FreshUntil time.Time `json:"fresh_until"`
}

type CompositionResult struct {
	Status          string             `json:"status"` // success | partial | unavailable | cancelled | ambiguous
	SourceCoverage  []string           `json:"source_coverage"`
	ObservedAt      time.Time          `json:"observed_at"`
	FreshUntil      time.Time          `json:"fresh_until"`
	BlockingReasons []string           `json:"blocking_reasons"`
	Results         []CapabilityResult `json:"results"`
}

// ComposeCapabilityResults compares returned evidence with an independent
// expected contract. Terminal precedence is fixed and order-independent:
// cancellation, ambiguity, required unavailability, optional partial failure,
// then success.
func ComposeCapabilityResults(expected []CapabilityExpectation, results []CapabilityResult) CompositionResult {
	out := CompositionResult{Status: "success", SourceCoverage: []string{}, BlockingReasons: []string{}, Results: append([]CapabilityResult(nil), results...)}
	expectedByCapability := make(map[string]CapabilityExpectation, len(expected))
	ambiguous := false
	for _, item := range expected {
		if strings.TrimSpace(item.Capability) == "" || strings.TrimSpace(item.Source) == "" {
			ambiguous = true
			out.BlockingReasons = append(out.BlockingReasons, "capability contract contains an incomplete expectation")
			continue
		}
		if _, duplicate := expectedByCapability[item.Capability]; duplicate {
			ambiguous = true
			out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("capability contract repeats %s", item.Capability))
			continue
		}
		expectedByCapability[item.Capability] = item
	}

	seenResults := map[string]bool{}
	seenSources := map[string]bool{}
	cancelled, requiredUnavailable, optionalPartial := false, false, false
	for _, result := range results {
		want, declared := expectedByCapability[result.Capability]
		if !declared {
			ambiguous = true
			out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("unexpected capability result %s", result.Capability))
		} else if seenResults[result.Capability] {
			ambiguous = true
			out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("capability %s returned more than once", result.Capability))
		} else if result.Source != want.Source || result.Required != want.Required {
			ambiguous = true
			out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("capability %s result does not match its expected source and requirement", result.Capability))
		}
		seenResults[result.Capability] = true
		if strings.TrimSpace(result.Source) != "" && !seenSources[result.Source] {
			seenSources[result.Source] = true
			out.SourceCoverage = append(out.SourceCoverage, result.Source)
		}

		validStatus := result.Status == "success" || result.Status == "unavailable" || result.Status == "failed" || result.Status == "cancelled"
		validTime := !result.ObservedAt.IsZero() && !result.FreshUntil.IsZero() && result.FreshUntil.After(result.ObservedAt)
		if !validStatus || !validTime {
			if declared && want.Required {
				requiredUnavailable = true
			} else {
				optionalPartial = true
			}
			out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("capability %s returned invalid status or freshness evidence", result.Capability))
			continue
		}
		if out.ObservedAt.IsZero() || result.ObservedAt.Before(out.ObservedAt) {
			out.ObservedAt = result.ObservedAt
		}
		if out.FreshUntil.IsZero() || result.FreshUntil.Before(out.FreshUntil) {
			out.FreshUntil = result.FreshUntil
		}
		if result.Status == "cancelled" {
			cancelled = true
			out.BlockingReasons = append(out.BlockingReasons, "orchestration was cancelled")
		} else if result.Status != "success" {
			if declared && want.Required {
				requiredUnavailable = true
				out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("required capability %s is %s", result.Capability, result.Status))
			} else {
				optionalPartial = true
			}
		}
	}
	for capability, want := range expectedByCapability {
		if !seenResults[capability] && want.Required {
			requiredUnavailable = true
			out.BlockingReasons = append(out.BlockingReasons, fmt.Sprintf("required capability %s returned no result", capability))
		}
	}

	switch {
	case cancelled:
		out.Status = "cancelled"
	case ambiguous:
		out.Status = "ambiguous"
	case requiredUnavailable:
		out.Status = "unavailable"
	case optionalPartial:
		out.Status = "partial"
	default:
		out.Status = "success"
	}
	sort.Strings(out.SourceCoverage)
	sort.Strings(out.BlockingReasons)
	return out
}
