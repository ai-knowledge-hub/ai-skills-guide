package compatibility

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

const SignedEvidenceVersion = "skills-hub.signed-compatibility-evidence/v1"
const signatureDomain = "skills-hub/compatibility-evidence/ed25519/v1\x00"

type SignedEvidence struct {
	SchemaVersion string `json:"schema_version"`
	KeyID         string `json:"key_id"`
	Evidence      string `json:"evidence"`
	Signature     string `json:"signature"`
}

// ProducerKey is independently configured by the verifier/operator. A record
// cannot authorize its own key, producer, test or fixture/live scope.
type ProducerKey struct {
	EntryID        string `json:"entry_id"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	PublicKey      string `json:"public_key"`
	Producer       string `json:"producer"`
	TestID         string `json:"test_id"`
	TestVersion    string `json:"test_version"`
	Operation      string `json:"operation"`
	ProviderTarget string `json:"provider_target"`
}

func signatureMessage(keyID string, payload []byte) []byte {
	return append([]byte(signatureDomain+keyID+"\x00"), payload...)
}

func SignEvidence(payload []byte, keyID string, key ed25519.PrivateKey) ([]byte, error) {
	if keyID == "" || len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid producer signing identity")
	}
	if _, err := ParseEvidence(payload); err != nil {
		return nil, err
	}
	envelope := SignedEvidence{SchemaVersion: SignedEvidenceVersion, KeyID: keyID, Evidence: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, signatureMessage(keyID, payload)))}
	return json.Marshal(envelope)
}

// VerifySignedEvidence establishes producer approval, not compatibility or live
// readiness. Evaluate must still bind the approved bytes to an independent target.
func VerifySignedEvidence(envelopeJSON []byte, keys map[string]ProducerKey) ([]byte, error) {
	if len(envelopeJSON) > 2*1024*1024 {
		return nil, fmt.Errorf("signed evidence exceeds 2 MiB")
	}
	var envelope SignedEvidence
	decoder := json.NewDecoder(bytes.NewReader(envelopeJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("invalid signed evidence envelope")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing signed evidence")
	}
	if envelope.SchemaVersion != SignedEvidenceVersion || envelope.KeyID == "" {
		return nil, fmt.Errorf("unsupported signed evidence identity")
	}
	policy, ok := keys[envelope.KeyID]
	if !ok || policy.EntryID == "" || !digestPattern.MatchString(policy.ArtifactSHA256) || policy.Producer == "" || policy.TestID == "" || policy.TestVersion == "" || policy.Operation == "" || policy.ProviderTarget == "" {
		return nil, fmt.Errorf("producer key is unknown or scope policy is incomplete")
	}
	public, err := base64.StdEncoding.DecodeString(policy.PublicKey)
	if err != nil || len(public) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid producer public key")
	}
	payload, err := base64.StdEncoding.DecodeString(envelope.Evidence)
	if err != nil || len(payload) > 1024*1024 {
		return nil, fmt.Errorf("invalid signed evidence payload")
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil || !ed25519.Verify(public, signatureMessage(envelope.KeyID, payload), signature) {
		return nil, fmt.Errorf("producer signature verification failed")
	}
	evidence, err := ParseEvidence(payload)
	if err != nil {
		return nil, err
	}
	if evidence.Subject.EntryID != policy.EntryID || evidence.Subject.ArtifactSHA256 != policy.ArtifactSHA256 || evidence.Producer != policy.Producer || evidence.TestID != policy.TestID || evidence.TestVersion != policy.TestVersion || evidence.Subject.Operation != policy.Operation || evidence.Subject.ProviderTarget != policy.ProviderTarget {
		return nil, fmt.Errorf("evidence is outside the producer key's approved test scope")
	}
	return payload, nil
}

// ApproveSignedEvidence adapts cryptographic verification to the evaluator's
// raw-byte approval boundary. Policy never comes from the signed envelope.
func ApproveSignedEvidence(envelopes [][]byte, keys map[string]ProducerKey) ([][]byte, map[string]string, error) {
	approved := map[string]string{}
	var records [][]byte
	for _, envelope := range envelopes {
		payload, err := VerifySignedEvidence(envelope, keys)
		if err != nil {
			return nil, nil, err
		}
		evidence, err := ParseEvidence(payload)
		if err != nil {
			return nil, nil, err
		}
		sha := sha256.Sum256(payload)
		approved[hex.EncodeToString(sha[:])] = evidence.Producer
		records = append(records, payload)
	}
	return records, approved, nil
}
