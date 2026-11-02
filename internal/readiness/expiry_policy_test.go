package readiness

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// The independent oracle is the provider's explicit deadlines, not package age
// or an implementation lifetime constant. Each Status call runs the validator.
func TestAuthenticationUsesProviderDeadlinesBeyondThirtyDays(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name                                      string
		credentialDays, attestationDays, wantDays int
	}{
		{"credential-first", 45, 60, 45},
		{"attestation-first", 120, 45, 45},
		{"non-expiring-credential", 0, 60, 60},
		{"short-provider-deadline", 120, 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			entry := authenticatedEntry()
			value := observation(true, false, "acct-1", entry.Authentication.Scopes, now.AddDate(0, 0, test.credentialDays))
			if test.credentialDays == 0 {
				delete(value, "expires_at")
			}
			value["attestation_expires_at"] = now.AddDate(0, 0, test.attestationDays)
			payload, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			configureProfile(t, root, entry, "acct-1", string(payload), false)
			t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
			t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
			opts := InspectOptions{StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Now: now}
			report, err := Status(context.Background(), opts)
			deadline := now.AddDate(0, 0, test.wantDays)
			if err != nil || !report.Authenticated || !report.ExpiresAt.Equal(deadline) {
				t.Fatalf("provider deadline %s lost: %#v, %v", deadline, report, err)
			}
			// No implicit thirty-day boundary, but the actual deadline remains half-open.
			for _, boundary := range []struct {
				delta time.Duration
				ready bool
			}{
				{-time.Nanosecond, true}, {0, false}, {time.Nanosecond, false},
			} {
				opts.Now = deadline.Add(boundary.delta)
				current, err := Status(context.Background(), opts)
				if err != nil || current.Authenticated != boundary.ready || (!boundary.ready && current.State != "expired") {
					t.Fatalf("ready=%v at deadline %+v: %#v, %v", boundary.ready, boundary.delta, current, err)
				}
			}
		})
	}
}

func TestLongLivedCredentialStillRequiresProviderAttestationAndScope(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name, state string
		mutate      func(map[string]any)
	}{
		{"no-attestation", "expired", func(v map[string]any) { delete(v, "attestation_expires_at") }},
		{"revoked", "revoked", func(v map[string]any) { v["revoked"] = true }},
		{"wrong-scope", "wrong-scope", func(v map[string]any) { v["scopes"] = []string{"read"} }},
		{"wrong-account", "wrong-account", func(v map[string]any) { v["account"] = "another-account" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			entry := authenticatedEntry()
			value := observation(true, false, "acct-1", entry.Authentication.Scopes, now.AddDate(1, 0, 0))
			test.mutate(value)
			payload, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			configureProfile(t, root, entry, "acct-1", string(payload), false)
			t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
			t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
			report, err := Status(context.Background(), InspectOptions{StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Now: now})
			if err != nil || report.Authenticated || report.State != test.state {
				t.Fatalf("authority protection changed: %#v, %v", report, err)
			}
		})
	}
}
