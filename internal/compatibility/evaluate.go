// Package compatibility evaluates scoped compatibility observations. It never
// grants authentication, operational readiness, or permission to execute.
package compatibility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

const EvidenceVersion = "skills-hub.compatibility-evidence/v1"

type Dimension struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Subject comes from the selected artifact/lock and actual target observation,
// not from the evidence being evaluated. Digests include configuration identity.
type Subject struct {
	EntryID              string      `json:"entry_id"`
	EntryVersion         string      `json:"entry_version"`
	ArtifactSHA256       string      `json:"artifact_sha256"`
	DependencyLockSHA256 string      `json:"dependency_lock_sha256"`
	ConfigurationSHA256  string      `json:"configuration_sha256"`
	Runtime              string      `json:"runtime"`
	Platform             string      `json:"platform"`
	Operation            string      `json:"operation"`
	ProviderTarget       string      `json:"provider_target"`
	Dimensions           []Dimension `json:"dimensions"`
}

type Evidence struct {
	SchemaVersion      string    `json:"schema_version"`
	ContractVersion    string    `json:"contract_version"`
	Subject            Subject   `json:"subject"`
	RequirementsSHA256 string    `json:"requirements_sha256"`
	Producer           string    `json:"producer"`
	TestID             string    `json:"test_id"`
	TestVersion        string    `json:"test_version"`
	ObservedAt         time.Time `json:"observed_at"`
	Result             string    `json:"result"`
	Reason             string    `json:"reason"`
}

type Assessment struct {
	SubjectSHA256      string     `json:"subject_sha256,omitempty"`
	RequirementsSHA256 string     `json:"requirements_sha256,omitempty"`
	Status             string     `json:"status"`
	Reason             string     `json:"reason"`
	EvidenceSHA256     string     `json:"evidence_sha256,omitempty"`
	ObservedAt         *time.Time `json:"observed_at,omitempty"`
}

// Context is independent authority: a selected target, current requirements and
// a trust store of exact raw evidence digests to approved producer identities.
// A record's own producer field or hash is not authority to approve itself.
type Context struct {
	InvalidatedAt   time.Time // Independent scoped change event; requires strictly newer verification.
	Subject         Subject
	Requirements    []registry.CompatibilityRequirement
	TrustedEvidence map[string]string
	Now             time.Time
}

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+:/-]*$`)

func validKind(kind string) bool {
	return kind == "api" || kind == "sdk" || kind == "protocol" || kind == "runtime" || kind == "platform"
}
func digest(value any) string {
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func sorted(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return result
}
func unique(values []string) bool {
	seen := map[string]bool{}
	for _, v := range values {
		if strings.TrimSpace(v) == "" || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

func canonicalSubject(s Subject) (Subject, error) {
	if s.EntryID == "" || s.EntryVersion == "" || s.Runtime == "" || s.Platform == "" || s.Operation == "" || s.ProviderTarget == "" || len(s.Dimensions) == 0 {
		return s, fmt.Errorf("incomplete compatibility subject")
	}
	for _, v := range []string{s.ArtifactSHA256, s.DependencyLockSHA256, s.ConfigurationSHA256} {
		if !digestPattern.MatchString(v) {
			return s, fmt.Errorf("invalid subject digest")
		}
	}
	s.Dimensions = append([]Dimension{}, s.Dimensions...)
	seen := map[string]bool{}
	for i, d := range s.Dimensions {
		key := d.Kind + ":" + d.Name
		if !validKind(d.Kind) || strings.TrimSpace(d.Name) == "" || seen[key] || (!versionPattern.MatchString(d.Version) && d.Version != "") || !unique(d.Capabilities) || (d.Version == "" && len(d.Capabilities) == 0) {
			return s, fmt.Errorf("invalid or repeated target dimension")
		}
		seen[key] = true
		s.Dimensions[i].Capabilities = sorted(d.Capabilities)
	}
	sort.Slice(s.Dimensions, func(i, j int) bool {
		a, b := s.Dimensions[i], s.Dimensions[j]
		return a.Kind+":"+a.Name < b.Kind+":"+b.Name
	})
	return s, nil
}

// RequirementsDigest binds observations to all current declarations, including
// unilateral deletion. Exact version lists are alternatives; capabilities and
// separate requirements are conjunctive. Ordering and explanatory notes do not
// change this binding. This function does not infer semantic version ranges.
func RequirementsDigest(requirements []registry.CompatibilityRequirement) (string, error) {
	clean := append([]registry.CompatibilityRequirement{}, requirements...)
	seen := map[string]bool{}
	for i, r := range clean {
		key := r.Kind + ":" + r.Name
		if !validKind(r.Kind) || strings.TrimSpace(r.Name) == "" || seen[key] || !unique(r.Versions) || !unique(r.Capabilities) || (len(r.Versions) == 0 && len(r.Capabilities) == 0) {
			return "", fmt.Errorf("invalid or repeated compatibility requirement")
		}
		for _, v := range r.Versions {
			if !versionPattern.MatchString(v) {
				return "", fmt.Errorf("unsupported version identifier")
			}
		}
		seen[key] = true
		clean[i].Versions = sorted(r.Versions)
		clean[i].Capabilities = sorted(r.Capabilities)
		clean[i].Notes = ""
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i].Kind+":"+clean[i].Name < clean[j].Kind+":"+clean[j].Name })
	return digest(clean), nil
}

func includes(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// Evaluate uses exact full-combination evidence, never a union of separately
// passing API/SDK tests. Record age is informational; future records are ignored.
// Reproducible compatibility failures are distinct from transient/inconclusive
// observations. Latest applicable evidence wins, with conservative ties.
func Evaluate(ctx Context, payloads [][]byte) (assessment Assessment, err error) {
	unknown := Assessment{Status: "unknown", Reason: "no applicable compatibility evidence or support declaration"}
	if ctx.Now.IsZero() {
		return unknown, fmt.Errorf("evaluation time is required")
	}
	if !ctx.InvalidatedAt.IsZero() && ctx.InvalidatedAt.After(ctx.Now) {
		return unknown, fmt.Errorf("invalidation event is in the future")
	}
	target, err := canonicalSubject(ctx.Subject)
	if err != nil {
		return unknown, err
	}
	requirementsDigest, err := RequirementsDigest(ctx.Requirements)
	if err != nil {
		return unknown, err
	}
	defer func() {
		if err == nil {
			assessment.SubjectSHA256 = digest(target)
			assessment.RequirementsSHA256 = requirementsDigest
		}
	}()
	if len(ctx.Requirements) == 0 {
		return unknown, nil
	}
	declared := true
	for _, r := range ctx.Requirements {
		var found *Dimension
		for i := range target.Dimensions {
			d := &target.Dimensions[i]
			if d.Kind == r.Kind && d.Name == r.Name {
				found = d
				break
			}
		}
		if found == nil {
			return Assessment{Status: "unknown", Reason: "target is missing a required compatibility dimension"}, nil
		}
		if len(r.Versions) > 0 && !includes(r.Versions, found.Version) {
			declared = false
		}
		for _, capability := range r.Capabilities {
			if !includes(found.Capabilities, capability) {
				declared = false
			}
		}
	}
	var selected *Evidence
	selectedDigest := ""
	priority := map[string]int{"passed": 0, "inconclusive": 1, "incompatible": 2}
	for _, payload := range payloads {
		sum := sha256.Sum256(payload)
		sha := hex.EncodeToString(sum[:])
		producer, trusted := ctx.TrustedEvidence[sha]
		if !trusted || producer == "" {
			continue
		}
		e, err := ParseEvidence(payload)
		if err != nil {
			return unknown, fmt.Errorf("trusted compatibility evidence is invalid: %w", err)
		}
		if e.Producer != producer || e.RequirementsSHA256 != requirementsDigest || e.ObservedAt.After(ctx.Now) || (!ctx.InvalidatedAt.IsZero() && !e.ObservedAt.After(ctx.InvalidatedAt)) {
			continue
		}
		observed, err := canonicalSubject(e.Subject)
		if err != nil {
			return unknown, err
		}
		if digest(observed) != digest(target) {
			continue
		}
		if selected == nil || e.ObservedAt.After(selected.ObservedAt) || (e.ObservedAt.Equal(selected.ObservedAt) && (priority[e.Result] > priority[selected.Result] || (priority[e.Result] == priority[selected.Result] && sha > selectedDigest))) {
			copy := e
			selected = &copy
			selectedDigest = sha
		}
	}
	if selected != nil {
		status := "unknown"
		if selected.Result == "passed" && declared {
			status = "tested-compatible"
		}
		if selected.Result == "incompatible" {
			status = "known-incompatible"
		}
		return Assessment{Status: status, Reason: selected.Reason, EvidenceSHA256: selectedDigest, ObservedAt: &selected.ObservedAt}, nil
	}
	if !ctx.InvalidatedAt.IsZero() {
		return Assessment{Status: "unknown", Reason: "a scoped change invalidated prior compatibility; new verification is required"}, nil
	}
	if declared {
		return Assessment{Status: "declared-supported", Reason: "exact versions and required capabilities match maintainer declarations; no applicable trusted test evidence"}, nil
	}
	return unknown, nil
}

// ComposeDependencyCoverage evaluates the independently supplied locked member
// set. This reports child coverage only, not root workflow or live readiness.
func ComposeDependencyCoverage(expected map[string]DependencyExpectation, reports map[string]Assessment) (Assessment, error) {
	if len(expected) == 0 || len(expected) != len(reports) {
		return Assessment{}, fmt.Errorf("dependency reports must cover the exact locked membership")
	}
	status := "tested-compatible"
	required := false
	for id, expectation := range expected {
		report, ok := reports[id]
		if !ok {
			return Assessment{}, fmt.Errorf("missing locked dependency %s", id)
		}
		if !digestPattern.MatchString(expectation.SubjectSHA256) || !digestPattern.MatchString(expectation.RequirementsSHA256) || report.SubjectSHA256 != expectation.SubjectSHA256 || report.RequirementsSHA256 != expectation.RequirementsSHA256 {
			return Assessment{}, fmt.Errorf("dependency %s does not match the expected target and requirement identity", id)
		}
		switch report.Status {
		case "tested-compatible", "declared-supported", "unknown", "known-incompatible":
		default:
			return Assessment{}, fmt.Errorf("invalid dependency assessment")
		}
		if !expectation.Required {
			continue
		}
		required = true
		if report.Status == "known-incompatible" {
			status = "known-incompatible"
		} else if status != "known-incompatible" {
			if report.Status == "unknown" {
				status = "unknown"
			} else if report.Status == "declared-supported" && status != "unknown" {
				status = "declared-supported"
			}
		}
	}
	if !required {
		status = "unknown"
	}
	return Assessment{Status: status, Reason: "required dependency compatibility coverage only; evaluate the root operation separately"}, nil
}

// DependencyExpectation is derived independently from the locked member and
// its selected actual target. Swapped child reports cannot substitute authority.
type DependencyExpectation struct {
	Required           bool
	SubjectSHA256      string
	RequirementsSHA256 string
}

func SubjectDigest(subject Subject) (string, error) {
	canonical, err := canonicalSubject(subject)
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}
