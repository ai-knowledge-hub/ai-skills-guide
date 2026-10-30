package agents

import (
	"reflect"
	"testing"
	"time"
)

func TestComposeCapabilityResultsUsesIndependentContract(t *testing.T) {
	t0 := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	expected := []CapabilityExpectation{
		{Capability: "ga4_query", Source: "analytics/ga4-mcp-connector@0.2.0", Required: true},
		{Capability: "warehouse_query", Source: "warehouse/bigquery-mcp-query-runner@0.2.0", Required: true},
		{Capability: "optional_alert", Source: "human/delivery", Required: false},
	}
	base := []CapabilityResult{
		{Capability: "ga4_query", Source: expected[0].Source, Status: "success", Required: true, ObservedAt: t0, FreshUntil: t0.Add(time.Hour)},
		{Capability: "warehouse_query", Source: expected[1].Source, Status: "success", Required: true, ObservedAt: t0.Add(time.Minute), FreshUntil: t0.Add(30 * time.Minute)},
	}
	tests := []struct {
		name, want string
		mutate     func([]CapabilityResult) []CapabilityResult
	}{
		{name: "success", want: "success", mutate: func(in []CapabilityResult) []CapabilityResult { return in }},
		{name: "missing required", want: "unavailable", mutate: func(in []CapabilityResult) []CapabilityResult { return in[:1] }},
		{name: "only optional cannot self-certify completeness", want: "unavailable", mutate: func(_ []CapabilityResult) []CapabilityResult {
			return []CapabilityResult{{Capability: "optional_alert", Source: expected[2].Source, Status: "success", ObservedAt: t0, FreshUntil: t0.Add(time.Hour)}}
		}},
		{name: "unexpected capability", want: "ambiguous", mutate: func(in []CapabilityResult) []CapabilityResult {
			return append(in, CapabilityResult{Capability: "other", Source: "other/provider", Status: "success", ObservedAt: t0, FreshUntil: t0.Add(time.Hour)})
		}},
		{name: "duplicate capability", want: "ambiguous", mutate: func(in []CapabilityResult) []CapabilityResult { return append(in, in[0]) }},
		{name: "substituted source", want: "ambiguous", mutate: func(in []CapabilityResult) []CapabilityResult { in[0].Source = "other/provider"; return in }},
		{name: "zero timestamps", want: "unavailable", mutate: func(in []CapabilityResult) []CapabilityResult { in[0].ObservedAt = time.Time{}; return in }},
		{name: "unknown status", want: "unavailable", mutate: func(in []CapabilityResult) []CapabilityResult { in[0].Status = "maybe"; return in }},
		{name: "unavailable dependency", want: "unavailable", mutate: func(in []CapabilityResult) []CapabilityResult { in[0].Status = "unavailable"; return in }},
		{name: "cancellation", want: "cancelled", mutate: func(in []CapabilityResult) []CapabilityResult { in[1].Status = "cancelled"; return in }},
		{name: "optional failure", want: "partial", mutate: func(in []CapabilityResult) []CapabilityResult {
			return append(in, CapabilityResult{Capability: "optional_alert", Source: expected[2].Source, Status: "failed", ObservedAt: t0, FreshUntil: t0.Add(time.Hour)})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := append([]CapabilityResult(nil), base...)
			result := ComposeCapabilityResults(expected, test.mutate(input))
			if result.Status != test.want {
				t.Fatalf("status=%s want=%s (%#v)", result.Status, test.want, result.BlockingReasons)
			}
			if test.want == "success" && (!result.ObservedAt.Equal(t0) || !result.FreshUntil.Equal(t0.Add(30*time.Minute)) || len(result.SourceCoverage) != 2) {
				t.Fatalf("source coverage/freshness not retained: %#v", result)
			}
		})
	}
}

func TestComposeCapabilityResultsTerminalPrecedenceIsOrderIndependent(t *testing.T) {
	t0 := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	expected := []CapabilityExpectation{{Capability: "first", Source: "one", Required: true}, {Capability: "second", Source: "two", Required: true}}
	results := []CapabilityResult{
		{Capability: "first", Source: "one", Required: true, Status: "cancelled", ObservedAt: t0, FreshUntil: t0.Add(time.Hour)},
		{Capability: "second", Source: "wrong", Required: true, Status: "success", ObservedAt: t0, FreshUntil: t0.Add(time.Hour)},
	}
	forward := ComposeCapabilityResults(expected, results)
	reversed := ComposeCapabilityResults(expected, []CapabilityResult{results[1], results[0]})
	if forward.Status != "cancelled" || !reflect.DeepEqual(forward.BlockingReasons, reversed.BlockingReasons) || forward.Status != reversed.Status {
		t.Fatalf("composition depends on input order: forward=%#v reversed=%#v", forward, reversed)
	}
}
