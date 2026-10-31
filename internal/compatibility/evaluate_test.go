package compatibility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func fixture() Context {
	return Context{Now: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC), Subject: Subject{
		EntryID: "analytics/example", EntryVersion: "1.0.0", ArtifactSHA256: strings.Repeat("a", 64), DependencyLockSHA256: strings.Repeat("b", 64), ConfigurationSHA256: strings.Repeat("c", 64), Runtime: "generic", Platform: "linux", Operation: "read-report", ProviderTarget: "sandbox-property-1",
		Dimensions: []Dimension{{Kind: "api", Name: "Example API", Version: "v1beta"}, {Kind: "sdk", Name: "Example SDK", Version: "3.2.1"}},
	}, Requirements: []registry.CompatibilityRequirement{{Kind: "api", Name: "Example API", Versions: []string{"v1beta"}}, {Kind: "sdk", Name: "Example SDK", Versions: []string{"3.2.1"}}}, TrustedEvidence: map[string]string{}}
}
func record(t *testing.T, ctx *Context, result string, at time.Time, alter func(*Evidence)) []byte {
	t.Helper()
	binding, err := RequirementsDigest(ctx.Requirements)
	if err != nil {
		t.Fatal(err)
	}
	e := Evidence{SchemaVersion: EvidenceVersion, ContractVersion: "2.0.0", Subject: ctx.Subject, RequirementsSHA256: binding, Producer: "test-ci", TestID: "bounded-query", TestVersion: "1", ObservedAt: at, Result: result, Reason: "synthetic compatibility test result"}
	// Independent copies prevent the test builder from changing the target.
	data, _ := json.Marshal(e)
	var copied Evidence
	if err := json.Unmarshal(data, &copied); err != nil {
		t.Fatal(err)
	}
	e = copied
	if alter != nil {
		alter(&e)
	}
	data, _ = json.Marshal(e)
	sum := sha256.Sum256(data)
	ctx.TrustedEvidence[hex.EncodeToString(sum[:])] = "test-ci"
	return data
}
func assess(t *testing.T, ctx Context, want string, records ...[]byte) Assessment {
	t.Helper()
	report, err := Evaluate(ctx, records)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != want {
		t.Fatalf("want %s, got %+v", want, report)
	}
	return report
}

func TestCompatibilityUsesIdentityAndChangesRatherThanAge(t *testing.T) {
	ctx := fixture()
	at := ctx.Now.AddDate(-1, 0, 0)
	payload := record(t, &ctx, "passed", at, nil)
	assess(t, ctx, "tested-compatible", payload)
	ctx.Now = ctx.Now.AddDate(100, 0, 0)
	report := assess(t, ctx, "tested-compatible", payload)
	if report.ObservedAt == nil || !report.ObservedAt.Equal(at) {
		t.Fatal("historical observation time was renewed")
	}
	for name, change := range map[string]func(*Subject){
		"artifact":        func(s *Subject) { s.ArtifactSHA256 = strings.Repeat("d", 64) },
		"lock":            func(s *Subject) { s.DependencyLockSHA256 = strings.Repeat("d", 64) },
		"configuration":   func(s *Subject) { s.ConfigurationSHA256 = strings.Repeat("d", 64) },
		"entry":           func(s *Subject) { s.EntryID = "analytics/other" },
		"release":         func(s *Subject) { s.EntryVersion = "2.0.0" },
		"runtime":         func(s *Subject) { s.Runtime = "codex" },
		"platform":        func(s *Subject) { s.Platform = "macos" },
		"operation":       func(s *Subject) { s.Operation = "write-report" },
		"provider-target": func(s *Subject) { s.ProviderTarget = "live-property-2" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := ctx
			change(&candidate.Subject)
			r := assess(t, candidate, "declared-supported", payload)
			if r.EvidenceSHA256 != "" {
				t.Fatal("other scope retained evidence")
			}
		})
	}
}

func TestCompatibilityRequiresIndependentTrustAndCompleteRequirements(t *testing.T) {
	ctx := fixture()
	payload := record(t, &ctx, "passed", ctx.Now.Add(-time.Hour), nil)
	untrusted := ctx
	untrusted.TrustedEvidence = nil
	assess(t, untrusted, "declared-supported", payload)
	wrongProducer := ctx
	wrongProducer.TrustedEvidence = map[string]string{}
	for sha := range ctx.TrustedEvidence {
		wrongProducer.TrustedEvidence[sha] = "other-ci"
	}
	assess(t, wrongProducer, "declared-supported", payload)
	// Changing even insignificant raw bytes requires independent approval again.
	assess(t, ctx, "declared-supported", append(append([]byte{}, payload...), byte(' ')))
	fewer := ctx
	fewer.Requirements = ctx.Requirements[:1]
	assess(t, fewer, "declared-supported", payload)
	absent := ctx
	absent.Requirements = nil
	assess(t, absent, "unknown", payload)
	missing := ctx
	missing.Subject.Dimensions = ctx.Subject.Dimensions[:1]
	assess(t, missing, "unknown", payload)
	unknownVersion := fixture()
	unknownVersion.Subject.Dimensions[0].Version = "v2"
	assess(t, unknownVersion, "unknown", payload)
}

func TestCompatibilityDoesNotUnionSeparateVersionCombinations(t *testing.T) {
	ctx := fixture()
	apiOnly := record(t, &ctx, "passed", ctx.Now.Add(-time.Hour), func(e *Evidence) { e.Subject.Dimensions[1].Version = "3.0.0" })
	sdkOnly := record(t, &ctx, "passed", ctx.Now.Add(-time.Hour), func(e *Evidence) { e.Subject.Dimensions[0].Version = "v2" })
	assess(t, ctx, "declared-supported", apiOnly, sdkOnly)
}

func TestCompatibilityChronologyAndConservativeTies(t *testing.T) {
	ctx := fixture()
	at := ctx.Now.Add(-time.Hour)
	pass := record(t, &ctx, "passed", at, nil)
	fail := record(t, &ctx, "incompatible", at.Add(time.Second), nil)
	inconclusive := record(t, &ctx, "inconclusive", at.Add(2*time.Second), nil)
	assess(t, ctx, "known-incompatible", fail, pass)
	assess(t, ctx, "unknown", pass, fail, inconclusive)
	future := record(t, &ctx, "passed", ctx.Now.Add(time.Second), nil)
	assess(t, ctx, "known-incompatible", fail, future)
	tied := record(t, &ctx, "passed", at.Add(time.Second), nil)
	assess(t, ctx, "known-incompatible", fail, tied)
	assess(t, ctx, "known-incompatible", tied, fail)
	recovery := record(t, &ctx, "passed", at.Add(3*time.Second), nil)
	assess(t, ctx, "tested-compatible", fail, recovery)
}

func TestRollingCapabilitiesAndCanonicalOrdering(t *testing.T) {
	ctx := fixture()
	ctx.Requirements[0].Versions = nil
	ctx.Requirements[0].Capabilities = []string{"bounded-query", "pagination"}
	ctx.Subject.Dimensions[0].Version = ""
	ctx.Subject.Dimensions[0].Capabilities = []string{"bounded-query", "pagination"}
	payload := record(t, &ctx, "passed", ctx.Now, nil)
	before, _ := json.Marshal(ctx)
	reordered := ctx
	reordered.Subject.Dimensions = append([]Dimension{}, ctx.Subject.Dimensions...)
	reordered.Subject.Dimensions[0].Capabilities = []string{"pagination", "bounded-query"}
	reordered.Subject.Dimensions[0], reordered.Subject.Dimensions[1] = reordered.Subject.Dimensions[1], reordered.Subject.Dimensions[0]
	reordered.Requirements = append([]registry.CompatibilityRequirement{}, ctx.Requirements...)
	reordered.Requirements[0].Notes = "explanatory change"
	assess(t, reordered, "tested-compatible", payload)
	after, _ := json.Marshal(ctx)
	if string(before) != string(after) {
		t.Fatal("evaluation mutated caller state")
	}
	ctx.Subject.Dimensions[0].Capabilities = []string{"bounded-query"}
	assess(t, ctx, "unknown", payload)
	failure := record(t, &ctx, "incompatible", ctx.Now, nil)
	assess(t, ctx, "known-incompatible", failure)
}

func TestEvidenceSchemaAndContextRejectInvalidClaims(t *testing.T) {
	ctx := fixture()
	payload := record(t, &ctx, "passed", ctx.Now, nil)
	for name, edit := range map[string]func(map[string]any){
		"self-promotion":    func(e map[string]any) { e["authenticated"] = true },
		"artificial-expiry": func(e map[string]any) { e["expires_at"] = "2100-01-01T00:00:00Z" },
		"policy":            func(e map[string]any) { e["contract_version"] = "1.0.3" },
		"result":            func(e map[string]any) { e["result"] = "ready" },
		"missing-test":      func(e map[string]any) { delete(e, "test_id") },
		"invalid-time":      func(e map[string]any) { e["observed_at"] = "yesterday" },
	} {
		t.Run(name, func(t *testing.T) {
			var e map[string]any
			json.Unmarshal(payload, &e)
			edit(e)
			data, _ := json.Marshal(e)
			if _, err := ParseEvidence(data); err == nil {
				t.Fatal("invalid evidence accepted")
			}
		})
	}
	if _, err := ParseEvidence(append(payload, []byte(" {}")...)); err == nil {
		t.Fatal("trailing evidence accepted")
	}
	ctx.Subject.Dimensions = append(ctx.Subject.Dimensions, ctx.Subject.Dimensions[0])
	if _, err := Evaluate(ctx, nil); err == nil {
		t.Fatal("duplicate dimension accepted")
	}
	ctx = fixture()
	ctx.Requirements[0].Versions = []string{">=1"}
	if _, err := Evaluate(ctx, nil); err == nil {
		t.Fatal("range interpreted as exact version")
	}
	ctx = fixture()
	ctx.Now = time.Time{}
	if _, err := Evaluate(ctx, nil); err == nil {
		t.Fatal("implicit clock accepted")
	}
}

func TestDependencyCoverageUsesExactMembershipAndRequiredStates(t *testing.T) {
	states := []string{"tested-compatible", "declared-supported", "unknown", "known-incompatible"}
	want := [][]string{{states[0], states[1], states[2], states[3]}, {states[1], states[1], states[2], states[3]}, {states[2], states[2], states[2], states[3]}, {states[3], states[3], states[3], states[3]}}
	aHash, bHash, reqHash := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	expected := map[string]DependencyExpectation{"tool-a": {Required: true, SubjectSHA256: aHash, RequirementsSHA256: reqHash}, "tool-b": {Required: true, SubjectSHA256: bHash, RequirementsSHA256: reqHash}}
	reports := func(a, b string) map[string]Assessment {
		return map[string]Assessment{"tool-a": {Status: a, SubjectSHA256: aHash, RequirementsSHA256: reqHash}, "tool-b": {Status: b, SubjectSHA256: bHash, RequirementsSHA256: reqHash}}
	}
	for i, a := range states {
		for j, b := range states {
			result, err := ComposeDependencyCoverage(expected, reports(a, b))
			if err != nil || result.Status != want[i][j] {
				t.Fatalf("%s + %s: %+v %v", a, b, result, err)
			}
		}
	}
	optional := expected["tool-b"]
	optional.Required = false
	expected["tool-b"] = optional
	result, err := ComposeDependencyCoverage(expected, reports(states[0], states[3]))
	if err != nil || result.Status != states[0] {
		t.Fatal("optional failure blocked required coverage")
	}
	missing := reports(states[0], states[0])
	delete(missing, "tool-b")
	if _, err = ComposeDependencyCoverage(expected, missing); err == nil {
		t.Fatal("omitted member accepted")
	}
	missing["substituted"] = Assessment{Status: states[0]}
	if _, err = ComposeDependencyCoverage(expected, missing); err == nil {
		t.Fatal("same-count substitution accepted")
	}
	swapped := reports(states[0], states[0])
	swapped["tool-a"], swapped["tool-b"] = swapped["tool-b"], swapped["tool-a"]
	if _, err = ComposeDependencyCoverage(expected, swapped); err == nil {
		t.Fatal("swapped child scope accepted")
	}
	ctx := fixture()
	payload := record(t, &ctx, "passed", ctx.Now, nil)
	actual := assess(t, ctx, states[0], payload)
	sha, _ := SubjectDigest(ctx.Subject)
	req, _ := RequirementsDigest(ctx.Requirements)
	r, err := ComposeDependencyCoverage(map[string]DependencyExpectation{ctx.Subject.EntryID: {Required: true, SubjectSHA256: sha, RequirementsSHA256: req}}, map[string]Assessment{ctx.Subject.EntryID: actual})
	if err != nil || r.Status != states[0] {
		t.Fatalf("evaluator result did not compose: %+v %v", r, err)
	}
}

func TestScopedChangeInvalidatesEvenWhenProviderVersionIsUnchanged(t *testing.T) {
	ctx := fixture()
	old := record(t, &ctx, "passed", ctx.Now.AddDate(-1, 0, 0), nil)
	event := ctx.Now.Add(-time.Hour)
	ctx.InvalidatedAt = event
	assess(t, ctx, "unknown", old)
	// A declaration cannot erase an independently observed invalidation.
	assess(t, ctx, "unknown")
	tied := record(t, &ctx, "passed", event, nil)
	assess(t, ctx, "unknown", old, tied)
	recovered := record(t, &ctx, "passed", event.Add(time.Second), nil)
	assess(t, ctx, "tested-compatible", old, recovered)
	ctx.InvalidatedAt = ctx.Now.Add(time.Second)
	if _, err := Evaluate(ctx, nil); err == nil {
		t.Fatal("future event accepted")
	}
}
