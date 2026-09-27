package readiness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestCompositeStatusDerivesRequiredAndOptionalCoverage(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	fixture := installCompositeStatusFixture(t, "read-only")

	report, err := CompositeStatus(context.Background(), fixture.options(now))
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.CoverageState != "unavailable" || len(report.Providers) != 2 {
		t.Fatalf("unconfigured composite status = %#v", report)
	}

	configureCompositeProvider(t, fixture.stateDir, fixture.toolEntries[0], now)
	report, err = CompositeStatus(context.Background(), fixture.options(now))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready || report.CoverageState != "partial" || !report.SetupRequired {
		t.Fatalf("required-only composite status = %#v", report)
	}
	if report.Providers[0].State != "ready" || report.Providers[1].State != "unavailable" {
		t.Fatalf("provider states = %#v", report.Providers)
	}
	if len(report.Providers[1].ConfigureCommands) != 1 {
		t.Fatalf("optional provider setup commands = %#v", report.Providers[1].ConfigureCommands)
	}
	if command := strings.Join(report.Providers[1].ConfigureCommands[0], " "); !strings.Contains(command, "auth configure") || !strings.Contains(command, "env:PROVIDER_API_KEY") || !strings.Contains(command, "--state-dir "+fixture.stateDir) {
		t.Fatalf("optional provider setup command = %q", command)
	}

	configureCompositeProvider(t, fixture.stateDir, fixture.toolEntries[1], now)
	report, err = CompositeStatus(context.Background(), fixture.options(now))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready || report.CoverageState != "complete" || report.SetupRequired {
		t.Fatalf("complete composite status = %#v", report)
	}
}

func TestCompositeConfigureCommandsRequireAnExplicitMethod(t *testing.T) {
	auth := registry.AuthenticationMetadata{
		Methods: []string{"bearer-token", "service-account"}, CredentialBindings: []string{"GA4_CREDENTIAL"},
	}
	commands := compositeConfigureCommands(
		"analytics/ga4-mcp-connector", "0.2.0", "generic", "/tmp/My Project/tools-mcp", "/tmp/custom state", auth,
	)
	if len(commands) != 2 {
		t.Fatalf("configure commands = %#v", commands)
	}
	for index, method := range auth.Methods {
		command := strings.Join(commands[index], " ")
		if !strings.Contains(command, "--method "+method) || !strings.Contains(command, "--state-dir /tmp/custom state") {
			t.Fatalf("configure command for %s = %q", method, command)
		}
	}
}

func TestCompositeStatusRejectsInstalledProviderAuthorityMismatch(t *testing.T) {
	fixture := installCompositeStatusFixture(t, "read-write")
	report, err := CompositeStatus(context.Background(), fixture.options(time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.CoverageState != "unavailable" || !strings.Contains(report.Providers[0].Reason, "authority") {
		t.Fatalf("authority mismatch status = %#v", report)
	}
}

func TestCompositeStatusRejectsExpiredInstalledProviderEvidence(t *testing.T) {
	fixture := installCompositeStatusFixtureWithVerification(t, "read-only", "2020-01-01T00:00:00Z")
	report, err := CompositeStatus(context.Background(), fixture.options(time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.CoverageState != "unavailable" || !strings.Contains(report.Providers[0].Reason, "expired") {
		t.Fatalf("expired provider evidence status = %#v", report)
	}
}

type compositeStatusFixture struct {
	stateDir    string
	pluginEntry registry.SkillEntry
	pluginDir   string
	pluginRoot  string
	toolEntries []registry.SkillEntry
}

func (fixture compositeStatusFixture) options(now time.Time) CompositeStatusOptions {
	return CompositeStatusOptions{
		StateDir: fixture.stateDir, Entry: fixture.pluginEntry, Version: "1.0.0",
		PackageDir: fixture.pluginDir, TargetRoot: fixture.pluginRoot, Runtime: "generic", Now: now,
	}
}

func installCompositeStatusFixture(t *testing.T, requiredAccess string) compositeStatusFixture {
	return installCompositeStatusFixtureWithVerification(t, requiredAccess, "2026-09-17T00:00:00Z")
}

func installCompositeStatusFixtureWithVerification(t *testing.T, requiredAccess, lastVerifiedAt string) compositeStatusFixture {
	t.Helper()
	root := t.TempDir()
	pluginRoot := filepath.Join(root, "plugins")
	toolRoot := filepath.Join(root, "tools-mcp")
	pluginID := "marketing/composite-status-plugin"
	toolIDs := []string{"analytics/required-provider", "warehouse/optional-provider"}
	dependencies := []registry.ProviderDependencyMetadata{
		{Tool: toolIDs[0], Requirement: "required", Access: requiredAccess},
		{Tool: toolIDs[1], Requirement: "optional", Access: "read-only"},
	}
	pluginEntry := registry.SkillEntry{
		ID: pluginID, Latest: "1.0.0", Runtimes: []string{"generic"},
		Execution: &registry.ExecutionMetadata{Kind: "bundle", SupportedPlatforms: []string{currentPlatform()}, SupportedRuntimes: []string{"native"}},
		Artifact:  &registry.ArtifactMetadata{SelfContained: true},
		Authentication: &registry.AuthenticationMetadata{
			Status: "none", Methods: []string{"none"}, CredentialStorage: "Provider tools own credentials.",
			Validation: "Provider tools validate identity and scopes.", Revocation: "Revoke credentials with each provider.",
		},
		Includes: &registry.IncludeSet{Tools: append([]string(nil), toolIDs...)}, ProviderDependencies: dependencies,
	}
	pluginDir := filepath.Join(pluginRoot, filepath.FromSlash(pluginID))
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	toolEntries := make([]registry.SkillEntry, 0, len(toolIDs))
	closure := make([]installer.InstallReceiptMember, 0, len(toolIDs))
	for _, toolID := range toolIDs {
		toolDir := filepath.Join(toolRoot, filepath.FromSlash(toolID))
		materializeCompositeToolPaths(t, toolDir)
		manifest := compositeToolManifest(t, toolID, lastVerifiedAt)
		if err := os.WriteFile(filepath.Join(toolDir, "tool.yaml"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
		parsed, err := registry.ValidateHistoricalPackageManifest(filepath.Join(toolDir, "tool.yaml"))
		if err != nil {
			t.Fatalf("validate fixture tool %s: %v", toolID, err)
		}
		entry := registry.ProjectManifest(parsed)
		toolEntries = append(toolEntries, entry)
		contractDigest, err := installer.RuntimeContractSHA256(entry, "0.2.0")
		if err != nil {
			t.Fatal(err)
		}
		if err := installer.WriteInstallReceipt(toolDir, installer.InstallReceipt{
			Source: "local", Module: "tools", ID: toolID, Version: "0.2.0", Runtime: "generic", RuntimeContractSHA256: contractDigest,
		}); err != nil {
			t.Fatal(err)
		}
		receipt, err := installer.VerifyInstallReceipt(toolDir, "tools", toolID, "0.2.0", "generic", "", contractDigest)
		if err != nil {
			t.Fatal(err)
		}
		closure = append(closure, installer.InstallReceiptMember{
			Module: "tools", ID: toolID, Version: "0.2.0", RuntimeContractSHA256: contractDigest, TreeSHA256: receipt.TreeSHA256,
		})
	}
	pluginContract, err := installer.RuntimeContractSHA256(pluginEntry, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(pluginDir, installer.InstallReceipt{
		Source: "local", Module: "plugins", ID: pluginID, Version: "1.0.0", Runtime: "generic",
		RuntimeContractSHA256: pluginContract, Closure: closure,
	}); err != nil {
		t.Fatal(err)
	}
	return compositeStatusFixture{
		stateDir: filepath.Join(root, "state"), pluginEntry: pluginEntry, pluginDir: pluginDir,
		pluginRoot: pluginRoot, toolEntries: toolEntries,
	}
}

func materializeCompositeToolPaths(t *testing.T, toolDir string) {
	t.Helper()
	for _, directory := range []string{"examples", "scripts", "tests"} {
		if err := os.MkdirAll(filepath.Join(toolDir, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{
		"TOOL.md", "README.md", "tests/ga4_mcp_server.test.mjs", "scripts/ga4_mcp_server.mjs",
	} {
		if err := os.WriteFile(filepath.Join(toolDir, filepath.FromSlash(path)), []byte("fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func compositeToolManifest(t *testing.T, toolID, lastVerifiedAt string) []byte {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve fixture source")
	}
	path := filepath.Join(filepath.Dir(filename), "..", "..", "tools-mcp", "analytics", "ga4-mcp-connector", "tool.yaml")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value := strings.ReplaceAll(string(payload), "analytics/ga4-mcp-connector", toolID)
	value = strings.Replace(value, "methods: [bearer-token, service-account]", "methods: [api-key]", 1)
	value = strings.ReplaceAll(value, "GA4_CREDENTIAL", "PROVIDER_API_KEY")
	value = strings.Replace(value, `last_verified_at: "2026-09-17T00:00:00Z"`, `last_verified_at: "`+lastVerifiedAt+`"`, 1)
	return []byte(value)
}

func configureCompositeProvider(t *testing.T, stateDir string, entry registry.SkillEntry, now time.Time) {
	t.Helper()
	payload, err := json.Marshal(observation(true, false, "acct-1", entry.Authentication.Scopes, now.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Configure(ConfigureOptions{
		StateDir: stateDir, Module: "tools", Entry: entry, Version: "0.2.0", Method: "api-key", ExpectedAccount: "acct-1",
		Bindings:    map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY"},
		Generations: map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY_GENERATION"},
		Validator:   helperValidator(string(payload), false), Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
}
