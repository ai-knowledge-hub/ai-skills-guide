package connectorcheck

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/compatibility"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestRetainedGA4ReleaseProducesOnlyFixtureCompatibility(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	release := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0")
	t.Setenv("GA4_CREDENTIAL", "must-not-reach-child")
	t.Setenv("NODE_OPTIONS", "--invalid-ambient-option-must-not-reach-child")
	request, payload, err := RunGA4(context.Background(), release, "node", now)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := compatibility.ParseEvidence(payload)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Result != "passed" || evidence.Subject.EntryVersion != "0.2.0" || evidence.Subject.ArtifactSHA256 != GA4ArtifactSHA256 {
		t.Fatalf("unexpected fixture result: %+v", evidence)
	}
	if evidence.Subject.Operation != GA4Operation || evidence.Subject.ProviderTarget != GA4Target {
		t.Fatal("fixture scope lost")
	}
	// Independent profile oracle: the reviewed archive and Google runReport
	// contract use v1beta; the selected MCP handler negotiation is 2025-06-18.
	versions := map[string]string{}
	for _, dimension := range request.Subject.Dimensions {
		versions[dimension.Kind+":"+dimension.Name] = dimension.Version
	}
	if versions["api:Google Analytics Data API"] != "v1beta" || versions["protocol:MCP"] != "2025-06-18" {
		t.Fatal("publisher changed the tested API/protocol identity")
	}
	versionCommand := exec.Command("node", "--version")
	versionCommand.Env = []string{}
	versionBytes, err := versionCommand.Output()
	if err != nil {
		t.Fatal(err)
	}
	if versions["runtime:Node.js"] != strings.TrimPrefix(strings.TrimSpace(string(versionBytes)), "v") {
		t.Fatal("publisher invented a Node runtime version")
	}
	for _, d := range request.Subject.Dimensions {
		if d.Kind == "sdk" {
			t.Fatal("invented provider SDK")
		}
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := compatibility.SignEvidence(payload, "ci-test-key", private)
	if err != nil {
		t.Fatal(err)
	}
	policy := map[string]compatibility.ProducerKey{"ci-test-key": {PublicKey: base64.StdEncoding.EncodeToString(public), Producer: GA4Producer, TestID: GA4Test, TestVersion: ProbeVersion(), EntryID: request.Subject.EntryID, ArtifactSHA256: GA4ArtifactSHA256, Operation: GA4Operation, ProviderTarget: GA4Target}}
	records, approved, err := compatibility.ApproveSignedEvidence([][]byte{envelope}, policy)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compatibility.Evaluate(compatibility.Context{Subject: request.Subject, Requirements: request.Requirements, TrustedEvidence: approved, Now: now.Add(time.Second)}, records)
	if err != nil || result.Status != "tested-compatible" {
		t.Fatalf("real artifact failed verified evaluation: %+v %v", result, err)
	}
	request.Subject.Operation = "live-run-report"
	result, err = compatibility.Evaluate(compatibility.Context{Subject: request.Subject, Requirements: request.Requirements, TrustedEvidence: approved, Now: now.Add(time.Second)}, records)
	if err != nil || result.Status == "tested-compatible" {
		t.Fatal("fixture observation promoted live operation")
	}
	// The exact version is measured, not copied from a declaration or test constant.
	data, _ := json.Marshal(request)
	if len(data) == 0 {
		t.Fatal("request could not be exported")
	}
}

func TestGA4ProfileRejectsMutatedArchiveBeforeCheckingIt(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0")
	archive, err := os.ReadFile(filepath.Join(source, "package.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	archive[0] ^= 1
	if err = os.WriteFile(filepath.Join(root, "package.tar.gz"), archive, 0o600); err != nil {
		t.Fatal(err)
	}
	_, payload, err := RunGA4(context.Background(), root, "node", time.Now().UTC())
	if err == nil || payload != nil {
		t.Fatal("mutated archive received an observation")
	}
}

func TestHostOracleDetectsChangedAPIContract(t *testing.T) {
	release := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0")
	root := t.TempDir()
	modulePath := filepath.Join(root, "changed-connector.mjs")
	probePath := filepath.Join(root, "probe.mjs")
	_, err := installer.InspectRetainedRelease(filepath.Join(release, "package.tar.gz"), filepath.Join(release, "tool.yaml"), "tools", "analytics/ga4-mcp-connector", "0.2.0", func(extracted string, _ registry.Manifest) error {
		source, err := os.ReadFile(filepath.Join(extracted, "scripts", "ga4_mcp_server.mjs"))
		if err != nil {
			return err
		}
		changed := strings.Replace(string(source), "/v1beta/properties/", "/v2/properties/", 1)
		if changed == string(source) {
			return fmt.Errorf("API endpoint mutation did not apply")
		}
		return os.WriteFile(modulePath, []byte(changed), 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(probePath, ga4Probe, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", probePath, modulePath)
	command.Env = []string{}
	output, err := command.Output()
	if err != nil {
		t.Fatal("host oracle did not complete")
	}
	var result struct {
		Result string `json:"result"`
	}
	if err = json.Unmarshal(output, &result); err != nil || result.Result != "incompatible" {
		t.Fatalf("changed endpoint was not detected: %s", output)
	}
}
