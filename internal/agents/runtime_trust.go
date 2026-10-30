package agents

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// Runtime model-attestation trust roots are release-governed linker inputs.
// Governed builds require all three values; direct development builds remain
// fail-closed. They are deliberately not runtime flags, environment variables,
// install options, or agent-package content.
var (
	codexModelAttestationPublicKey   string
	claudeModelAttestationPublicKey  string
	genericModelAttestationPublicKey string
)

func governedRuntimeModelTrustRoot(runtimeName string) (ed25519.PublicKey, error) {
	encoded := map[string]string{
		"codex":   codexModelAttestationPublicKey,
		"claude":  claudeModelAttestationPublicKey,
		"generic": genericModelAttestationPublicKey,
	}[strings.ToLower(strings.TrimSpace(runtimeName))]
	if strings.TrimSpace(encoded) == "" {
		return nil, fmt.Errorf("the selected runtime has no release-governed model-attestation identity")
	}
	publicKey, err := ValidateModelAttestationPublicKey(encoded)
	if err != nil {
		return nil, fmt.Errorf("the selected runtime's governed model-attestation identity is invalid")
	}
	return publicKey, nil
}

// ValidateModelAttestationPublicKey validates the release-time trust-root
// representation accepted by the runtime. Requiring canonical standard base64
// makes the Makefile linker inputs safe to pass as single values and prevents a
// build from succeeding with keys the resulting binary cannot use.
func ValidateModelAttestationPublicKey(encoded string) (ed25519.PublicKey, error) {
	if encoded == "" || strings.TrimSpace(encoded) != encoded {
		return nil, fmt.Errorf("must be canonical standard base64")
	}
	publicKey, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || base64.StdEncoding.EncodeToString(publicKey) != encoded {
		return nil, fmt.Errorf("must be canonical standard base64")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("must decode to a %d-byte Ed25519 public key", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(publicKey), nil
}
