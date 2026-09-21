package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeTargetExplicitTargetWins(t *testing.T) {
	target, err := ResolveRuntimeTarget("codex", "./tmp-target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(target.TargetPath) {
		t.Fatalf("expected absolute target path, got: %s", target.TargetPath)
	}
	if target.Runtime != "codex" {
		t.Fatalf("unexpected runtime: %s", target.Runtime)
	}
}

func TestGA4ReleaseArchiveLaunchesAsNativeMCPServer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the GA4 MCP launch regression: %v", err)
	}
	destination := t.TempDir()
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "analytics", "ga4-mcp-connector", "0.2.0", "package.tar.gz")
	if err := extractVerifiedTarGz(archive, destination); err != nil {
		t.Fatalf("extract clean GA4 release: %v", err)
	}
	health := exec.Command(node, "scripts/ga4_mcp_server.mjs", "--healthcheck")
	health.Dir = destination
	output, err := health.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"status":"ok"`)) {
		t.Fatalf("GA4 release healthcheck failed: %v\n%s", err, output)
	}
	command := exec.Command(node, "scripts/ga4_mcp_server.mjs")
	command.Dir = destination
	command.Stdin = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	output, err = command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"name":"ga4-mcp-connector"`)) || !bytes.Contains(output, []byte(`"name":"ga4_run_report"`)) {
		t.Fatalf("GA4 release MCP handshake failed: %v\n%s", err, output)
	}
}

func TestBigQueryReleaseArchiveLaunchesAsNativeMCPServer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the BigQuery MCP launch regression: %v", err)
	}
	destination := t.TempDir()
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "warehouse", "bigquery-mcp-query-runner", "0.2.0", "package.tar.gz")
	if err := extractVerifiedTarGz(archive, destination); err != nil {
		t.Fatalf("extract clean BigQuery release: %v", err)
	}
	health := exec.Command(node, "scripts/bigquery_mcp_server.mjs", "--healthcheck")
	health.Dir = destination
	output, err := health.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"status":"ok"`)) {
		t.Fatalf("BigQuery release healthcheck failed: %v\n%s", err, output)
	}
	command := exec.Command(node, "scripts/bigquery_mcp_server.mjs")
	command.Dir = destination
	command.Stdin = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	output, err = command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"name":"bigquery-mcp-query-runner"`)) || !bytes.Contains(output, []byte(`"name":"bigquery_run_query"`)) {
		t.Fatalf("BigQuery release MCP handshake failed: %v\n%s", err, output)
	}
}

func TestMetaAdsReleaseArchiveLaunchesAsNativeMCPServer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the Meta Ads MCP launch regression: %v", err)
	}
	destination := t.TempDir()
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "ads", "meta-ads-mcp-connector", "0.2.0", "package.tar.gz")
	if err := extractVerifiedTarGz(archive, destination); err != nil {
		t.Fatalf("extract clean Meta Ads release: %v", err)
	}
	health := exec.Command(node, "scripts/meta_ads_mcp_server.mjs", "--healthcheck")
	health.Dir = destination
	output, err := health.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"status":"ok"`)) {
		t.Fatalf("Meta Ads release healthcheck failed: %v\n%s", err, output)
	}
	command := exec.Command(node, "scripts/meta_ads_mcp_server.mjs")
	command.Dir = destination
	command.Stdin = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	output, err = command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"name":"meta-ads-mcp-connector"`)) || !bytes.Contains(output, []byte(`"name":"meta_ads_read_insights"`)) {
		t.Fatalf("Meta Ads release MCP handshake failed: %v\n%s", err, output)
	}
}

func TestAgentControlPlaneReleaseArchiveLaunchesAsNativeMCPServer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the Agent Control Plane MCP launch regression: %v", err)
	}
	destination := t.TempDir()
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "agentops", "agent-control-plane-server", "0.2.1", "package.tar.gz")
	if err := extractVerifiedTarGz(archive, destination); err != nil {
		t.Fatalf("extract clean Agent Control Plane release: %v", err)
	}
	health := exec.Command(node, "scripts/agent_control_plane_server.mjs", "--healthcheck")
	health.Dir = destination
	output, err := health.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"status":"ok"`)) {
		t.Fatalf("Agent Control Plane release healthcheck failed: %v\n%s", err, output)
	}
	command := exec.Command(node, "scripts/agent_control_plane_server.mjs")
	command.Dir = destination
	command.Stdin = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	output, err = command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"name":"agent-control-plane-server"`)) || !bytes.Contains(output, []byte(`"name":"authorize_action"`)) {
		t.Fatalf("Agent Control Plane release MCP handshake failed: %v\n%s", err, output)
	}
}

func TestAdPlatformExecutorReleaseArchiveLaunchesAsNativeMCPServer(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the governed executor launch regression: %v", err)
	}
	destination := t.TempDir()
	archive := filepath.Join("..", "..", "releases", "tools-mcp", "adtech", "ad-platform-executor-template", "0.2.0", "package.tar.gz")
	if err := extractVerifiedTarGz(archive, destination); err != nil {
		t.Fatalf("extract clean governed executor release: %v", err)
	}
	health := exec.Command(node, "scripts/ad_platform_executor_server.mjs", "--healthcheck")
	health.Dir = destination
	output, err := health.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"status":"ok"`)) {
		t.Fatalf("governed executor release healthcheck failed: %v\n%s", err, output)
	}
	command := exec.Command(node, "scripts/ad_platform_executor_server.mjs")
	command.Dir = destination
	command.Stdin = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	output, err = command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"name":"governed-ad-platform-executor"`)) || !bytes.Contains(output, []byte(`"name":"execute_approved_change"`)) {
		t.Fatalf("governed executor release MCP handshake failed: %v\n%s", err, output)
	}
}

func TestResolveRuntimeTargetCodexFromEnv(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")
	target, err := ResolveRuntimeTarget("codex", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/codex-home", "skills")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetClaudeFromEnv(t *testing.T) {
	t.Setenv("CLAUDE_HOME", "/tmp/claude-home")
	target, err := ResolveRuntimeTarget("claude", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/claude-home", "skills")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetGenericNeedsTarget(t *testing.T) {
	_, err := ResolveRuntimeTarget("generic", "")
	if err == nil {
		t.Fatalf("expected error for missing target")
	}
}

func TestResolveRuntimeTargetForModuleAgents(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")
	target, err := ResolveRuntimeTargetForModule("codex", "agents", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/codex-home", "agents")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetForModuleToolsMcp(t *testing.T) {
	t.Setenv("CLAUDE_HOME", "/tmp/claude-home")
	target, err := ResolveRuntimeTargetForModule("claude", "tools", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/claude-home", "tools-mcp")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestResolveRuntimeTargetForModulePlugins(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/codex-home")
	target, err := ResolveRuntimeTargetForModule("codex", "plugins", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/codex-home", "plugins")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}

func TestPreparePluginRuntimeArtifactsForCodex(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)

	artifacts, err := PreparePluginRuntimeArtifacts(dir, "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected runtime contract and native manifest, got %d", len(artifacts))
	}
	if artifacts[1] != filepath.Join(dir, "plugin.json") {
		t.Fatalf("unexpected artifact path: %s", artifacts[1])
	}
	assertGoldenRuntimeArtifact(t, filepath.Join(dir, ".runtime", "codex.json"), "codex.runtime.json")
	assertGoldenRuntimeArtifact(t, artifacts[1], "codex.plugin.json")
	if _, err := os.Stat(filepath.Join(dir, "skills", "creative-workshop-pmax-reels", "SKILL.md")); err != nil {
		t.Fatalf("compiled skill is not discoverable: %v", err)
	}
}

func TestPreparePluginRuntimeArtifactsForClaude(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)

	artifacts, err := PreparePluginRuntimeArtifacts(dir, "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected runtime contract and native manifest, got %d", len(artifacts))
	}
	if !strings.HasSuffix(artifacts[1], filepath.Join(".claude-plugin", "plugin.json")) {
		t.Fatalf("unexpected artifact path: %s", artifacts[1])
	}
	assertGoldenRuntimeArtifact(t, filepath.Join(dir, ".runtime", "claude.json"), "claude.runtime.json")
	assertGoldenRuntimeArtifact(t, artifacts[1], "claude.plugin.json")
}

func TestClaudeCompiledPackagePassesNativeValidatorWhenAvailable(t *testing.T) {
	validator, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("Claude Code is not installed")
	}
	dir := extractRuntimeCompilerFixture(t)
	if _, err := PreparePluginRuntimeArtifacts(dir, "claude"); err != nil {
		t.Fatalf("compile Claude package: %v", err)
	}
	command := exec.Command(validator, "plugin", "validate", dir)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Claude plugin validation failed: %v\n%s", err, string(output))
	}
}

func TestClaudeNativeManifestAcceptsNestedMCPConfigWhenValidatorAvailable(t *testing.T) {
	validator, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("Claude Code is not installed")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server", ".mcp.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("{\"mcpServers\":{\"demo\":{\"command\":\"node\",\"args\":[\"server.js\"]}}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeDeterministicJSON(filepath.Join(manifestDir, "plugin.json"), nativePluginManifest{
		Name: "demo-plugin", Version: "1.0.0", Description: "Demo plugin.",
		MCPServers: "./bundled/tools-mcp/engineering/demo-server/.mcp.json",
	}); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(validator, "plugin", "validate", dir)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Claude nested MCP registration validation failed: %v\n%s", err, string(output))
	}
}

func TestPreparePluginRuntimeArtifactsForGeneric(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)

	artifacts, err := PreparePluginRuntimeArtifacts(dir, "generic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one generic runtime contract, got %d", len(artifacts))
	}
	assertGoldenRuntimeArtifact(t, artifacts[0], "generic.runtime.json")
}

func TestPreparePluginRuntimeArtifactsRejectsUndeclaredDiscoveryRoots(t *testing.T) {
	for _, relative := range []string{
		"skills/rogue/SKILL.md",
		"agents/rogue.md",
		"commands/rogue.md",
		"mcp.json",
		".mcp.json",
		"hooks/hooks.json",
	} {
		t.Run(strings.ReplaceAll(relative, "/", "_"), func(t *testing.T) {
			dir := extractRuntimeCompilerFixture(t)
			path := filepath.Join(dir, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := PreparePluginRuntimeArtifacts(dir, "codex"); err == nil || !strings.Contains(err.Error(), "runtime discovery input") {
				t.Fatalf("expected undeclared discovery rejection for %s, got %v", relative, err)
			}
		})
	}
}

func TestPreparePluginRuntimeArtifactsRejectsMinimalManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte("{\"name\":\"demo-plugin\",\"version\":\"0.1.0\",\"description\":\"demo\"}\n"), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
	if _, err := PreparePluginRuntimeArtifacts(dir, "codex"); err == nil || !strings.Contains(err.Error(), "plugin.yaml") {
		t.Fatalf("expected incomplete source manifest rejection, got %v", err)
	}
}

func TestPreparePluginRuntimeArtifactsKeepsRepositorySourceInstallExplicitlyLegacy(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join("..", "..", "plugins", "marketing", "content-repurposing-plugin")
	if err := copyTree(source, dir); err != nil {
		t.Fatalf("copy source plugin fixture: %v", err)
	}
	artifacts, err := PreparePluginRuntimeArtifacts(dir, "codex")
	if err != nil {
		t.Fatalf("prepare source development install: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one legacy compatibility artifact, got %d", len(artifacts))
	}
	assertFileContains(t, artifacts[0], `"compatibility": "legacy-uncompiled"`)
	if _, err := os.Stat(filepath.Join(dir, ".runtime", "codex.json")); !os.IsNotExist(err) {
		t.Fatalf("source development install was represented as a compiled release")
	}
}

func TestValidateCompiledRuntimePackageRejectsOmittedLockedComponent(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)
	if _, err := PreparePluginRuntimeArtifacts(dir, "generic"); err != nil {
		t.Fatalf("compile fixture: %v", err)
	}
	contractPath := filepath.Join(dir, ".runtime", "generic.json")
	payload, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var contract compiledRuntimePackage
	if err := json.Unmarshal(payload, &contract); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	contract.Registrations.Skills = contract.Registrations.Skills[1:]
	if err := writeDeterministicJSON(contractPath, contract); err != nil {
		t.Fatalf("mutate contract: %v", err)
	}
	if err := ValidateCompiledRuntimePackage(dir, "generic"); err == nil || !strings.Contains(err.Error(), "component count") {
		t.Fatalf("expected component coverage rejection, got %v", err)
	}
}

func TestValidateCompiledRuntimePackageRejectsUnsafeSourceLock(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)
	if _, err := PreparePluginRuntimeArtifacts(dir, "generic"); err != nil {
		t.Fatalf("compile fixture: %v", err)
	}
	contractPath := filepath.Join(dir, ".runtime", "generic.json")
	contract := readCompiledRuntimeContract(t, contractPath)
	contract.SourceLock = "../dependencies.lock.json"
	if err := writeDeterministicJSON(contractPath, contract); err != nil {
		t.Fatalf("mutate contract: %v", err)
	}
	if err := ValidateCompiledRuntimePackage(dir, "generic"); err == nil || !strings.Contains(err.Error(), "source lock") {
		t.Fatalf("expected source lock rejection, got %v", err)
	}
}

func TestValidateCompiledRuntimePackageRejectsWrongCategoryOrVersion(t *testing.T) {
	for _, mutation := range []struct {
		name   string
		mutate func(*compiledRuntimePackage)
	}{
		{
			name: "category",
			mutate: func(contract *compiledRuntimePackage) {
				registration := contract.Registrations.Skills[0]
				contract.Registrations.Skills = contract.Registrations.Skills[1:]
				contract.Registrations.Agents = append(contract.Registrations.Agents, registration)
			},
		},
		{
			name: "version",
			mutate: func(contract *compiledRuntimePackage) {
				contract.Registrations.Skills[0].Version = "99.0.0"
			},
		},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			dir := extractRuntimeCompilerFixture(t)
			if _, err := PreparePluginRuntimeArtifacts(dir, "generic"); err != nil {
				t.Fatalf("compile fixture: %v", err)
			}
			contractPath := filepath.Join(dir, ".runtime", "generic.json")
			contract := readCompiledRuntimeContract(t, contractPath)
			mutation.mutate(&contract)
			if err := writeDeterministicJSON(contractPath, contract); err != nil {
				t.Fatalf("mutate contract: %v", err)
			}
			if err := ValidateCompiledRuntimePackage(dir, "generic"); err == nil {
				t.Fatal("expected mutated lock registration to be rejected")
			}
		})
	}
}

func TestValidateCompiledRuntimePackageRejectsPostCompileDiscoveryExpansion(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)
	if _, err := PreparePluginRuntimeArtifacts(dir, "codex"); err != nil {
		t.Fatalf("compile fixture: %v", err)
	}
	rogue := filepath.Join(dir, "skills", "rogue", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(rogue), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rogue, []byte("---\nname: rogue\ndescription: rogue\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCompiledRuntimePackage(dir, "codex"); err == nil || !strings.Contains(err.Error(), "runtime discovery layout") {
		t.Fatalf("expected expanded discovery layout rejection, got %v", err)
	}
}

func TestValidateCompiledRuntimePackageRejectsNativeManifestDiscoveryExpansion(t *testing.T) {
	dir := extractRuntimeCompilerFixture(t)
	if _, err := PreparePluginRuntimeArtifacts(dir, "claude"); err != nil {
		t.Fatalf("compile fixture: %v", err)
	}
	rogueConfig := filepath.Join(dir, "rogue-mcp.json")
	if err := os.WriteFile(rogueConfig, []byte(`{"mcpServers":{"rogue":{"command":"node"}}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, ".claude-plugin", "plugin.json")
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["mcpServers"] = "./rogue-mcp.json"
	if err := writeDeterministicJSON(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCompiledRuntimePackage(dir, "claude"); err == nil || !strings.Contains(err.Error(), "discovery does not match") {
		t.Fatalf("expected native manifest expansion rejection, got %v", err)
	}
}

func TestMaterializeRuntimeAgentUsesNativeClaudeLayout(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join("..", "..", "agents", "marketing", "weekly-performance-supervisor")
	target := filepath.Join(dir, "bundled", "agents", "marketing", "weekly-performance-supervisor")
	if err := copyTree(source, target); err != nil {
		t.Fatalf("copy agent fixture: %v", err)
	}
	component := runtimeDependencyLockComponent{
		Module: "agents", ID: "marketing/weekly-performance-supervisor", Version: "0.1.0",
		ManifestPath: "bundled/agents/marketing/weekly-performance-supervisor/agent.yaml",
	}
	registration, err := materializeRuntimeComponent(dir, "claude", component)
	if err != nil {
		t.Fatalf("compile Claude agent: %v", err)
	}
	if registration.Enforcement != "native" || registration.Path != "./agents/weekly-performance-supervisor.md" {
		t.Fatalf("unexpected Claude agent registration: %#v", registration)
	}
	assertFileContains(t, filepath.Join(dir, "agents", "weekly-performance-supervisor.md"), "name: weekly-performance-supervisor")
	assertFileContains(t, filepath.Join(dir, "agents", "weekly-performance-supervisor.md"), "# Weekly Performance Supervisor")
}

func TestMaterializeRuntimeAgentMakesUnsupportedRuntimeVisible(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join("..", "..", "agents", "marketing", "weekly-performance-supervisor")
	target := filepath.Join(dir, "bundled", "agents", "marketing", "weekly-performance-supervisor")
	if err := copyTree(source, target); err != nil {
		t.Fatalf("copy agent fixture: %v", err)
	}
	registration, err := materializeRuntimeComponent(dir, "codex", runtimeDependencyLockComponent{
		Module: "agents", ID: "marketing/weekly-performance-supervisor", Version: "0.1.0",
		ManifestPath: "bundled/agents/marketing/weekly-performance-supervisor/agent.yaml",
	})
	if err != nil {
		t.Fatalf("compile Codex agent: %v", err)
	}
	if registration.Enforcement != "advisory" || registration.Reason == "" {
		t.Fatalf("unsupported Codex agent was not disclosed: %#v", registration)
	}
}

func TestMaterializeRuntimeToolRegistersPackagedMCPConfig(t *testing.T) {
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte("{\"mcpServers\":{\"demo\":{\"command\":\"node\",\"args\":[\"server.js\"]}}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registration, err := materializeRuntimeComponent(dir, "claude", runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	})
	if err != nil {
		t.Fatalf("compile MCP server: %v", err)
	}
	if registration.Enforcement != "native" || !strings.HasSuffix(registration.Path, "/.mcp.json") {
		t.Fatalf("packaged MCP config was not registered: %#v", registration)
	}
}

func TestMaterializeRuntimeToolRejectsEmptyMCPConfig(t *testing.T) {
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte("{\"mcpServers\":{}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := materializeRuntimeComponent(dir, "claude", runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	})
	if err == nil || !strings.Contains(err.Error(), "declares no MCP servers") {
		t.Fatalf("expected empty MCP config rejection, got %v", err)
	}
}

func TestCompileCodexMCPConfigProducesPortableRootDocument(t *testing.T) {
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"mcpServers":{"demo":{"type":"stdio","command":"node","args":["server.js"]}}}` + "\n"
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	component := runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	}
	registration, err := materializeRuntimeComponent(dir, "codex", component)
	if err != nil {
		t.Fatalf("classify Codex MCP component: %v", err)
	}
	if registration.Enforcement != "native" || registration.Path != "./mcp.json" {
		t.Fatalf("unexpected Codex MCP registration: %#v", registration)
	}
	if len(registration.Names) != 1 || registration.Names[0] != "demo" {
		t.Fatalf("Codex MCP server inventory is incomplete: %#v", registration.Names)
	}
	if err := compileCodexMCPConfig(dir, []runtimeDependencyLockComponent{component}, []compiledRuntimeRegistration{registration}); err != nil {
		t.Fatalf("compile Codex MCP config: %v", err)
	}
	assertFileContains(t, filepath.Join(dir, "mcp.json"), agentPluginMCPConfigSchema)
	assertFileContains(t, filepath.Join(dir, "mcp.json"), `"type": "stdio"`)
	assertGoldenRuntimeArtifact(t, filepath.Join(dir, "mcp.json"), "codex.mcp.json")
}

func TestCompileCodexMCPConfigExecutesFromBundledComponentRoot(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the MCP execution regression: %v", err)
	}
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolRoot, "server.js"), []byte("process.stdout.write('component-root')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"mcpServers":{"demo":{"type":"stdio","command":"node","args":["server.js"]}}}` + "\n"
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	component := runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	}
	registration, err := materializeRuntimeComponent(dir, "codex", component)
	if err != nil {
		t.Fatal(err)
	}
	if err := compileCodexMCPConfig(dir, []runtimeDependencyLockComponent{component}, []compiledRuntimeRegistration{registration}); err != nil {
		t.Fatal(err)
	}
	document, err := loadMCPServerConfig(filepath.Join(dir, "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var server portableMCPStdioServer
	if err := json.Unmarshal(document.MCPServers["demo"], &server); err != nil {
		t.Fatal(err)
	}
	if server.CWD != "./bundled/tools-mcp/engineering/demo-server" {
		t.Fatalf("component-local cwd was not rebased: %q", server.CWD)
	}
	command := exec.Command(node, server.Args...)
	command.Dir = filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(server.CWD, "./")))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled MCP command failed: %v\n%s", err, output)
	}
	if string(output) != "component-root" {
		t.Fatalf("compiled MCP command ran from the wrong root: %q", output)
	}
}

func TestCompileCodexMCPConfigRebasesExplicitComponentPaths(t *testing.T) {
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"mcpServers":{"demo":{"type":"stdio","command":"./bin/server","args":["./relative.json","${PLUGIN_ROOT}/config.json"],"env":{"CONFIG":"${PLUGIN_ROOT}/config.json"},"cwd":"./work"}}}` + "\n"
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	component := runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	}
	registration, err := materializeRuntimeComponent(dir, "codex", component)
	if err != nil {
		t.Fatal(err)
	}
	if err := compileCodexMCPConfig(dir, []runtimeDependencyLockComponent{component}, []compiledRuntimeRegistration{registration}); err != nil {
		t.Fatal(err)
	}
	document, err := loadMCPServerConfig(filepath.Join(dir, "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var server portableMCPStdioServer
	if err := json.Unmarshal(document.MCPServers["demo"], &server); err != nil {
		t.Fatal(err)
	}
	componentPath := "bundled/tools-mcp/engineering/demo-server"
	if server.Command != "./"+componentPath+"/bin/server" || server.CWD != "./"+componentPath+"/work" {
		t.Fatalf("component paths were not rebased: %#v", server)
	}
	expectedReference := "${PLUGIN_ROOT}/" + componentPath + "/config.json"
	if server.Args[0] != "./relative.json" || server.Args[1] != expectedReference || server.Env["CONFIG"] != expectedReference {
		t.Fatalf("PLUGIN_ROOT references were not rebased: %#v", server)
	}
}

func TestMaterializeCodexMCPRequiresPortableTransportType(t *testing.T) {
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte(`{"mcpServers":{"demo":{"command":"node"}}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := materializeRuntimeComponent(dir, "codex", runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	})
	if err == nil || !strings.Contains(err.Error(), "transport type") {
		t.Fatalf("expected portable transport rejection, got %v", err)
	}
}

func TestValidatePortableMCPRemoteEndpoints(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		url     string
		headers map[string]string
	}{
		{name: "https", url: "https://mcp.example.com/rpc", headers: map[string]string{"X-Tenant": "public"}},
		{name: "localhost-http", url: "http://localhost:8080/rpc"},
		{name: "ipv4-loopback-http", url: "http://127.0.0.1:8080/rpc"},
		{name: "ipv6-loopback-http", url: "http://[::1]:8080/rpc"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validatePortableMCPRemoteEndpoint(testCase.url, testCase.headers); err != nil {
				t.Fatalf("expected valid remote MCP endpoint, got %v", err)
			}
		})
	}
}

func TestValidatePortableMCPRemoteEndpointsRejectsInvalidTransportClaims(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		url     string
		headers map[string]string
	}{
		{name: "non-loopback-http", url: "http://mcp.example.com/rpc"},
		{name: "relative-url", url: "/rpc"},
		{name: "unsupported-scheme", url: "ftp://mcp.example.com/rpc"},
		{name: "credentials", url: "https://user:secret@mcp.example.com/rpc"},
		{name: "fragment", url: "https://mcp.example.com/rpc#token"},
		{name: "malformed-url", url: "https://[::1"},
		{name: "invalid-header-name", url: "https://mcp.example.com/rpc", headers: map[string]string{"Bad Header": "value"}},
		{name: "invalid-header-value", url: "https://mcp.example.com/rpc", headers: map[string]string{"X-Test": "value\r\ninjected: true"}},
		{name: "case-insensitive-duplicate", url: "https://mcp.example.com/rpc", headers: map[string]string{"X-Tenant": "one", "x-tenant": "two"}},
		{name: "authorization-header", url: "https://mcp.example.com/rpc", headers: map[string]string{"Authorization": "Bearer visible-secret-value"}},
		{name: "api-key-header", url: "https://mcp.example.com/rpc", headers: map[string]string{"X-API-Key": "visible-secret-value"}},
		{name: "provider-token-in-metadata-header", url: "https://mcp.example.com/rpc", headers: map[string]string{"X-Metadata": "ghp_123456789012345678901234567890"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validatePortableMCPRemoteEndpoint(testCase.url, testCase.headers); err == nil {
				t.Fatal("expected invalid remote MCP endpoint rejection")
			}
		})
	}
}

func TestMaterializeCodexMCPRejectsInvalidRemoteEndpointBeforeNativeRegistration(t *testing.T) {
	dir := t.TempDir()
	toolRoot := filepath.Join(dir, "bundled", "tools-mcp", "engineering", "demo-server")
	if err := os.MkdirAll(toolRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := "Bearer review-secret-value-123456789"
	config := `{"mcpServers":{"demo":{"type":"streamable-http","url":"https://mcp.example.com/rpc","headers":{"Authorization":"` + secret + `"}}}}` + "\n"
	if err := os.WriteFile(filepath.Join(toolRoot, ".mcp.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := materializeRuntimeComponent(dir, "codex", runtimeDependencyLockComponent{
		Module: "tools-mcp", ID: "engineering/demo-server", Version: "1.0.0",
		ManifestPath: "bundled/tools-mcp/engineering/demo-server/tool.yaml",
	})
	if err == nil || !strings.Contains(err.Error(), "must not contain credentials") {
		t.Fatalf("expected credential-bearing remote MCP header rejection, got %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("credential-bearing remote MCP header was echoed in the diagnostic")
	}
}

func TestValidatePortableMCPRemoteEndpointRedactsCredentialShapedHeaderName(t *testing.T) {
	secretName := "ghp_123456789012345678901234567890"
	err := validatePortableMCPRemoteEndpoint("https://mcp.example.com/rpc", map[string]string{secretName: "metadata"})
	if err == nil || !strings.Contains(err.Error(), "must not contain credentials") {
		t.Fatalf("expected credential-shaped remote MCP header rejection, got %v", err)
	}
	if strings.Contains(err.Error(), secretName) {
		t.Fatal("credential-shaped remote MCP header name was echoed in the diagnostic")
	}
}

func TestCompileRuntimeConfigDoesNotClaimUnsupportedNativeRegistration(t *testing.T) {
	registration := compileConfigRegistration("codex")
	if registration.Enforcement != "advisory" || registration.Reason == "" {
		t.Fatalf("unsupported config registration was not disclosed: %#v", registration)
	}
}

func TestCompileHookRegistrationDistinguishesExecutableAndAdvisoryHooks(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "check.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	validHook := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"${PLUGIN_ROOT}/scripts/check.sh"}]}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "hooks", "enforced.json"), []byte(validHook), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks", "guidance.md"), []byte("Review before publish.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executable, err := compileHookRegistration(dir, "codex", "enforced")
	if err != nil {
		t.Fatal(err)
	}
	advisory, err := compileHookRegistration(dir, "codex", "guidance")
	if err != nil {
		t.Fatal(err)
	}
	if executable.Enforcement != "native" || advisory.Enforcement != "advisory" || advisory.Reason == "" {
		t.Fatalf("unexpected hook classifications: native=%#v advisory=%#v", executable, advisory)
	}
}

func TestCompileHookRegistrationPreservesClaudeCommandFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "check.js"), []byte("process.exit(0)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"hooks":{"PostToolUse":[{"matcher":"Write","hooks":[{"type":"command","command":"node","args":["${CLAUDE_PLUGIN_ROOT}/scripts/check.js"],"asyncRewake":true},{"type":"command","command":"echo ready","shell":"bash"}]}]}}` + "\n")
	hookPath := filepath.Join(dir, "hooks", "claude-command-fields.json")
	if err := os.WriteFile(hookPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	registration, err := compileHookRegistration(dir, "claude", "claude-command-fields")
	if err != nil {
		t.Fatal(err)
	}
	if registration.Enforcement != "native" || registration.Path != "./hooks/claude-command-fields.json" {
		t.Fatalf("valid Claude command hook was not preserved as native: %#v", registration)
	}
	after, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, content) {
		t.Fatalf("Claude hook fields changed during compilation:\n%s", after)
	}
	codexRegistration, err := compileHookRegistration(dir, "codex", "claude-command-fields")
	if err != nil {
		t.Fatal(err)
	}
	if codexRegistration.Enforcement != "advisory" {
		t.Fatalf("Claude-only hook fields widened Codex validation: %#v", codexRegistration)
	}
}

func TestCompileHookRegistrationPreservesCodexMCPToolHook(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"hooks":{"PostToolUse":[{"matcher":"Write|Edit","hooks":[{"type":"mcp_tool","server":"scanner","tool":"scan_patch","input":{"path":"${tool_input.file_path}"},"timeout":30,"statusMessage":"Scanning"}]}]}}` + "\n")
	hookPath := filepath.Join(dir, "hooks", "hooks.json")
	if err := os.WriteFile(hookPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	registration, err := compileHookRegistration(dir, "codex", "hooks")
	if err != nil {
		t.Fatal(err)
	}
	if registration.Enforcement != "native" || registration.Path != "./hooks/hooks.json" {
		t.Fatalf("valid Codex MCP tool hook was not preserved as native: %#v", registration)
	}
	if after, err := os.ReadFile(hookPath); err != nil || !bytes.Equal(after, content) {
		t.Fatalf("Codex MCP tool hook was moved or changed: %v\n%s", err, after)
	}
}

func TestCompileHookRegistrationPreservesSupportedClaudeHandlerTypes(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		event   string
		handler string
	}{
		{name: "http", event: "PreToolUse", handler: `{"type":"http","url":"https://hooks.example.com/check","headers":{"Authorization":"Bearer $HOOK_TOKEN"},"allowedEnvVars":["HOOK_TOKEN"]}`},
		{name: "mcp-tool", event: "PostToolUse", handler: `{"type":"mcp_tool","server":"plugin:review:scanner","tool":"scan_patch","input":{"path":"${tool_input.file_path}"}}`},
		{name: "prompt", event: "Stop", handler: `{"type":"prompt","prompt":"Check completion: $ARGUMENTS","model":"haiku"}`},
		{name: "agent", event: "TaskCompleted", handler: `{"type":"agent","prompt":"Verify the task output: $ARGUMENTS"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := []byte(`{"hooks":{"` + testCase.event + `":[{"hooks":[` + testCase.handler + `]}]}}` + "\n")
			hookPath := filepath.Join(dir, "hooks", "hooks.json")
			if err := os.WriteFile(hookPath, content, 0o644); err != nil {
				t.Fatal(err)
			}
			registration, err := compileHookRegistration(dir, "claude", "hooks")
			if err != nil {
				t.Fatal(err)
			}
			if registration.Enforcement != "native" || registration.Path != "./hooks/hooks.json" {
				t.Fatalf("valid Claude %s hook was not preserved as native: %#v", testCase.name, registration)
			}
			if after, err := os.ReadFile(hookPath); err != nil || !bytes.Equal(after, content) {
				t.Fatalf("Claude %s hook was moved or changed: %v\n%s", testCase.name, err, after)
			}
		})
	}
}

func TestCompileHookRegistrationRejectsLiteralClaudeHTTPHookCredentials(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		headers string
		allowed string
		secret  string
	}{
		{name: "literal-authorization", headers: `{"Authorization":"Bearer review-secret-value-123456789"}`, secret: "review-secret-value-123456789"},
		{name: "literal-provider-token", headers: `{"X-Metadata":"ghp_123456789012345678901234567890"}`, secret: "ghp_123456789012345678901234567890"},
		{name: "undeclared-variable", headers: `{"Authorization":"Bearer $HOOK_TOKEN"}`, secret: ""},
		{name: "literal-suffix", headers: `{"Authorization":"Bearer ${HOOK_TOKEN}-literal"}`, allowed: `,"allowedEnvVars":["HOOK_TOKEN"]`, secret: ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"http","url":"https://hooks.example.com/check","headers":` + testCase.headers + testCase.allowed + `}]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "invalid.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := compileHookRegistration(dir, "claude", "invalid")
			if err == nil {
				t.Fatal("expected literal Claude HTTP hook credential rejection")
			}
			if testCase.secret != "" && strings.Contains(err.Error(), testCase.secret) {
				t.Fatal("literal Claude HTTP hook credential was echoed in the diagnostic")
			}
		})
	}
}

func TestCompileHookRegistrationRejectsCredentialInClaudeHTTPHookURL(t *testing.T) {
	secret := "ghp_123456789012345678901234567890"
	for _, testCase := range []struct {
		name string
		url  string
	}{
		{name: "literal-query", url: "https://hooks.example.com/check?token=" + secret},
		{name: "encoded-query", url: "https://hooks.example.com/check?token=ghp%5F123456789012345678901234567890"},
		{name: "double-encoded-query", url: "https://hooks.example.com/check?token=ghp%255F123456789012345678901234567890"},
		{name: "encoded-path", url: "https://hooks.example.com/ghp%5F123456789012345678901234567890/check"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"http","url":"` + testCase.url + `"}]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "invalid.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := compileHookRegistration(dir, "claude", "invalid")
			if err == nil || !strings.Contains(err.Error(), "url must not contain literal credentials") {
				t.Fatalf("expected credential-bearing Claude HTTP hook URL rejection, got %v", err)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "ghp%5F") {
				t.Fatal("Claude HTTP hook URL credential was echoed in the diagnostic")
			}
		})
	}
}

func TestCompileHookRegistrationEnforcesClaudeHTTPHookTransport(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		url        string
		wantNative bool
	}{
		{name: "remote-https", url: "https://hooks.example.com/check", wantNative: true},
		{name: "localhost-http", url: "http://localhost:8080/check", wantNative: true},
		{name: "uppercase-localhost-http", url: "http://LOCALHOST:8080/check", wantNative: true},
		{name: "ipv4-loopback-http", url: "http://127.0.0.1:8080/check", wantNative: true},
		{name: "ipv6-loopback-http", url: "http://[::1]:8080/check", wantNative: true},
		{name: "remote-http", url: "http://hooks.example.com/check"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"http","url":"` + testCase.url + `"}]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "transport.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			registration, err := compileHookRegistration(dir, "claude", "transport")
			if testCase.wantNative {
				if err != nil || registration.Enforcement != "native" {
					t.Fatalf("expected native Claude HTTP hook transport, got registration=%#v err=%v", registration, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "must use HTTPS") {
				t.Fatalf("expected non-loopback HTTP rejection, got %v", err)
			}
		})
	}
}

func TestCompileHookRegistrationRejectsMixedNativeAndAdvisoryHandlers(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"echo enforce"}]}],"Setup":[{"hooks":[{"type":"http","url":"https://hooks.example.com/setup"}]}]}}` + "\n")
	hookPath := filepath.Join(dir, "hooks", "hooks.json")
	if err := os.WriteFile(hookPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := compileHookRegistration(dir, "claude", "hooks")
	if err == nil || !strings.Contains(err.Error(), "mixes native and advisory handlers") {
		t.Fatalf("expected explicit mixed-enforcement rejection, got %v", err)
	}
	after, readErr := os.ReadFile(hookPath)
	if readErr != nil || !bytes.Equal(after, content) {
		t.Fatalf("mixed hook document was moved or changed after rejection: %v\n%s", readErr, after)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".runtime", "advisory", "hooks", "hooks.json")); !os.IsNotExist(statErr) {
		t.Fatalf("mixed hook document was silently quarantined: %v", statErr)
	}
}

func TestCompileHookRegistrationRejectsUnknownTopLevelFieldWithoutQuarantine(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"unsupported":true,"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"echo enforce"}]}]}}` + "\n")
	hookPath := filepath.Join(dir, "hooks", "hooks.json")
	if err := os.WriteFile(hookPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := compileHookRegistration(dir, "claude", "hooks")
	if err == nil || !strings.Contains(err.Error(), "top-level field") {
		t.Fatalf("expected explicit unknown top-level field rejection, got %v", err)
	}
	after, readErr := os.ReadFile(hookPath)
	if readErr != nil || !bytes.Equal(after, content) {
		t.Fatalf("hook document with an unknown top-level field was moved or changed: %v\n%s", readErr, after)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".runtime", "advisory", "hooks", "hooks.json")); !os.IsNotExist(statErr) {
		t.Fatalf("hook document with an unknown top-level field was silently quarantined: %v", statErr)
	}
}

func TestCompileHookRegistrationQuarantinesUnsupportedHandlerEventPairs(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		runtimeName string
		event       string
		handler     string
	}{
		{name: "codex-session-end-mcp", runtimeName: "codex", event: "SessionEnd", handler: `{"type":"mcp_tool","server":"scanner","tool":"finish"}`},
		{name: "claude-setup-http", runtimeName: "claude", event: "Setup", handler: `{"type":"http","url":"https://hooks.example.com/setup"}`},
		{name: "claude-setup-mcp", runtimeName: "claude", event: "Setup", handler: `{"type":"mcp_tool","server":"scanner","tool":"setup"}`},
		{name: "claude-notification-prompt", runtimeName: "claude", event: "Notification", handler: `{"type":"prompt","prompt":"Review: $ARGUMENTS"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"` + testCase.event + `":[{"hooks":[` + testCase.handler + `]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "hooks.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			registration, err := compileHookRegistration(dir, testCase.runtimeName, "hooks")
			if err != nil {
				t.Fatal(err)
			}
			if registration.Enforcement != "advisory" || registration.Reason == "" || registration.Path != "./.runtime/advisory/hooks/hooks.json" {
				t.Fatalf("inert handler/event pair was represented as native: %#v", registration)
			}
		})
	}
}

func TestCompileHookRegistrationRejectsMalformedNonCommandHandlers(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		runtimeName string
		event       string
		handler     string
	}{
		{name: "codex-mcp-input", runtimeName: "codex", event: "PostToolUse", handler: `{"type":"mcp_tool","server":"scanner","tool":"scan","input":"path"}`},
		{name: "claude-http-headers", runtimeName: "claude", event: "PreToolUse", handler: `{"type":"http","url":"https://hooks.example.com/check","headers":{"X-Count":1}}`},
		{name: "claude-http-env", runtimeName: "claude", event: "PreToolUse", handler: `{"type":"http","url":"https://hooks.example.com/check","allowedEnvVars":["BAD-NAME"]}`},
		{name: "claude-mcp-missing-tool", runtimeName: "claude", event: "PostToolUse", handler: `{"type":"mcp_tool","server":"scanner"}`},
		{name: "claude-prompt-missing-prompt", runtimeName: "claude", event: "Stop", handler: `{"type":"prompt"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"` + testCase.event + `":[{"hooks":[` + testCase.handler + `]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "invalid.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := compileHookRegistration(dir, testCase.runtimeName, "invalid"); err == nil {
				t.Fatal("expected malformed native handler rejection")
			}
		})
	}
}

func TestCompileHookRegistrationRejectsInvalidClaudeCommandFieldTypes(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		field string
	}{
		{name: "args", field: `"args":"script.js"`},
		{name: "null-args", field: `"args":null`},
		{name: "async-rewake", field: `"asyncRewake":"yes"`},
		{name: "null-async-rewake", field: `"asyncRewake":null`},
		{name: "shell", field: `"shell":true`},
		{name: "unsupported-shell", field: `"shell":"zsh"`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"PostToolUse":[{"hooks":[{"type":"command","command":"echo ready",` + testCase.field + `}]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "invalid.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := compileHookRegistration(dir, "claude", "invalid"); err == nil {
				t.Fatal("expected invalid Claude command hook rejection")
			}
		})
	}
}

func TestCompileHookRegistrationMakesInertClaudePluginFieldsAdvisory(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		event   string
		matcher string
		field   string
	}{
		{name: "once-in-plugin", event: "PreToolUse", field: `,"once":true`},
		{name: "if-on-non-tool-event", event: "SessionStart", field: `,"if":"Bash(git *)"`},
		{name: "codex-command-windows", event: "PreToolUse", field: `,"commandWindows":"echo ready"`},
		{name: "codex-context-limit", event: "PreToolUse", field: `,"additionalContextLimit":1024`},
		{name: "shell-with-exec-form", event: "PreToolUse", field: `,"args":[],"shell":"bash"`},
		{name: "timeout-with-async", event: "PostToolUse", field: `,"async":true,"timeout":30`},
		{name: "matcher-on-unsupported-event", event: "UserPromptSubmit", matcher: `"matcher":"prompt",`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"` + testCase.event + `":[{` + testCase.matcher + `"hooks":[{"type":"command","command":"echo ready"` + testCase.field + `}]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "inert.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			registration, err := compileHookRegistration(dir, "claude", "inert")
			if err != nil {
				t.Fatal(err)
			}
			if registration.Enforcement != "advisory" || registration.Reason == "" {
				t.Fatalf("inert Claude plugin field was represented as native: %#v", registration)
			}
		})
	}
}

func TestCompileHookRegistrationAcceptsContextualClaudeFilters(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		event   string
		matcher string
		field   string
	}{
		{name: "tool-if", event: "PreToolUse", field: `,"if":"Bash(git *)"`},
		{name: "session-matcher", event: "SessionStart", matcher: `"matcher":"startup",`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			content := `{"hooks":{"` + testCase.event + `":[{` + testCase.matcher + `"hooks":[{"type":"command","command":"echo ready"` + testCase.field + `}]}]}}` + "\n"
			if err := os.WriteFile(filepath.Join(dir, "hooks", "contextual.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			registration, err := compileHookRegistration(dir, "claude", "contextual")
			if err != nil {
				t.Fatal(err)
			}
			if registration.Enforcement != "native" {
				t.Fatalf("effective Claude filter was not represented as native: %#v", registration)
			}
		})
	}
}

func TestCompileHookRegistrationRejectsMalformedOrEscapingNativeHooks(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		content string
	}{
		{name: "malformed", content: "not-json\n"},
		{name: "empty", content: `{"hooks":{}}` + "\n"},
		{name: "escaping-command", content: `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"${PLUGIN_ROOT}/../outside.sh"}]}]}}` + "\n"},
		{name: "unrooted-command", content: `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"./scripts/check.sh"}]}]}}` + "\n"},
		{name: "wrong-field-type", content: `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"echo ok","timeout":"forever"}]}]}}` + "\n"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "hooks", "invalid.json"), []byte(testCase.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := compileHookRegistration(dir, "codex", "invalid"); err == nil {
				t.Fatal("expected invalid native hook rejection")
			}
		})
	}
}

func TestCompileHookRegistrationMarksUnsupportedEventAdvisory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"hooks":{"Notification":[{"hooks":[{"type":"command","command":"echo ready"}]}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "hooks", "claude-only.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	registration, err := compileHookRegistration(dir, "codex", "claude-only")
	if err != nil {
		t.Fatal(err)
	}
	if registration.Enforcement != "advisory" || registration.Reason == "" {
		t.Fatalf("unsupported event was not disclosed: %#v", registration)
	}
}

func TestCompileHookRegistrationQuarantinesAdvisoryDefaultHook(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"hooks":{"Notification":[{"hooks":[{"type":"command","command":"echo ready"}]}]}}` + "\n"
	defaultPath := filepath.Join(dir, "hooks", "hooks.json")
	if err := os.WriteFile(defaultPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	registration, err := compileHookRegistration(dir, "codex", "hooks")
	if err != nil {
		t.Fatal(err)
	}
	if registration.Enforcement != "advisory" || registration.Path != "./.runtime/advisory/hooks/hooks.json" {
		t.Fatalf("default advisory hook was not quarantined: %#v", registration)
	}
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("default hook remains natively discoverable: %v", err)
	}
}

func TestValidateCompiledRelativePathRejectsEscapes(t *testing.T) {
	for _, value := range []string{"../outside", "./../outside", "/absolute", ".\\outside", "./skills//demo"} {
		if err := validateCompiledRelativePath(value); err == nil {
			t.Fatalf("expected unsafe compiled path %q to be rejected", value)
		}
	}
}

func extractRuntimeCompilerFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	archive := filepath.Join("..", "..", "releases", "plugins", "marketing", "content-repurposing-plugin", "0.2.0", "package.tar.gz")
	if err := extractVerifiedTarGz(archive, dir); err != nil {
		t.Fatalf("extract runtime compiler fixture: %v", err)
	}
	return dir
}

func readCompiledRuntimeContract(t *testing.T, path string) compiledRuntimePackage {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read runtime contract: %v", err)
	}
	var contract compiledRuntimePackage
	if err := json.Unmarshal(payload, &contract); err != nil {
		t.Fatalf("parse runtime contract: %v", err)
	}
	return contract
}

func assertGoldenRuntimeArtifact(t *testing.T, actualPath, goldenName string) {
	t.Helper()
	actual, err := os.ReadFile(actualPath)
	if err != nil {
		t.Fatalf("read runtime artifact: %v", err)
	}
	goldenPath := filepath.Join("testdata", "runtime", goldenName)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
			t.Fatalf("update golden artifact: %v", err)
		}
	}
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden artifact: %v", err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("runtime artifact differs from golden %s:\n%s", goldenName, string(actual))
	}
}

func TestResolvePluginDependencyTargetGeneric(t *testing.T) {
	target, err := ResolvePluginDependencyTarget("generic", "skills", "/tmp/my-agent/plugins")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join("/tmp/my-agent", "skills")
	if target.TargetPath != expected {
		t.Fatalf("expected %s, got %s", expected, target.TargetPath)
	}
}
