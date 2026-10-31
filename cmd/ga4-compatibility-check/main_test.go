package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/compatibility"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/connectorcheck"
)

func TestCommandPublishesExactSignedFixtureBytesWithoutKeys(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	seedPath := filepath.Join(root, "private-seed")
	out := filepath.Join(root, "observation")
	if err = os.WriteFile(seedPath, []byte(base64.StdEncoding.EncodeToString(private.Seed())), 0o600); err != nil {
		t.Fatal(err)
	}
	release := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0")
	args := []string{"--release-dir", release, "--signing-seed-file", seedPath, "--key-id", "fixture-key", "--out", out}
	if err = run(args, io.Discard); err != nil {
		t.Fatal(err)
	}
	envelope, err := os.ReadFile(filepath.Join(out, "signed-evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy := map[string]compatibility.ProducerKey{"fixture-key": {PublicKey: base64.StdEncoding.EncodeToString(public), Producer: connectorcheck.GA4Producer, TestID: connectorcheck.GA4Test, TestVersion: connectorcheck.ProbeVersion(), EntryID: "analytics/ga4-mcp-connector", ArtifactSHA256: connectorcheck.GA4ArtifactSHA256, Operation: connectorcheck.GA4Operation, ProviderTarget: connectorcheck.GA4Target}}
	records, approved, err := compatibility.ApproveSignedEvidence([][]byte{envelope}, policy)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "evidence.json"))
	if err != nil || string(raw) != string(records[0]) {
		t.Fatal("published evidence differs from signed bytes")
	}
	data, err := os.ReadFile(filepath.Join(out, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request connectorcheck.Request
	if err = json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	assessment, err := compatibility.Evaluate(compatibility.Context{Subject: request.Subject, Requirements: request.Requirements, TrustedEvidence: approved, Now: time.Now().UTC()}, records)
	if err != nil || assessment.Status != "tested-compatible" {
		t.Fatalf("published fixture did not evaluate: %+v %v", assessment, err)
	}
	files, err := os.ReadDir(out)
	if err != nil || len(files) != 3 {
		t.Fatal("unexpected producer outputs")
	}
	for _, file := range files {
		if file.Name() != "evidence.json" && file.Name() != "signed-evidence.json" && file.Name() != "request.json" {
			t.Fatal("unintended key or policy output")
		}
	}
	if err = run(args, io.Discard); err == nil {
		t.Fatal("existing observation directory was overwritten")
	}
	unchanged, _ := os.ReadFile(filepath.Join(out, "evidence.json"))
	if string(unchanged) != string(raw) {
		t.Fatal("existing observation changed")
	}
}

func TestProfileDescriptionProvidesNoProducerApproval(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"--describe-profile"}, &out); err != nil {
		t.Fatal(err)
	}
	var policy compatibility.ProducerKey
	if err := json.Unmarshal(out.Bytes(), &policy); err != nil {
		t.Fatal(err)
	}
	if policy.PublicKey != "" || policy.ArtifactSHA256 != connectorcheck.GA4ArtifactSHA256 || policy.TestVersion != connectorcheck.ProbeVersion() {
		t.Fatal("description changed profile or invented key authority")
	}
}
