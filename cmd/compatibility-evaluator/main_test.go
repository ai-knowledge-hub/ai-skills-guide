package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

func TestSignedObservationModeRequiresSeparatePolicy(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	req := request{Subject: compatibility.Subject{EntryID: "example/client", EntryVersion: "1.0.0", ArtifactSHA256: strings.Repeat("a", 64), DependencyLockSHA256: strings.Repeat("b", 64), ConfigurationSHA256: strings.Repeat("c", 64), Runtime: "generic", Platform: "linux", Operation: "fixture-contract", ProviderTarget: "local-fixture", Dimensions: []compatibility.Dimension{{Kind: "api", Name: "Example", Version: "v1"}}}, Requirements: []registry.CompatibilityRequirement{{Kind: "api", Name: "Example", Versions: []string{"v1"}}}}
	binding, _ := compatibility.RequirementsDigest(req.Requirements)
	evidence := compatibility.Evidence{SchemaVersion: compatibility.EvidenceVersion, ContractVersion: "2.0.0", Subject: req.Subject, RequirementsSHA256: binding, Producer: "fixture-ci", TestID: "fixture-test", TestVersion: "1", ObservedAt: now.Add(-time.Minute), Result: "passed", Reason: "local fixture only"}
	payload, _ := json.Marshal(evidence)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := compatibility.SignEvidence(payload, "producer-key", private)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	write := func(name string, data []byte) string {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	reqData, _ := json.Marshal(req)
	reqPath := write("request", reqData)
	signedPath := write("signed", signed)
	policy := map[string]compatibility.ProducerKey{"producer-key": {PublicKey: base64.StdEncoding.EncodeToString(public), Producer: "fixture-ci", TestID: "fixture-test", TestVersion: "1", EntryID: req.Subject.EntryID, ArtifactSHA256: req.Subject.ArtifactSHA256, Operation: req.Subject.Operation, ProviderTarget: req.Subject.ProviderTarget}}
	policyData, _ := json.Marshal(policy)
	policyPath := write("policy", policyData)
	var out bytes.Buffer
	args := []string{"--request", reqPath, "--signed-evidence", signedPath, "--producer-policy", policyPath}
	if err := run(args, &out, now); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"status":"tested-compatible"`) || !strings.Contains(out.String(), `"operational_readiness":"not_evaluated"`) {
		t.Fatalf("signed mode misrepresented result: %s", out.String())
	}
	for _, extra := range [][]string{
		{"--signed-evidence", signedPath},
		{"--signed-evidence", signedPath, "--producer-policy", policyPath, "--evidence", signedPath},
		{"--producer-policy", policyPath},
	} {
		out.Reset()
		if err := run(append([]string{"--request", reqPath}, extra...), &out, now); err == nil || out.Len() != 0 {
			t.Fatal("ambiguous or policy-free signed input accepted")
		}
	}
	revoked := write("revoked-policy", []byte(`{}`))
	out.Reset()
	if err := run([]string{"--request", reqPath, "--signed-evidence", signedPath, "--producer-policy", revoked}, &out, now); err == nil || out.Len() != 0 {
		t.Fatal("revoked key produced an assessment")
	}
}
