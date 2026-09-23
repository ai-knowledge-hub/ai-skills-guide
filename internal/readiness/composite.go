package readiness

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	manifestschemas "github.com/ai-knowledge-hub/ai-skills-guide/shared/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const compositeEvidenceSchema = "skills-hub.composite-readiness-evidence/v1"

var (
	compositeEvidenceSchemaOnce sync.Once
	compositeEvidenceContract   *jsonschema.Schema
	compositeEvidenceSchemaErr  error
)

// CompositeReadinessEvidence is the plugin-level receipt for a workflow that
// composes independently authenticated provider tools. It deliberately stores
// only target fingerprints and evidence digests, not provider credentials or
// self-attested identity claims.
type CompositeReadinessEvidence struct {
	SchemaVersion             string                         `json:"schema_version"`
	ContractID                string                         `json:"contract_id"`
	ContractVersion           string                         `json:"contract_version"`
	EntryID                   string                         `json:"entry_id"`
	EntryVersion              string                         `json:"entry_version"`
	ArtifactSHA256            string                         `json:"artifact_sha256"`
	DependencyLockSHA256      string                         `json:"dependency_lock_sha256"`
	Runtime                   string                         `json:"runtime"`
	Platform                  string                         `json:"platform"`
	EnvironmentID             string                         `json:"environment_id"`
	ConfigurationClass        string                         `json:"configuration_class"`
	CoverageState             string                         `json:"coverage_state"`
	WorkflowTargetFingerprint string                         `json:"workflow_target_fingerprint"`
	ProviderObservations      []CompositeProviderObservation `json:"provider_observations"`
	ObservedAt                time.Time                      `json:"observed_at"`
	ExpiresAt                 time.Time                      `json:"expires_at"`
	Result                    string                         `json:"result"`
	PromotionEligible         bool                           `json:"promotion_eligible"`
	FailureReason             string                         `json:"failure_reason,omitempty"`
}

type CompositeProviderObservation struct {
	ToolID            string     `json:"tool_id"`
	ToolVersion       string     `json:"tool_version"`
	Requirement       string     `json:"requirement"`
	Access            string     `json:"access"`
	State             string     `json:"state"`
	TargetFingerprint string     `json:"target_fingerprint,omitempty"`
	EvidenceSHA256    string     `json:"evidence_sha256,omitempty"`
	ObservedAt        *time.Time `json:"observed_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	FailureReason     string     `json:"failure_reason,omitempty"`
}

// ProviderEvidenceExpectation comes from the current child-tool readiness
// receipts and the plugin dependency lock. A stored composite receipt is not
// authoritative unless every ready observation matches this current state.
type ProviderEvidenceExpectation struct {
	ToolVersion       string
	TargetFingerprint string
	EvidenceSHA256    string
	ObservedAt        time.Time
	ExpiresAt         time.Time
}

// CompositeInstallationExpectation is derived from the current install
// receipt, dependency lock, runtime selection, platform, and local environment
// identity. Receipt fields are accepted only when they match this independent
// current-installation authority exactly.
type CompositeInstallationExpectation struct {
	EntryID              string
	EntryVersion         string
	ArtifactSHA256       string
	DependencyLockSHA256 string
	Runtime              string
	Platform             string
	EnvironmentID        string
}

// ValidateCompositeReadinessEvidence applies the schema and the cross-record
// invariants JSON Schema cannot express: exact dependency membership, current
// target binding, required/optional coverage, and aggregate chronology.
func ValidateCompositeReadinessEvidence(payload []byte, installation CompositeInstallationExpectation, dependencies []registry.ProviderDependencyMetadata, expected map[string]ProviderEvidenceExpectation, now time.Time) error {
	if err := validateCompositeEvidencePayload(payload); err != nil {
		return errors.New("composite readiness evidence does not satisfy its schema")
	}
	var evidence CompositeReadinessEvidence
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err != nil {
		return errors.New("composite readiness evidence is invalid")
	}
	if evidence.EntryID != installation.EntryID || evidence.EntryVersion != installation.EntryVersion ||
		evidence.ArtifactSHA256 != installation.ArtifactSHA256 || evidence.DependencyLockSHA256 != installation.DependencyLockSHA256 ||
		evidence.Runtime != installation.Runtime || evidence.Platform != installation.Platform || evidence.EnvironmentID != installation.EnvironmentID {
		return errors.New("composite readiness evidence does not match the current installation")
	}

	requirements := make(map[string]registry.ProviderDependencyMetadata, len(dependencies))
	for _, dependency := range dependencies {
		if _, duplicate := requirements[dependency.Tool]; duplicate {
			return fmt.Errorf("provider dependency contract repeats %s", dependency.Tool)
		}
		requirements[dependency.Tool] = dependency
	}
	if len(evidence.ProviderObservations) != len(requirements) {
		return errors.New("composite readiness evidence does not cover the exact provider dependency set")
	}

	seen := make(map[string]struct{}, len(evidence.ProviderObservations))
	ready := make([]CompositeProviderObservation, 0, len(evidence.ProviderObservations))
	requiredUnavailable := false
	optionalUnavailable := false
	for _, observation := range evidence.ProviderObservations {
		requirement, ok := requirements[observation.ToolID]
		if !ok {
			return errors.New("composite readiness evidence contains an undeclared provider")
		}
		if _, duplicate := seen[observation.ToolID]; duplicate {
			return fmt.Errorf("composite readiness evidence repeats %s", observation.ToolID)
		}
		seen[observation.ToolID] = struct{}{}
		if observation.Requirement != requirement.Requirement || observation.Access != requirement.Access {
			return fmt.Errorf("composite readiness evidence changes the declared authority for %s", observation.ToolID)
		}
		if observation.State == "unavailable" {
			if requirement.Requirement == "required" {
				requiredUnavailable = true
			} else {
				optionalUnavailable = true
			}
			continue
		}
		want, ok := expected[observation.ToolID]
		if !ok || observation.ToolVersion != want.ToolVersion || observation.TargetFingerprint != want.TargetFingerprint || observation.EvidenceSHA256 != want.EvidenceSHA256 ||
			observation.ObservedAt == nil || observation.ExpiresAt == nil || !observation.ObservedAt.Equal(want.ObservedAt) || !observation.ExpiresAt.Equal(want.ExpiresAt) {
			return fmt.Errorf("composite readiness evidence is not bound to the current %s target", observation.ToolID)
		}
		if want.ObservedAt.IsZero() || want.ExpiresAt.IsZero() || want.ObservedAt.After(now) || !want.ExpiresAt.After(now) || !want.ExpiresAt.After(want.ObservedAt) {
			return fmt.Errorf("composite readiness evidence for %s is not current", observation.ToolID)
		}
		ready = append(ready, observation)
	}
	for tool := range expected {
		if _, ok := seen[tool]; !ok {
			return errors.New("current provider evidence contains an undeclared provider")
		}
	}

	wantResult := "passed"
	wantCoverage := "complete"
	if requiredUnavailable {
		wantResult = "failed"
		wantCoverage = "unavailable"
	} else if optionalUnavailable {
		wantCoverage = "partial"
	}
	if evidence.Result != wantResult || evidence.CoverageState != wantCoverage {
		return errors.New("composite readiness result does not match provider coverage")
	}
	if evidence.PromotionEligible && wantCoverage != "complete" {
		return errors.New("incomplete provider coverage cannot be promotion eligible")
	}
	if evidence.ObservedAt.After(now) {
		return errors.New("composite readiness evidence is from the future")
	}
	if len(ready) == 0 {
		if !evidence.ExpiresAt.Equal(evidence.ObservedAt) {
			return errors.New("unavailable composite evidence must expire when observed")
		}
	} else {
		latestObservation := *ready[0].ObservedAt
		earliestExpiry := *ready[0].ExpiresAt
		for _, observation := range ready[1:] {
			if observation.ObservedAt.After(latestObservation) {
				latestObservation = *observation.ObservedAt
			}
			if observation.ExpiresAt.Before(earliestExpiry) {
				earliestExpiry = *observation.ExpiresAt
			}
		}
		if !evidence.ObservedAt.Equal(latestObservation) || !evidence.ExpiresAt.Equal(earliestExpiry) {
			return errors.New("composite readiness chronology does not match its provider evidence")
		}
	}
	if evidence.WorkflowTargetFingerprint != CompositeWorkflowTargetFingerprint(ready) {
		return errors.New("composite readiness workflow target fingerprint is invalid")
	}
	return nil
}

// CompositeWorkflowTargetFingerprint produces an order-independent commitment
// to the exact tool versions, authority, targets, and child evidence receipts.
func CompositeWorkflowTargetFingerprint(observations []CompositeProviderObservation) string {
	parts := make([]string, 0, len(observations))
	for _, observation := range observations {
		if observation.State != "ready" {
			continue
		}
		parts = append(parts, strings.Join([]string{
			observation.ToolID,
			observation.ToolVersion,
			observation.Requirement,
			observation.Access,
			observation.TargetFingerprint,
			observation.EvidenceSHA256,
		}, "\x00"))
	}
	sort.Strings(parts)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func validateCompositeEvidencePayload(payload []byte) error {
	compositeEvidenceSchemaOnce.Do(func() {
		schemaPayload, err := manifestschemas.ManifestFiles.ReadFile("composite-readiness-evidence.schema.json")
		if err != nil {
			compositeEvidenceSchemaErr = err
			return
		}
		var schemaDocument any
		if err := json.Unmarshal(schemaPayload, &schemaDocument); err != nil {
			compositeEvidenceSchemaErr = err
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		const schemaURL = "https://skills.ai-knowledge-hub.org/schemas/composite-readiness-evidence.schema.json"
		if err := compiler.AddResource(schemaURL, schemaDocument); err != nil {
			compositeEvidenceSchemaErr = err
			return
		}
		compositeEvidenceContract, compositeEvidenceSchemaErr = compiler.Compile(schemaURL)
	})
	if compositeEvidenceSchemaErr != nil {
		return compositeEvidenceSchemaErr
	}
	var document any
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("composite readiness evidence contains trailing data")
	}
	return compositeEvidenceContract.Validate(document)
}
