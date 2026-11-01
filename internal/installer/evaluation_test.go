package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestEvaluationsAdmitSameRetainedPackageConcurrentlyAtDifferentInstants(t *testing.T) {
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0", "package.tar.gz")
	extracted := t.TempDir()
	if err := extractVerifiedTarGz(archive, extracted); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(extracted, "tool.yaml")
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	// GA4 0.2.0 records 2026-09-17. Operational v1 gives authenticated
	// manifest evidence 30 days, valid strictly before its deadline.
	expires := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	t.Run("isolated", func(t *testing.T) {
		for _, test := range []struct {
			name    string
			instant time.Time
			valid   bool
		}{
			{"before", expires.Add(-time.Nanosecond), true},
			{"at", expires, false},
			{"after", expires.Add(time.Nanosecond), false},
		} {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				evaluation := EvaluationAt(test.instant)
				for i := 0; i < 10; i++ {
					_, err := evaluation.ValidatePackageManifest(manifestPath)
					if (err == nil) != test.valid {
						t.Fatalf("admissible=%v at %s: %v", test.valid, test.instant, err)
					}
					if !test.valid && !strings.Contains(err.Error(), "evidence older than") {
						t.Fatalf("unrelated rejection: %v", err)
					}
				}
			})
		}
	})
	after, err := os.ReadFile(manifestPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("evaluations changed retained manifest bytes: %v", err)
	}
}

func TestEvaluationDefaultsUseRealTimeWithoutChangingExplicitContexts(t *testing.T) {
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0", "package.tar.gz")
	extracted := t.TempDir()
	if err := extractVerifiedTarGz(archive, extracted); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(extracted, "tool.yaml")
	// Only the scratch fixture changes; retained archive bytes stay immutable.
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	payload = bytes.Replace(payload, []byte(`last_verified_at: "2026-09-17T00:00:00Z"`), []byte(`last_verified_at: "2000-01-01T00:00:00Z"`), 1)
	if err := os.WriteFile(manifestPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	explicit := EvaluationAt(time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))
	if _, err := explicit.ValidatePackageManifest(manifestPath); err != nil {
		t.Fatalf("explicit control: %v", err)
	}
	for name, evaluation := range map[string]Evaluation{"zero value": {}, "zero instant": EvaluationAt(time.Time{})} {
		if _, err := evaluation.ValidatePackageManifest(manifestPath); err == nil || !strings.Contains(err.Error(), "evidence older than") {
			t.Fatalf("%s did not use real time: %v", name, err)
		}
	}
	if _, err := registry.ValidatePackageManifest(manifestPath); err == nil || !strings.Contains(err.Error(), "evidence older than") {
		t.Fatalf("default registry admission did not use real time: %v", err)
	}
	if _, err := explicit.ValidatePackageManifest(manifestPath); err != nil {
		t.Fatalf("default evaluation contaminated explicit context: %v", err)
	}
}
