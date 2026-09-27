package agents

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestValidateModelAttestationPublicKey(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	if _, err := ValidateModelAttestationPublicKey(valid); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
	for name, value := range map[string]string{
		"empty":        "",
		"malformed":    "x/y/z",
		"wrong length": base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize-1)),
		"whitespace":   " " + valid,
		"url alphabet": base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xfb}, ed25519.PublicKeySize)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateModelAttestationPublicKey(value); err == nil {
				t.Fatalf("invalid key %q accepted", value)
			}
		})
	}
}
