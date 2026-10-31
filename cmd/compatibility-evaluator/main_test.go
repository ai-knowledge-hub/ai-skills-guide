package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/compatibility"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestOfflineEvaluationDistinguishesTrustWithoutGrantingReadiness(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	req := request{Subject: compatibility.Subject{EntryID: "example/client", EntryVersion: "1.0.0", ArtifactSHA256: strings.Repeat("a", 64), DependencyLockSHA256: strings.Repeat("b", 64), ConfigurationSHA256: strings.Repeat("c", 64), Runtime: "generic", Platform: "linux", Operation: "query", ProviderTarget: "sandbox", Dimensions: []compatibility.Dimension{{Kind: "api", Name: "Example", Version: "v1beta"}}}, Requirements: []registry.CompatibilityRequirement{{Kind: "api", Name: "Example", Versions: []string{"v1beta"}}}}
	root := t.TempDir()
	write := func(name string, value any) string {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	reqPath := write("request.json", req)
	binding, err := compatibility.RequirementsDigest(req.Requirements)
	if err != nil {
		t.Fatal(err)
	}
	evidence := compatibility.Evidence{SchemaVersion: compatibility.EvidenceVersion, ContractVersion: "2.0.0", Subject: req.Subject, RequirementsSHA256: binding, Producer: "fixture-ci", TestID: "query-contract", TestVersion: "1", ObservedAt: now.AddDate(-1, 0, 0), Result: "passed", Reason: "synthetic credential-free fixture"}
	evidencePath := write("evidence.json", evidence)
	data, _ := os.ReadFile(evidencePath)
	sha := sha256.Sum256(data)
	trustPath := write("trust.json", map[string]string{hex.EncodeToString(sha[:]): "fixture-ci"})
	for _, test := range []struct {
		name  string
		extra []string
		want  string
	}{
		{"declaration", nil, "declared-supported"},
		{"untrusted-record", []string{"--evidence", evidencePath}, "declared-supported"},
		{"approved-record", []string{"--evidence", evidencePath, "--trust-store", trustPath}, "tested-compatible"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			args := append([]string{"--request", reqPath}, test.extra...)
			if err := run(args, &out, now); err != nil {
				t.Fatal(err)
			}
			var result struct {
				Compatibility        compatibility.Assessment `json:"compatibility"`
				OperationalReadiness string                   `json:"operational_readiness"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Compatibility.Status != test.want || result.OperationalReadiness != "not_evaluated" {
				t.Fatalf("diagnostic misrepresented authority: %s", out.String())
			}
		})
	}
	bad := write("invalid.json", map[string]any{"subject": req.Subject, "requirements": req.Requirements, "authenticated": true})
	var out bytes.Buffer
	if err := run([]string{"--request", bad}, &out, now); err == nil || out.Len() != 0 {
		t.Fatal("request self-promotion was accepted")
	}
}
