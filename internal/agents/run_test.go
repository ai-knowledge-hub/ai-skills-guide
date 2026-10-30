package agents

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestValidateCompiledAgentContractFailsClosed(t *testing.T) {
	root := t.TempDir()
	orchestration := &OrchestrationContract{
		Model:      ModelRequirement{Selection: "runtime-selected", RequiredCapabilities: []string{"tool-use"}},
		Memory:     ProfileRequirement{Mode: "runtime", Required: true},
		Governance: ProfileRequirement{Mode: "runtime", Required: true},
		Bindings:   []PackageBinding{{Name: "policy", Kind: "skill", Package: "agentops/policy", Version: "1.0.0", Requirement: "required", Availability: "resolved"}},
	}
	manifest := Manifest{ID: "marketing/demo", Version: "1.0.0", Orchestration: orchestration}
	document := compiledAgentContract{
		SchemaVersion: "skills-hub.agent-runtime/v1", Runtime: "generic",
		Agent: compiledAgentRoot{ID: manifest.ID, Version: manifest.Version}, Status: "ready",
		Model: orchestration.Model, Memory: orchestration.Memory, Governance: orchestration.Governance,
		Bindings: []compiledBinding{{Name: "policy", Kind: "skill", Package: "agentops/policy", Version: "1.0.0", Requirement: "required", Status: "missing", Path: filepath.Join(root, "skills", "agentops", "policy")}},
	}
	reasons := validateCompiledAgentContract(document, manifest, RunOptions{
		AgentsRoot: filepath.Join(root, "agents"), SkillsRoot: filepath.Join(root, "skills"), ToolsRoot: filepath.Join(root, "tools"),
	}, "generic")
	encoded, _ := json.Marshal(reasons)
	for _, fragment := range []string{"Runtime model readiness is unverified", "Required runtime memory profile is unresolved", "Required runtime governance profile is unresolved", "Required compiled skill binding policy is unresolved"} {
		if !strings.Contains(string(encoded), fragment) {
			t.Fatalf("missing fail-closed reason %q in %s", fragment, encoded)
		}
	}

	document.Status = "blocked"
	document.BlockingReasons = []string{"required agent unavailable"}
	reasons = validateCompiledAgentContract(document, manifest, RunOptions{
		AgentsRoot: filepath.Join(root, "agents"), SkillsRoot: filepath.Join(root, "skills"), ToolsRoot: filepath.Join(root, "tools"),
		MemoryPath: "memory.json", GovernancePath: "governance.json",
	}, "generic")
	encoded, _ = json.Marshal(reasons)
	if !strings.Contains(string(encoded), "required agent unavailable") {
		t.Fatalf("top-level blocked state was discarded: %s", encoded)
	}
}

func TestLoadCompiledAgentContractRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generic.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"skills-hub.agent-runtime/v1","runtime":"generic","agent":{"id":"marketing/demo","name":"Demo","version":"1.0.0","description":"Demo"},"status":"ready","blocking_reasons":[],"model":{"selection":"runtime-selected","required_capabilities":["tool-use"]},"memory":{"mode":"runtime","required":false},"governance":{"mode":"runtime","required":false},"bindings":[],"ignored":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCompiledAgentContract(path); err == nil {
		t.Fatal("expected unknown compiled contract field rejection")
	}
}

func TestRunPreflightVerifiesInstalledAgentClosureEndToEnd(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	manifest, err := registry.ValidatePackageManifest(filepath.Join(repositoryRoot, "agents", "adtech", "bi-insights-orchestrator", "agent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	entry := registry.ProjectManifest(manifest)
	indexes := writeRunAgentTestIndexes(t, repositoryRoot)
	runtimeRoot := t.TempDir()
	agentsRoot := filepath.Join(runtimeRoot, "agents")
	if err := os.MkdirAll(agentsRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	destination, err := installer.InstallSkill(filepath.Join(repositoryRoot, "agents", filepath.FromSlash(entry.ID)), agentsRoot, entry.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	dependencyResult, err := installer.InstallAgentDependencies(entry, "generic", agentsRoot, map[string]string{
		"skills": filepath.Join(repositoryRoot, "skills"), "agents": filepath.Join(repositoryRoot, "agents"), "tools": filepath.Join(repositoryRoot, "tools-mcp"),
	}, indexes, false)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	useTestRuntimeTrustRoot(t, "generic", publicKey)
	compiledPath, err := installer.CompileAgentRuntimePackage(destination, "generic", agentsRoot)
	if err != nil {
		t.Fatal(err)
	}
	contractDigest, err := installer.RuntimeContractSHA256(entry, manifest.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(destination, installer.InstallReceipt{
		Source: "local", Module: "agents", ID: entry.ID, Version: manifest.Version, Runtime: "generic",
		RuntimeContractSHA256: contractDigest, Closure: dependencyResult.Closure,
	}); err != nil {
		t.Fatal(err)
	}
	attestationPath := writeModelAttestation(t, privateKey, compiledPath, modelAttestationClaims{
		SchemaVersion: "skills-hub.model-attestation/v1", Runtime: "generic", AgentID: entry.ID, AgentVersion: manifest.Version,
		RuntimeContractSHA256: contractDigest, ModelIdentity: "openai/gpt-runtime", ModelVersion: "2026-09-26",
		Capabilities: append([]string(nil), manifest.Orchestration.Model.RequiredCapabilities...), IssuedAt: time.Now().UTC().Add(-time.Minute), ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	opts := RunOptions{
		AgentsRoot: agentsRoot, SkillsRoot: filepath.Join(runtimeRoot, "skills"), ToolsRoot: filepath.Join(runtimeRoot, "tools-mcp"),
		AgentID: entry.ID, Runtime: "generic", ApproveLive: true, AuditPath: filepath.Join(runtimeRoot, "audit.json"),
		ModelAttestationPath: attestationPath,
	}
	report, err := RunPreflight(opts)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(report.BlockingReasons, "\n")
	for _, forbidden := range []string{"receipt verification failed", "closure verification failed", "Compiled runtime", "Missing skill dependency", "Missing agent dependency"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("valid installation failed integrity preflight (%s): %#v", forbidden, report.BlockingReasons)
		}
	}
	if report.Model.Status != "ready" || report.Model.Identity != "openai/gpt-runtime" || strings.Contains(joined, "model-bound capability attestation") {
		t.Fatalf("signed model readiness was not accepted and audited: %#v", report)
	}
	auditPayload, err := os.ReadFile(opts.AuditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(auditPayload), `"status": "ready"`) || !strings.Contains(string(auditPayload), `"identity": "openai/gpt-runtime"`) {
		t.Fatalf("audit report does not record verified model identity: %s", auditPayload)
	}

	compiledPayload, err := os.ReadFile(filepath.Join(destination, ".runtime", "generic.json"))
	if err != nil {
		t.Fatal(err)
	}
	var compiled compiledAgentContract
	if err := json.Unmarshal(compiledPayload, &compiled); err != nil {
		t.Fatal(err)
	}
	var toolName, compiledEndpoint string
	for _, binding := range compiled.Bindings {
		if binding.Kind == "tool" {
			toolName, compiledEndpoint = binding.Name, binding.Endpoint
			break
		}
	}
	overridePath := filepath.Join(runtimeRoot, "bindings.json")
	if err := os.WriteFile(overridePath, []byte(fmt.Sprintf(`{"tools":{%q:{"endpoint":"mcp://attacker.example","mode":"read_only"}}}`, toolName)), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.BindingsPath = overridePath
	report, err = RunPreflight(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(report.BlockingReasons, "\n"), "compiled binding overrides are not permitted") || report.ResolvedTools[toolName].Endpoint != compiledEndpoint {
		t.Fatalf("endpoint override escaped receipt-bound binding: %#v", report)
	}
	opts.BindingsPath = ""

	memberPath := filepath.Join(opts.SkillsRoot, "adtech", "dashboard-generator", "README.md")
	payload, err := os.ReadFile(memberPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memberPath, append(payload, []byte("\ntampered\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = RunPreflight(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(report.BlockingReasons, "\n"), "closure verification failed") {
		t.Fatalf("tampered dependency did not block run-agent preflight: %#v", report.BlockingReasons)
	}
}

func TestVerifyModelAttestationRejectsReplayExpiryAndMissingCapability(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	compiledPath := filepath.Join(root, "generic.json")
	if err := os.WriteFile(compiledPath, []byte("compiled-contract"), 0o644); err != nil {
		t.Fatal(err)
	}
	compiledDigest := sha256.Sum256([]byte("compiled-contract"))
	now := time.Now().UTC()
	base := modelAttestationClaims{
		SchemaVersion: "skills-hub.model-attestation/v1", Runtime: "generic", AgentID: "marketing/demo", AgentVersion: "1.0.0",
		RuntimeContractSHA256: "runtime-digest", CompiledContractSHA256: hex.EncodeToString(compiledDigest[:]), ModelIdentity: "openai/model", ModelVersion: "v1",
		Capabilities: []string{"tool-use", "structured-output"}, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}
	contract := compiledAgentContract{Runtime: "generic", Agent: compiledAgentRoot{ID: "marketing/demo", Version: "1.0.0"}, Model: ModelRequirement{Selection: "runtime-selected", RequiredCapabilities: []string{"tool-use", "structured-output"}}}
	useTestRuntimeTrustRoot(t, "generic", publicKey)
	for name, mutate := range map[string]func(*modelAttestationClaims){
		"other agent":        func(claims *modelAttestationClaims) { claims.AgentID = "marketing/other" },
		"other target":       func(claims *modelAttestationClaims) { claims.CompiledContractSHA256 = strings.Repeat("0", 64) },
		"expired":            func(claims *modelAttestationClaims) { claims.ExpiresAt = now.Add(-time.Second) },
		"missing capability": func(claims *modelAttestationClaims) { claims.Capabilities = []string{"tool-use"} },
	} {
		t.Run(name, func(t *testing.T) {
			claims := base
			claims.Capabilities = append([]string(nil), base.Capabilities...)
			mutate(&claims)
			path := writeModelAttestation(t, privateKey, compiledPath, claims)
			if _, err := verifyModelAttestation(path, contract, compiledPath, "runtime-digest", now); err == nil {
				t.Fatalf("accepted %s attestation", name)
			}
		})
	}
	validPath := writeModelAttestation(t, privateKey, compiledPath, base)
	var forged modelAttestationDocument
	payload, err := os.ReadFile(validPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &forged); err != nil {
		t.Fatal(err)
	}
	forged.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	payload, err = json.Marshal(forged)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyModelAttestation(validPath, contract, compiledPath, "runtime-digest", now); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("accepted forged signature: %v", err)
	}
}

func TestGovernedRuntimeModelTrustCannotComeFromAgentContract(t *testing.T) {
	previous := genericModelAttestationPublicKey
	genericModelAttestationPublicKey = ""
	t.Cleanup(func() { genericModelAttestationPublicKey = previous })
	contract := compiledAgentContract{Runtime: "generic", Model: ModelRequirement{RequiredCapabilities: []string{"tool-use"}}}
	if _, err := verifyModelAttestation("ignored.json", contract, "ignored-contract.json", "digest", time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "release-governed") {
		t.Fatalf("unconfigured runtime identity did not fail closed: %v", err)
	}
}

func TestVerifiedModelDoesNotRetainStaticModelBlockingReason(t *testing.T) {
	document := compiledAgentContract{
		Status: "blocked", BlockingReasons: []string{
			"model: trusted runtime adapter attestation is required before execution",
			"required tool is unavailable",
		},
		Model: ModelRequirement{RequiredCapabilities: []string{"tool-use"}},
	}
	reasons := compiledAgentContractReadinessReasons(document, Manifest{}, RunOptions{}, true)
	joined := strings.Join(reasons, "\n")
	if strings.Contains(joined, "model") || !strings.Contains(joined, "required tool is unavailable") {
		t.Fatalf("verified model produced stale or missing blocking reasons: %v", reasons)
	}
}

func useTestRuntimeTrustRoot(t *testing.T, runtimeName string, publicKey ed25519.PublicKey) {
	t.Helper()
	encoded := base64.StdEncoding.EncodeToString(publicKey)
	switch runtimeName {
	case "codex":
		previous := codexModelAttestationPublicKey
		codexModelAttestationPublicKey = encoded
		t.Cleanup(func() { codexModelAttestationPublicKey = previous })
	case "claude":
		previous := claudeModelAttestationPublicKey
		claudeModelAttestationPublicKey = encoded
		t.Cleanup(func() { claudeModelAttestationPublicKey = previous })
	case "generic":
		previous := genericModelAttestationPublicKey
		genericModelAttestationPublicKey = encoded
		t.Cleanup(func() { genericModelAttestationPublicKey = previous })
	default:
		t.Fatalf("unsupported test runtime %q", runtimeName)
	}
}

func writeModelAttestation(t *testing.T, privateKey ed25519.PrivateKey, compiledPath string, claims modelAttestationClaims) string {
	t.Helper()
	compiledBytes, err := os.ReadFile(compiledPath)
	if err != nil {
		t.Fatal(err)
	}
	compiledDigest := sha256.Sum256(compiledBytes)
	if claims.CompiledContractSHA256 == "" {
		claims.CompiledContractSHA256 = hex.EncodeToString(compiledDigest[:])
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	document := modelAttestationDocument{modelAttestationClaims: claims, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, claimsJSON))}
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model-attestation.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestToolBindingReadinessPreservesOptionality(t *testing.T) {
	compiled := compiledToolBindings(compiledAgentContract{Bindings: []compiledBinding{{
		Name: "optional_writer", Kind: "tool", Requirement: "optional", Access: "read-write", Status: "setup-required",
	}}})
	if compiled.Tools["optional_writer"].Requirement != "optional" {
		t.Fatal("compiled tool projection discarded optional requirement")
	}
	optional := ToolBinding{Requirement: "optional", Status: "setup-required", Reason: "provider authentication is missing", Mode: "read_write"}
	blocking, warning := toolBindingReadiness("optional_writer", optional)
	if blocking != "" || !strings.Contains(warning, "Optional tool") {
		t.Fatalf("optional unavailable tool became mandatory: blocking=%q warning=%q", blocking, warning)
	}
	required := optional
	required.Requirement = "required"
	blocking, warning = toolBindingReadiness("required_writer", required)
	if blocking == "" || warning != "" {
		t.Fatalf("required unavailable tool did not block: blocking=%q warning=%q", blocking, warning)
	}
}

func TestRunPreflightOptionalUnavailableWriteToolIsWarning(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agents", "marketing", "optional-demo")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `id: marketing/optional-demo
name: Optional demo
description: Demonstrates optional tool degradation.
version: 0.1.0
released_at: "2026-09-26T00:00:00Z"
category: marketing-agents/performance
tags: [demo]
runtimes: [generic]
dependencies:
  tools: [optional_writer]
`
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte("## Workflow\n1. Continue without the optional writer.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bindingsPath := filepath.Join(root, "bindings.json")
	bindings := `{"tools":{"optional_writer":{"requirement":"optional","endpoint":"package://optional","mode":"read_write","status":"setup-required","reason":"not configured"}}}`
	if err := os.WriteFile(bindingsPath, []byte(bindings), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := RunPreflight(RunOptions{
		AgentsRoot: filepath.Join(root, "agents"), SkillsRoot: filepath.Join(root, "skills"), ToolsRoot: filepath.Join(root, "tools"),
		AgentID: "marketing/optional-demo", BindingsPath: bindingsPath, ApproveLive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "ready" || len(report.BlockingReasons) != 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "Optional tool optional_writer") || !strings.Contains(strings.Join(report.Warnings, "\n"), "Optional write-capable tool") {
		t.Fatalf("optional unavailable writer blocked the agent or was not reported: %#v", report)
	}
}

func writeRunAgentTestIndexes(t *testing.T, root string) map[string]string {
	t.Helper()
	dir := t.TempDir()
	builders := map[string]func(string) (registry.Index, error){"skills": registry.BuildSkillsIndex, "agents": registry.BuildAgentsIndex, "tools": registry.BuildToolsIndex}
	paths := map[string]string{}
	for module, build := range builders {
		index, err := build(root)
		if err != nil {
			t.Fatalf("build %s index: %v", module, err)
		}
		payload, err := json.Marshal(index)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, module+".json")
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			t.Fatal(err)
		}
		paths[module] = path
	}
	return paths
}

func TestRunPreflightReady(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agents", "marketing", "demo")
	dependencyAgentDir := filepath.Join(root, "agents", "marketing", "creative-supervisor")
	skillDir := filepath.Join(root, "skills", "adtech", "dashboard-generator")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.MkdirAll(dependencyAgentDir, 0o755); err != nil {
		t.Fatalf("mkdir dependency agent: %v", err)
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte("## Workflow\n1. Step A\n2. Step B\n"), 0o644); err != nil {
		t.Fatalf("write AGENT.md: %v", err)
	}
	manifest := `id: marketing/demo
name: Demo
description: Demo.
version: 0.1.0
released_at: "2026-03-03T00:00:00Z"
category: marketing-agents/performance
tags:
  - demo
runtimes:
  - codex
dependencies:
  agents:
    - marketing/creative-supervisor
  skills:
    - adtech/dashboard-generator
  tools:
    - ga4_query
`
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	bindingsPath := filepath.Join(root, "bindings.json")
	if err := os.WriteFile(bindingsPath, []byte(`{"tools":{"ga4_query":{"endpoint":"mcp://ga4","mode":"read_only"}}}`), 0o644); err != nil {
		t.Fatalf("write bindings: %v", err)
	}
	report, err := RunPreflight(RunOptions{
		AgentsRoot:     filepath.Join(root, "agents"),
		SkillsRoot:     filepath.Join(root, "skills"),
		ToolsRoot:      filepath.Join(root, "tools-mcp"),
		AgentID:        "marketing/demo",
		BindingsPath:   bindingsPath,
		ApproveLive:    true,
		GovernancePath: "",
	})
	if err != nil {
		t.Fatalf("run preflight: %v", err)
	}
	if report.Status != "ready" {
		t.Fatalf("expected ready, got %s (%v)", report.Status, report.BlockingReasons)
	}
	if len(report.WorkflowSteps) != 2 {
		t.Fatalf("expected workflow steps, got %#v", report.WorkflowSteps)
	}
}

func TestRunPreflightBlocksMissingCanonicalToolPackage(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agents", "marketing", "demo")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte("## Workflow\n1. Inspect\n"), 0o644); err != nil {
		t.Fatalf("write AGENT.md: %v", err)
	}
	manifest := `id: marketing/demo
name: Demo
description: Demo.
version: 0.1.0
released_at: "2026-07-25T00:00:00Z"
category: marketing-agents/performance
tags:
  - demo
runtimes:
  - codex
dependencies:
  tools:
    - adtech/missing-tool
`
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	bindings := filepath.Join(root, "bindings.json")
	if err := os.WriteFile(bindings, []byte(`{"tools":{"adtech/missing-tool":{"endpoint":"local://missing","mode":"read_only"}}}`), 0o644); err != nil {
		t.Fatalf("write bindings: %v", err)
	}
	report, err := RunPreflight(RunOptions{
		AgentsRoot:   filepath.Join(root, "agents"),
		SkillsRoot:   filepath.Join(root, "skills"),
		ToolsRoot:    filepath.Join(root, "tools-mcp"),
		AgentID:      "marketing/demo",
		BindingsPath: bindings,
		ApproveLive:  true,
	})
	if err != nil {
		t.Fatalf("run preflight: %v", err)
	}
	if report.Status != "blocked" || len(report.BlockingReasons) == 0 {
		t.Fatalf("expected missing tool to block preflight, got %#v", report)
	}
}
