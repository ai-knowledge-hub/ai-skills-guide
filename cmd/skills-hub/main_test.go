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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestPrintPluginSummary(t *testing.T) {
	entry := registry.SkillEntry{
		ID: "marketing/performance-reporting-plugin",
		Includes: &registry.IncludeSet{
			Skills: []string{"marketing/meta-google-weekly-performance-review", "adtech/dashboard-generator"},
			Agents: []string{"marketing/weekly-performance-supervisor"},
			Tools:  []string{"analytics/ga4-mcp-connector"},
			Hooks:  []string{"post-analysis-slack-summary"},
		},
		Requires: &registry.RequirementSet{
			Secrets:   []string{"GA4_PROPERTY_ID", "SLACK_WEBHOOK_URL"},
			Approvals: []string{"human-review-for-write-actions"},
		},
	}

	var out bytes.Buffer
	printPluginSummary(&out, entry)
	rendered := out.String()

	expected := []string{
		"includes.skills: 2",
		"includes.skills.list: marketing/meta-google-weekly-performance-review, adtech/dashboard-generator",
		"includes.agents: 1",
		"includes.agents.list: marketing/weekly-performance-supervisor",
		"includes.tools: 1",
		"includes.tools.list: analytics/ga4-mcp-connector",
		"includes.hooks: 1",
		"includes.hooks.list: post-analysis-slack-summary",
		"requires.secrets: GA4_PROPERTY_ID, SLACK_WEBHOOK_URL",
		"requires.approvals: human-review-for-write-actions",
	}
	for _, needle := range expected {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected output to contain %q, got:\n%s", needle, rendered)
		}
	}
}

func TestRunAgentRejectsCallerSelfAttestedModelCapabilities(t *testing.T) {
	err := runAgent([]string{"--agent", "marketing/example", "--model-capability", "tool-use"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("caller-supplied model capability was accepted: %v", err)
	}
}

func TestInstallRejectsCallerSelectedModelAttestationAuthority(t *testing.T) {
	err := runInstall([]string{"--module", "agents", "--entry", "marketing/example", "--model-attestation-public-key", "caller-key"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("install caller could select model attestation authority: %v", err)
	}
}

func TestResolveRunAgentRootsUsesRuntimeInstallTargetsAndPreservesOverrides(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "Codex Home")
	t.Setenv("CODEX_HOME", codexHome)
	agentsRoot, skillsRoot, toolsRoot, err := resolveRunAgentRoots("codex", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if agentsRoot != filepath.Join(codexHome, "agents") || skillsRoot != filepath.Join(codexHome, "skills") || toolsRoot != filepath.Join(codexHome, "tools-mcp") {
		t.Fatalf("run-agent roots do not match installer targets: agents=%q skills=%q tools=%q", agentsRoot, skillsRoot, toolsRoot)
	}
	explicitAgents := filepath.Join(t.TempDir(), "custom agents")
	explicitSkills := filepath.Join(t.TempDir(), "custom skills")
	explicitTools := filepath.Join(t.TempDir(), "custom tools")
	agentsRoot, skillsRoot, toolsRoot, err = resolveRunAgentRoots("generic", explicitAgents, explicitSkills, explicitTools)
	if err != nil {
		t.Fatal(err)
	}
	for label, gotWant := range map[string][2]string{
		"agents": {agentsRoot, explicitAgents}, "skills": {skillsRoot, explicitSkills}, "tools": {toolsRoot, explicitTools},
	} {
		want, absErr := filepath.Abs(gotWant[1])
		if absErr != nil {
			t.Fatal(absErr)
		}
		if gotWant[0] != want {
			t.Fatalf("%s override = %q, want %q", label, gotWant[0], want)
		}
	}
	if _, _, _, err := resolveRunAgentRoots("generic", "", "", ""); err == nil || !strings.Contains(err.Error(), "--target is required") {
		t.Fatalf("generic runtime inferred source-tree roots: %v", err)
	}
}

func TestBuiltBinaryAcceptsRuntimeSignedModelAttestation(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository root")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	binaryPath := filepath.Join(root, "skills-hub")
	linkerValue := "github.com/ai-knowledge-hub/ai-skills-guide/internal/agents.genericModelAttestationPublicKey=" + base64.StdEncoding.EncodeToString(publicKey)
	build := exec.Command("go", "build", "-ldflags", "-X "+linkerValue, "-o", binaryPath, "./cmd/skills-hub")
	build.Dir = repositoryRoot
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(root, "go-cache"))
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build governed test binary: %v\n%s", err, output)
	}

	agentsRoot := filepath.Join(root, "runtime", "agents")
	install := exec.Command(binaryPath, "install", "--module", "agents", "--entry", "marketing/creative-operating-system-supervisor@0.3.0", "--runtime", "generic", "--target", agentsRoot)
	install.Dir = repositoryRoot
	if output, err := install.CombinedOutput(); err != nil {
		t.Fatalf("install agent with built binary: %v\n%s", err, output)
	}
	agentDir := filepath.Join(agentsRoot, "marketing", "creative-operating-system-supervisor")
	var receipt struct {
		RuntimeContractSHA256 string `json:"runtime_contract_sha256"`
	}
	receiptPayload, err := os.ReadFile(filepath.Join(agentDir, ".skills-hub-install.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(receiptPayload, &receipt); err != nil {
		t.Fatal(err)
	}
	compiledPath := filepath.Join(agentDir, ".runtime", "generic.json")
	compiledPayload, err := os.ReadFile(compiledPath)
	if err != nil {
		t.Fatal(err)
	}
	compiledDigest := sha256.Sum256(compiledPayload)
	now := time.Now().UTC()
	claims := builtBinaryModelAttestationClaims{
		SchemaVersion: "skills-hub.model-attestation/v1", Runtime: "generic",
		AgentID: "marketing/creative-operating-system-supervisor", AgentVersion: "0.3.0",
		RuntimeContractSHA256: receipt.RuntimeContractSHA256, CompiledContractSHA256: hex.EncodeToString(compiledDigest[:]),
		ModelIdentity: "test-runtime/model", ModelVersion: "v1", Capabilities: []string{"structured-output"},
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}
	claimsPayload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	attestationPayload, err := json.Marshal(builtBinaryModelAttestation{builtBinaryModelAttestationClaims: claims, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, claimsPayload))})
	if err != nil {
		t.Fatal(err)
	}
	attestationPath := filepath.Join(root, "model-attestation.json")
	if err := os.WriteFile(attestationPath, attestationPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	run := exec.Command(binaryPath, "run-agent", "--agent", claims.AgentID, "--runtime", "generic", "--agents-root", agentsRoot,
		"--skills-root", filepath.Join(root, "runtime", "skills"), "--tools-root", filepath.Join(root, "runtime", "tools-mcp"),
		"--model-attestation", attestationPath, "--approve-live")
	run.Dir = repositoryRoot
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("built binary rejected genuine runtime attestation: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "status: ready") {
		t.Fatalf("built binary did not reach readiness:\n%s", output)
	}
}

type builtBinaryModelAttestationClaims struct {
	SchemaVersion          string    `json:"schema_version"`
	Runtime                string    `json:"runtime"`
	AgentID                string    `json:"agent_id"`
	AgentVersion           string    `json:"agent_version"`
	RuntimeContractSHA256  string    `json:"runtime_contract_sha256"`
	CompiledContractSHA256 string    `json:"compiled_contract_sha256"`
	ModelIdentity          string    `json:"model_identity"`
	ModelVersion           string    `json:"model_version"`
	Capabilities           []string  `json:"capabilities"`
	IssuedAt               time.Time `json:"issued_at"`
	ExpiresAt              time.Time `json:"expires_at"`
}

type builtBinaryModelAttestation struct {
	builtBinaryModelAttestationClaims
	Signature string `json:"signature"`
}

func TestPrintProviderAuthNextStepListsEveryAuthority(t *testing.T) {
	entry := registry.SkillEntry{
		ID: "marketing/performance-reporting-plugin",
		ProviderDependencies: []registry.ProviderDependencyMetadata{
			{Tool: "analytics/ga4-mcp-connector", Requirement: "required", Access: "read-only"},
			{Tool: "warehouse/bigquery-mcp-query-runner", Requirement: "optional", Access: "read-only"},
		},
	}
	var out bytes.Buffer
	printProviderAuthNextStep(&out, entry, "0.2.0", "codex", "/tmp/runtime plugins")
	rendered := out.String()
	for _, want := range []string{
		"analytics/ga4-mcp-connector (required, read-only)",
		"warehouse/bigquery-mcp-query-runner (optional, read-only)",
		"auth status marketing/performance-reporting-plugin@0.2.0 --module plugins --runtime codex",
		`--target "/tmp/runtime plugins"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("provider authentication guidance missing %q:\n%s", want, rendered)
		}
	}
}

func TestShellQuoteCommandPreservesArgumentBoundaries(t *testing.T) {
	got := shellQuoteCommand([]string{
		"skills-hub", "--target", "/Users/me/My Project/tools-mcp", "--state-dir", "/tmp/user's state",
	})
	want := `skills-hub --target '/Users/me/My Project/tools-mcp' --state-dir '/tmp/user'"'"'s state'`
	if got != want {
		t.Fatalf("quoted command = %q, want %q", got, want)
	}
}

func TestBindReadinessEntryUsesReceiptVerifiedSelectedVersionManifest(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	source := filepath.Join(filepath.Dir(filename), "..", "..", "tools-mcp", "analytics", "ga4-mcp-connector")
	targetRoot := filepath.Join(t.TempDir(), "tools-mcp")
	destination := filepath.Join(targetRoot, "analytics", "ga4-mcp-connector")
	copyTestPackage(t, source, destination)

	manifest, err := registry.ValidateHistoricalPackageManifest(filepath.Join(destination, "tool.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	selected := registry.ProjectManifest(manifest)
	contractDigest, err := installer.RuntimeContractSHA256(selected, manifest.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(destination, installer.InstallReceipt{
		Source: "local", Module: "tools", ID: manifest.ID, Version: manifest.Version,
		Runtime: "generic", RuntimeContractSHA256: contractDigest,
	}); err != nil {
		t.Fatal(err)
	}

	latestProjection := selected
	latestProjection.Latest = "0.3.0"
	latestProjection.Authentication = &registry.AuthenticationMetadata{
		Status: "required", Methods: []string{"api-key"}, CredentialBindings: []string{"OTHER_KEY"}, Scopes: []string{"other"},
	}
	bound, err := bindReadinessEntryToInstalledVersion(
		readinessEntry{Module: "tools", Entry: latestProjection, Version: manifest.Version},
		installer.RuntimeTarget{Runtime: "generic", TargetPath: targetRoot},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bound.Version != manifest.Version || len(bound.Entry.Authentication.Methods) != 2 || bound.Entry.Authentication.Methods[0] != "bearer-token" {
		t.Fatalf("selected-version projection = %#v", bound.Entry.Authentication)
	}
}

func TestRunAuthConfigureBindsHistoricalInstalledContract(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	root := t.TempDir()
	source := filepath.Join(filepath.Dir(filename), "..", "..", "tools-mcp", "analytics", "ga4-mcp-connector")
	targetRoot := filepath.Join(root, "tools-mcp")
	destination := filepath.Join(targetRoot, "analytics", "ga4-mcp-connector")
	copyTestPackage(t, source, destination)
	manifestPath := filepath.Join(destination, "tool.yaml")
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(payload), "methods: [bearer-token, service-account]", "methods: [api-key]", 1)
	updated = strings.ReplaceAll(updated, "GA4_CREDENTIAL", "PROVIDER_API_KEY")
	if err := os.WriteFile(manifestPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := registry.ValidatePackageManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	selected := registry.ProjectManifest(manifest)
	contractDigest, err := installer.RuntimeContractSHA256(selected, manifest.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(destination, installer.InstallReceipt{
		Source: "local", Module: "tools", ID: manifest.ID, Version: manifest.Version,
		Runtime: "generic", RuntimeContractSHA256: contractDigest,
	}); err != nil {
		t.Fatal(err)
	}

	latest := selected
	latest.Latest = "0.3.0"
	latest.Versions = []registry.VersionEntry{{Version: "0.3.0"}, {Version: manifest.Version}}
	latest.Authentication = &registry.AuthenticationMetadata{
		Status: "required", Methods: []string{"api-key"}, CredentialBindings: []string{"LATEST_ONLY_KEY"}, Scopes: []string{"latest"},
	}
	registryPath := filepath.Join(root, "tools-index.json")
	if err := registry.WriteIndex(registryPath, registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{latest}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRIVATE_PROVIDER_KEY", "runtime-owned-test-value")
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
	stateDir := filepath.Join(root, "state")
	if err := runAuth([]string{
		"configure", manifest.ID + "@" + manifest.Version,
		"--module", "tools", "--registry", registryPath, "--runtime", "generic", "--target", targetRoot,
		"--state-dir", stateDir, "--method", "api-key",
		"--binding", "PROVIDER_API_KEY=env:PRIVATE_PROVIDER_KEY",
		"--generation", "PROVIDER_API_KEY=env:PRIVATE_PROVIDER_KEY_GENERATION",
		"--validator-command", os.Args[0],
	}); err != nil {
		t.Fatal(err)
	}
	profiles, err := filepath.Glob(filepath.Join(stateDir, "auth", "tools", "analytics", "*.json"))
	if err != nil || len(profiles) != 1 {
		t.Fatalf("historical authentication profiles = %v, %v", profiles, err)
	}
	profile, err := os.ReadFile(profiles[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(profile), `"entry_version": "`+manifest.Version+`"`) || strings.Contains(string(profile), "LATEST_ONLY_KEY") {
		t.Fatalf("historical profile did not use installed contract: %s", profile)
	}
}

func copyTestPackage(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, payload, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrintInstallUsabilityWarning(t *testing.T) {
	tests := []struct {
		availability string
		want         string
	}{
		{availability: "template-only", want: "reference template"},
		{availability: "setup-required", want: "requires configuration"},
		{availability: "not-verified", want: "no current target-scoped operational evidence"},
		{availability: "documentation-only", want: "not executable runtime capability"},
	}
	for _, test := range tests {
		t.Run(test.availability, func(t *testing.T) {
			entry := registry.SkillEntry{ID: "shared/example", Usability: registry.UsabilityMetadata{Availability: test.availability}}
			var out bytes.Buffer
			printInstallUsabilityWarning(&out, entry)
			if !strings.Contains(out.String(), test.want) {
				t.Fatalf("warning %q does not contain %q", out.String(), test.want)
			}
		})
	}
}

func TestPrintInstallLifecycleWarning(t *testing.T) {
	entry := registry.SkillEntry{
		ID:         "shared/deprecated-example",
		Deprecated: true,
		ReplacedBy: "shared/replacement-example",
	}
	var out bytes.Buffer
	printInstallLifecycleWarning(&out, entry)
	for _, want := range []string{"is deprecated", "prefer shared/replacement-example"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("lifecycle warning %q does not contain %q", out.String(), want)
		}
	}
}

func TestPrintUsabilitySummaryIncludesExecutableHelper(t *testing.T) {
	entry := registry.SkillEntry{Usability: registry.UsabilityMetadata{
		Availability: "documentation-only",
		Execution:    "instructions",
		Source:       "declared",
		ExecutableHelpers: []registry.ExecutableHelperMetadata{{
			Entrypoint:   "scripts/check.py",
			Availability: "not-verified",
			Execution:    "local-tool",
			Limitations:  []string{"No current executable evidence."},
			Quickstart:   "python3 scripts/check.py",
		}},
	}}
	var out bytes.Buffer
	printUsabilitySummary(&out, entry)
	for _, want := range []string{
		"usability.executable_helper: scripts/check.py (not-verified, local-tool)",
		"usability.executable_helper.quickstart: python3 scripts/check.py",
		"usability.executable_helper.limitations: No current executable evidence.",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("summary missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunInstallRejectsTemplateBeforeFilesystemWrite(t *testing.T) {
	root := t.TempDir()
	registryPath := filepath.Join(root, "plugins-index.json")
	entry := registry.SkillEntry{
		ID:        "marketing/template-plugin",
		Latest:    "0.1.0",
		Versions:  []registry.VersionEntry{{Version: "0.1.0"}},
		Usability: registry.UsabilityMetadata{Availability: "template-only", Execution: "bundle"},
	}
	if err := registry.WriteIndex(registryPath, registry.Index{RegistryVersion: "1.2", Skills: []registry.SkillEntry{entry}}); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	target := filepath.Join(root, "runtime", "plugins")
	err := runInstall([]string{
		"--module", "plugins",
		"--registry", registryPath,
		"--root", filepath.Join(root, "source", "plugins"),
		"--runtime", "generic",
		"--target", target,
		"--entry", "marketing/template-plugin@0.1.0",
	})
	if err == nil || !strings.Contains(err.Error(), "template-only") {
		t.Fatalf("expected template-only install rejection, got %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("template install created runtime state: %v", statErr)
	}
}

func TestRunInstallKeepsLocalAndRemoteSourcesDistinct(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "local rejects remote registry",
			args: []string{
				"--source", "local",
				"--registry-url", "https://registry.example/index.json",
				"--entry", "engineering/example@1.0.0",
				"--runtime", "generic",
				"--target", filepath.Join(t.TempDir(), "skills"),
			},
			want: "local source does not accept --registry-url",
		},
		{
			name: "remote rejects local root",
			args: []string{
				"--source", "remote",
				"--registry-url", "https://registry.example/index.json",
				"--root", "skills",
				"--entry", "engineering/example@1.0.0",
				"--runtime", "generic",
				"--target", filepath.Join(t.TempDir(), "skills"),
			},
			want: "remote source does not accept --root",
		},
		{
			name: "local rejects execution runtime",
			args: []string{
				"--source", "local",
				"--execution-runtime", "node22",
				"--entry", "engineering/example@1.0.0",
				"--runtime", "generic",
				"--target", filepath.Join(t.TempDir(), "skills"),
			},
			want: "local source does not accept",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := runInstall(test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestRunAuthConfigureAcceptsOnlyRuntimeOwnedReferences(t *testing.T) {
	root := t.TempDir()
	secret := "runtime-only-credential-payload"
	t.Setenv("PRIVATE_PROVIDER_KEY", secret)
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
	registryPath := filepath.Join(root, "tools-index.json")
	entry := registry.SkillEntry{
		ID: "ads/example-tool", Latest: "1.0.0", Versions: []registry.VersionEntry{{Version: "1.0.0"}},
		Authentication: &registry.AuthenticationMetadata{
			Status: "required", Methods: []string{"api-key"}, CredentialBindings: []string{"PROVIDER_API_KEY"}, Scopes: []string{"read"},
		},
	}
	if err := registry.WriteIndex(registryPath, registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{entry}}); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(root, "state")
	if err := runAuth([]string{
		"configure", "ads/example-tool@1.0.0", "--module", "tools", "--registry", registryPath,
		"--state-dir", stateDir, "--method", "api-key",
		"--binding", "PROVIDER_API_KEY=env:PRIVATE_PROVIDER_KEY",
		"--generation", "PROVIDER_API_KEY=env:PRIVATE_PROVIDER_KEY_GENERATION",
		"--validator-command", os.Args[0],
	}); err != nil {
		t.Fatal(err)
	}
	profiles, err := filepath.Glob(filepath.Join(stateDir, "auth", "tools", "ads", "*.json"))
	if err != nil || len(profiles) != 1 {
		t.Fatalf("authentication profile paths = %v, %v", profiles, err)
	}
	payload, err := os.ReadFile(profiles[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), secret) {
		t.Fatal("authentication profile contains credential payload")
	}
}

func TestRunInstallPreflightsPluginClosureBeforeFilesystemWrite(t *testing.T) {
	root := t.TempDir()
	mustCreateDir := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	mustCreateFile := func(path, content string) {
		t.Helper()
		mustCreateDir(filepath.Dir(path))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	mustWriteRegistry := func(path string, entries []registry.SkillEntry) {
		t.Helper()
		mustCreateDir(filepath.Dir(path))
		if err := registry.WriteIndex(path, registry.Index{RegistryVersion: "1.2", Skills: entries}); err != nil {
			t.Fatalf("write registry %s: %v", path, err)
		}
	}

	plugin := registry.SkillEntry{
		ID:       "marketing/demo-plugin",
		Latest:   "1.0.0",
		Versions: []registry.VersionEntry{{Version: "1.0.0"}},
		Includes: &registry.IncludeSet{
			Skills: []string{"marketing/implemented"},
			Agents: []string{"marketing/template"},
		},
	}
	mustCreateFile(filepath.Join(root, "plugins", "marketing", "demo-plugin", "README.md"), "# Demo plugin\n")
	mustCreateFile(filepath.Join(root, "skills", "marketing", "implemented", "SKILL.md"), "# Implemented\n")
	mustCreateFile(filepath.Join(root, "agents", "marketing", "template", "AGENT.md"), "# Template\n")
	mustWriteRegistry(filepath.Join(root, "registry", "plugins-index.json"), []registry.SkillEntry{plugin})
	mustWriteRegistry(filepath.Join(root, "registry", "skills-index.json"), []registry.SkillEntry{{ID: "marketing/implemented"}})
	mustWriteRegistry(filepath.Join(root, "registry", "agents-index.json"), []registry.SkillEntry{{
		ID:        "marketing/template",
		Usability: registry.UsabilityMetadata{Availability: "template-only"},
	}})

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	err = runInstall([]string{
		"--module", "plugins",
		"--registry", filepath.Join("registry", "plugins-index.json"),
		"--root", "plugins",
		"--runtime", "generic",
		"--target", filepath.Join("runtime", "plugins"),
		"--entry", "marketing/demo-plugin@1.0.0",
	})
	if err == nil || !strings.Contains(err.Error(), "template-only") {
		t.Fatalf("expected dependency preflight rejection, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "runtime")); !os.IsNotExist(statErr) {
		t.Fatalf("plugin preflight left partial runtime state: %v", statErr)
	}
}

func TestPrintPluginInstallNotesWarnsWhenNotSecurityReviewed(t *testing.T) {
	entry := registry.SkillEntry{
		ID:               "marketing/performance-reporting-plugin",
		SecurityReviewed: false,
		Includes:         &registry.IncludeSet{},
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	printPluginInstallNotes(
		&out,
		&errOut,
		entry,
		"codex",
		"/tmp/plugins/marketing/performance-reporting-plugin",
		[]string{"/tmp/plugins/marketing/performance-reporting-plugin/.codex-plugin/plugin.json"},
		installer.DependencyInstallResult{
			InstalledSkills: []string{"/tmp/codex/skills/marketing/meta-google-weekly-performance-review"},
			HookPaths:       []string{"/tmp/plugins/marketing/performance-reporting-plugin/hooks/post-analysis-slack-summary.md"},
		},
	)

	if !strings.Contains(errOut.String(), "is not security reviewed") {
		t.Fatalf("expected security review warning, got: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "includes.skills: 0") {
		t.Fatalf("expected plugin summary in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "install.root: /tmp/plugins/marketing/performance-reporting-plugin") {
		t.Fatalf("expected install root in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "install.runtime_artifacts: /tmp/plugins/marketing/performance-reporting-plugin/.codex-plugin/plugin.json") {
		t.Fatalf("expected runtime artifact path in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "install.next_step: review the generated codex runtime manifest before enabling the plugin") {
		t.Fatalf("expected next step guidance in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "deps.skills.installed: 1") {
		t.Fatalf("expected dependency summary in stdout, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "deps.hooks.packaged.list: /tmp/plugins/marketing/performance-reporting-plugin/hooks/post-analysis-slack-summary.md") {
		t.Fatalf("expected packaged hook path in stdout, got: %s", out.String())
	}
}
