// Package connectorcheck runs host-owned compatibility probes against retained
// artifacts. Its GA4 probe is strictly scoped to a local fixture contract.
package connectorcheck

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/compatibility"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

//go:embed ga4-probe.mjs
var ga4Probe []byte

const GA4Producer = "skills-hub/ga4-fixture-contract"
const GA4Test = "ga4/run-report-fixture-contract"
const GA4Operation = "ga4-fixture-contract"
const GA4Target = "host-owned-local-fixture"
const ga4ID = "analytics/ga4-mcp-connector"

// This reviewed profile is deliberately release-specific, not a claim about
// arbitrary future archives using the same package name.
const GA4ArtifactSHA256 = "80efc8f9e5176097781da84595a9271b34cc5f1661b098422664cf739e154b84"

// ProbeVersion changes whenever the host-owned oracle changes.
func ProbeVersion() string    { return "1-" + hash(ga4Probe)[:16] }
func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

type Request struct {
	Subject      compatibility.Subject               `json:"subject"`
	Requirements []registry.CompatibilityRequirement `json:"requirements"`
}

// RunGA4 checks a snapshot of a retained artifact without installing it or
// using ambient provider credentials. It emits no live or SDK compatibility claim.
func RunGA4(ctx context.Context, releaseDir, node string, observedAt time.Time) (Request, []byte, error) {
	var request Request
	if observedAt.IsZero() {
		return request, nil, fmt.Errorf("observation time is required")
	}
	nodePath, err := exec.LookPath(node)
	if err != nil {
		return request, nil, fmt.Errorf("Node.js is required")
	}
	private, err := os.MkdirTemp("", "ga4-compatibility-")
	if err != nil {
		return request, nil, err
	}
	defer os.RemoveAll(private)
	// Snapshot before validation so the recorded digest and inspected bytes agree.
	archive, err := snapshot(filepath.Join(releaseDir, "package.tar.gz"), 100<<20)
	if err != nil {
		return request, nil, err
	}
	if hash(archive) != GA4ArtifactSHA256 {
		return request, nil, fmt.Errorf("archive is outside the reviewed GA4 fixture profile")
	}
	manifestBytes, err := snapshot(filepath.Join(releaseDir, "tool.yaml"), 1<<20)
	if err != nil {
		return request, nil, err
	}
	archivePath, manifestPath := filepath.Join(private, "package.tar.gz"), filepath.Join(private, "tool.yaml")
	if err = os.WriteFile(archivePath, archive, 0o600); err != nil {
		return request, nil, err
	}
	if err = os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		return request, nil, err
	}
	// The sidecar is parsed for version selection, then validated against archive bytes.
	_, err = registry.ParseManifest(manifestPath)
	if err != nil {
		return request, nil, err
	}
	scriptPath := filepath.Join(private, "probe.mjs")
	if err = os.WriteFile(scriptPath, ga4Probe, 0o600); err != nil {
		return request, nil, err
	}
	var observation struct {
		Result      string `json:"result"`
		NodeVersion string `json:"node_version"`
	}
	manifest, err := installer.InspectRetainedRelease(archivePath, manifestPath, "tools", ga4ID, "0.2.0", func(root string, m registry.Manifest) error {
		if m.Execution.Kind != "mcp-server" {
			return fmt.Errorf("GA4 fixture probe requires the packaged MCP server")
		}
		deadline, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		command := exec.CommandContext(deadline, nodePath, scriptPath, filepath.Join(root, "scripts", "ga4_mcp_server.mjs"))
		command.Dir = root
		command.Env = []string{}
		output, err := command.Output()
		if err != nil {
			return fmt.Errorf("GA4 host probe did not complete; no observation issued")
		}
		if len(output) > 4096 {
			return fmt.Errorf("unexpected probe output size")
		}
		if err := json.Unmarshal(output, &observation); err != nil {
			return fmt.Errorf("invalid host probe output")
		}
		return nil
	})
	if err != nil {
		return request, nil, err
	}
	major, err := strconv.Atoi(strings.Split(observation.NodeVersion, ".")[0])
	if err != nil || major < 22 {
		return request, nil, fmt.Errorf("fixture producer requires Node.js 22 or newer")
	}
	switch observation.Result {
	case "passed", "incompatible", "inconclusive":
	default:
		return request, nil, fmt.Errorf("invalid host probe result")
	}
	requirements := []registry.CompatibilityRequirement{
		{Kind: "api", Name: "Google Analytics Data API", Versions: []string{"v1beta"}},
		{Kind: "protocol", Name: "MCP", Versions: []string{"2024-11-05", "2025-03-26", "2025-06-18"}, Capabilities: []string{"initialize-handler", "tools-call-handler"}},
		{Kind: "runtime", Name: "Node.js", Versions: []string{observation.NodeVersion}},
	}
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	lockIdentity, err := json.Marshal(manifest.Artifact.DependencyLock)
	if err != nil {
		return request, nil, err
	}
	subject := compatibility.Subject{EntryID: manifest.ID, EntryVersion: manifest.Version, ArtifactSHA256: hash(archive), DependencyLockSHA256: hash(lockIdentity), ConfigurationSHA256: hash(ga4Probe), Runtime: "generic", Platform: platform, Operation: GA4Operation, ProviderTarget: GA4Target, Dimensions: []compatibility.Dimension{
		{Kind: "api", Name: "Google Analytics Data API", Version: "v1beta"},
		{Kind: "protocol", Name: "MCP", Version: "2025-06-18", Capabilities: []string{"initialize-handler", "tools-call-handler"}},
		{Kind: "runtime", Name: "Node.js", Version: observation.NodeVersion},
	}}
	binding, err := compatibility.RequirementsDigest(requirements)
	if err != nil {
		return request, nil, err
	}
	evidence := compatibility.Evidence{SchemaVersion: compatibility.EvidenceVersion, ContractVersion: "2.0.0", Subject: subject, RequirementsSHA256: binding, Producer: GA4Producer, TestID: GA4Test, TestVersion: ProbeVersion(), ObservedAt: observedAt.UTC(), Result: observation.Result, Reason: "Host-owned local GA4 API/MCP fixture contract only; no provider SDK, live provider or credential verification."}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return request, nil, err
	}
	if _, err = compatibility.ParseEvidence(payload); err != nil {
		return request, nil, err
	}
	return Request{Subject: subject, Requirements: requirements}, payload, nil
}

func snapshot(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("retained release input must be a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("retained release input exceeds size limit")
	}
	return data, nil
}
