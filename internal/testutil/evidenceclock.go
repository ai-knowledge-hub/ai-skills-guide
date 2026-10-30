package testutil

import (
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/evidenceclock"
)

// PackageEvidenceAt isolates historical package fixtures from the calendar.
// Use only in serial tests: this changes admission time for the test process.
// Fixture bytes, receipt hashes, and all freshness rules remain unchanged.
func PackageEvidenceAt(t *testing.T, instant time.Time) {
	t.Helper()
	previous := evidenceclock.Now
	evidenceclock.Now = func() time.Time { return instant }
	t.Cleanup(func() { evidenceclock.Now = previous })
}
