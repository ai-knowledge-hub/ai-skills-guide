package installer

import (
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

// Evaluation isolates a clock source for package admission and commit checks.
// Zero uses live time; an explicit instant supports fixed historical tests.
// It does not control receipt or provider clocks.
type Evaluation struct {
	at  time.Time
	now func() time.Time
}

// EvaluationAt creates an isolated operation context. Zero selects live time.
// Explicit instants are for trusted internal callers and historical tests;
// they are never populated from CLI flags or environment variables.
func EvaluationAt(at time.Time) Evaluation {
	return evaluationWithClock(at, time.Now)
}

// Tests can supply an advancing clock local to one operation. Production uses
// the same constructor with time.Now; neither source is process-global state.
func evaluationWithClock(at time.Time, now func() time.Time) Evaluation {
	return Evaluation{at: at.UTC(), now: now}
}

func (evaluation Evaluation) instant() time.Time {
	if !evaluation.at.IsZero() {
		return evaluation.at
	}
	if evaluation.now != nil {
		return evaluation.now().UTC()
	}
	return time.Now().UTC()
}

func (evaluation Evaluation) ValidatePackageManifest(path string) (registry.Manifest, error) {
	return registry.ValidatePackageManifestAt(path, evaluation.instant())
}
