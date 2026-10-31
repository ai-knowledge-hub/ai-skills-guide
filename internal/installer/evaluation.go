package installer

import (
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

// Evaluation carries one immutable instant through package admission, closure
// planning and compilation. It does not control receipt or provider clocks.
// The zero value captures real time when an operation starts.
type Evaluation struct {
	at time.Time
}

// EvaluationAt creates an isolated operation context. Zero selects real time.
// Explicit instants are for trusted internal callers and historical tests;
// they are never populated from CLI flags or environment variables.
func EvaluationAt(at time.Time) Evaluation {
	if at.IsZero() {
		at = time.Now()
	}
	return Evaluation{at: at.UTC()}
}

func (evaluation Evaluation) normalized() Evaluation {
	if evaluation.at.IsZero() {
		return EvaluationAt(time.Time{})
	}
	return evaluation
}

func (evaluation Evaluation) ValidatePackageManifest(path string) (registry.Manifest, error) {
	evaluation = evaluation.normalized()
	return registry.ValidatePackageManifestAt(path, evaluation.at)
}
