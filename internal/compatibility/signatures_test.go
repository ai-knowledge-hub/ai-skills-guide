package compatibility

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func signedFixture(t *testing.T) (Context, []byte, ed25519.PrivateKey, map[string]ProducerKey) {
	t.Helper()
	ctx := fixture()
	payload := record(t, &ctx, "passed", ctx.Now.Add(-time.Hour), nil)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ProducerKey{"fixture-key": {PublicKey: base64.StdEncoding.EncodeToString(public), Producer: "test-ci", TestID: "bounded-query", TestVersion: "1", EntryID: ctx.Subject.EntryID, ArtifactSHA256: ctx.Subject.ArtifactSHA256, Operation: ctx.Subject.Operation, ProviderTarget: ctx.Subject.ProviderTarget}}
	return ctx, payload, private, keys
}
func TestSignedEvidenceEstablishesIndependentProducerApproval(t *testing.T) {
	ctx, payload, key, policy := signedFixture(t)
	envelope, err := SignEvidence(payload, "fixture-key", key)
	if err != nil {
		t.Fatal(err)
	}
	records, approved, err := ApproveSignedEvidence([][]byte{envelope}, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || string(records[0]) != string(payload) {
		t.Fatal("signed raw bytes changed")
	}
	ctx.TrustedEvidence = approved
	assess(t, ctx, "tested-compatible", records...)
	ctx.Subject.ProviderTarget = "live-property"
	assess(t, ctx, "declared-supported", records...)
	if _, err := VerifySignedEvidence(envelope, nil); err == nil {
		t.Fatal("revoked producer key accepted")
	}
}
func TestSignedEvidenceRejectsIdentityAndSignatureSubstitutions(t *testing.T) {
	_, payload, key, policy := signedFixture(t)
	signed, err := SignEvidence(payload, "fixture-key", key)
	if err != nil {
		t.Fatal(err)
	}
	for name, alter := range map[string]func(*SignedEvidence){
		"renamed-key": func(e *SignedEvidence) { e.KeyID = "other-key" },
		"payload":     func(e *SignedEvidence) { e.Evidence = base64.StdEncoding.EncodeToString(append(payload, ' ')) },
		"signature":   func(e *SignedEvidence) { e.Signature = base64.StdEncoding.EncodeToString(make([]byte, 64)) },
		"version":     func(e *SignedEvidence) { e.SchemaVersion = "future" },
	} {
		t.Run(name, func(t *testing.T) {
			var e SignedEvidence
			json.Unmarshal(signed, &e)
			alter(&e)
			data, _ := json.Marshal(e)
			if _, err := VerifySignedEvidence(data, policy); err == nil {
				t.Fatal("signature substitution accepted")
			}
		})
	}
	public, _, _ := ed25519.GenerateKey(rand.Reader)
	p := policy["fixture-key"]
	p.PublicKey = base64.StdEncoding.EncodeToString(public)
	if _, err := VerifySignedEvidence(signed, map[string]ProducerKey{"fixture-key": p}); err == nil {
		t.Fatal("substituted public key accepted")
	}
	if _, err := VerifySignedEvidence(append(signed, []byte(" {}")...), policy); err == nil {
		t.Fatal("trailing envelope accepted")
	}
}
func TestEvenValidSignaturesCannotExpandApprovedFixtureScope(t *testing.T) {
	_, payload, key, policy := signedFixture(t)
	for name, alter := range map[string]func(*Evidence){
		"producer":     func(e *Evidence) { e.Producer = "other-ci" },
		"test":         func(e *Evidence) { e.TestID = "live-test" },
		"test-version": func(e *Evidence) { e.TestVersion = "2" },
		"entry":        func(e *Evidence) { e.Subject.EntryID = "example/other" },
		"artifact":     func(e *Evidence) { e.Subject.ArtifactSHA256 = digest("other") },
		"operation":    func(e *Evidence) { e.Subject.Operation = "live-report" },
		"target":       func(e *Evidence) { e.Subject.ProviderTarget = "live-property" },
	} {
		t.Run(name, func(t *testing.T) {
			var e Evidence
			json.Unmarshal(payload, &e)
			alter(&e)
			data, _ := json.Marshal(e)
			signed, err := SignEvidence(data, "fixture-key", key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifySignedEvidence(signed, policy); err == nil {
				t.Fatal("valid signature expanded producer scope")
			}
		})
	}
}

func TestSigningDomainAndPolicyCannotBeOmitted(t *testing.T) {
	_, payload, key, policy := signedFixture(t)
	signed, err := SignEvidence(payload, "fixture-key", key)
	if err != nil {
		t.Fatal(err)
	}
	var envelope SignedEvidence
	json.Unmarshal(signed, &envelope)
	// A valid signature for another protocol must not verify here.
	envelope.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, payload))
	data, _ := json.Marshal(envelope)
	if _, err = VerifySignedEvidence(data, policy); err == nil {
		t.Fatal("signature without the compatibility domain accepted")
	}
	json.Unmarshal(signed, &envelope)
	envelope.KeyID = "alias-key"
	policy["alias-key"] = policy["fixture-key"]
	data, _ = json.Marshal(envelope)
	if _, err = VerifySignedEvidence(data, policy); err == nil {
		t.Fatal("key ID was not cryptographically bound")
	}
	for _, field := range []string{"producer", "test_id", "test_version", "entry_id", "artifact_sha256", "operation", "provider_target"} {
		t.Run(field, func(t *testing.T) {
			var value map[string]any
			original, _ := json.Marshal(policy["fixture-key"])
			json.Unmarshal(original, &value)
			delete(value, field)
			partial, _ := json.Marshal(value)
			var p ProducerKey
			json.Unmarshal(partial, &p)
			if _, err := VerifySignedEvidence(signed, map[string]ProducerKey{"fixture-key": p}); err == nil {
				t.Fatal("incomplete producer authority accepted")
			}
		})
	}
}
