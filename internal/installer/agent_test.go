package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestAgentInstallCompilesPinnedBindingsWithoutManualFile(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	indexes := writeAgentTestIndexes(t, repositoryRoot)
	agentManifest, err := registry.ValidatePackageManifest(filepath.Join(repositoryRoot, "agents", "adtech", "bi-insights-orchestrator", "agent.yaml"))
	if err != nil {
		t.Fatalf("validate agent: %v", err)
	}
	entry := registry.ProjectManifest(agentManifest)
	targetRoot := filepath.Join(t.TempDir(), "agents")
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	destination, err := InstallSkill(filepath.Join(repositoryRoot, "agents", filepath.FromSlash(entry.ID)), targetRoot, entry.ID, false)
	if err != nil {
		t.Fatalf("install agent: %v", err)
	}
	result, err := InstallAgentDependencies(entry, "codex", targetRoot, map[string]string{
		"skills": filepath.Join(repositoryRoot, "skills"), "agents": filepath.Join(repositoryRoot, "agents"), "tools": filepath.Join(repositoryRoot, "tools-mcp"),
	}, indexes, false)
	if err != nil {
		t.Fatalf("install dependencies: %v", err)
	}
	if len(result.InstalledSkills) != 3 || len(result.InstalledTools) != 2 {
		t.Fatalf("unexpected closure: %#v", result)
	}
	contractPath, err := CompileAgentRuntimePackage(destination, "codex", targetRoot)
	if err != nil {
		t.Fatalf("compile agent: %v", err)
	}
	payload, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var contract compiledAgentRuntime
	if err := json.Unmarshal(payload, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Status != "setup-required" {
		t.Fatalf("status = %s, want setup-required: %#v", contract.Status, contract.BlockingReasons)
	}
	if strings.Contains(strings.Join(contract.BlockingReasons, "\n"), "model") {
		t.Fatalf("compiled contract mixed dynamic model readiness into static blockers: %#v", contract.BlockingReasons)
	}
	for _, name := range []string{"bigquery_sql", "ga4_query"} {
		found := false
		for _, binding := range contract.Bindings {
			if binding.Name == name {
				found = true
				if binding.Package == "" || binding.Version == "" || binding.ConfigureHint == "" {
					t.Fatalf("incomplete %s binding: %#v", name, binding)
				}
			}
		}
		if !found {
			t.Fatalf("missing compiled binding %s", name)
		}
	}
	contractDigest, err := RuntimeContractSHA256(entry, agentManifest.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteInstallReceipt(destination, InstallReceipt{
		Source: "local", Module: "agents", ID: entry.ID, Version: agentManifest.Version,
		Runtime: "codex", RuntimeContractSHA256: contractDigest, Closure: result.Closure,
	}); err != nil {
		t.Fatal(err)
	}
	receipt, err := VerifyInstallReceipt(destination, "agents", entry.ID, agentManifest.Version, "codex", "", contractDigest)
	if err != nil {
		t.Fatal(err)
	}
	moduleRoots := map[string]string{
		"agents": targetRoot,
		"skills": filepath.Join(filepath.Dir(targetRoot), "skills"),
		"tools":  filepath.Join(filepath.Dir(targetRoot), "tools-mcp"),
	}
	if err := VerifyAgentInstallClosure(receipt, moduleRoots, entry.Orchestration); err != nil {
		t.Fatalf("verify installed agent closure: %v", err)
	}
	incomplete := receipt
	incomplete.Closure = append([]InstallReceiptMember(nil), receipt.Closure[:len(receipt.Closure)-1]...)
	if err := VerifyAgentInstallClosure(incomplete, moduleRoots, entry.Orchestration); err == nil || !strings.Contains(err.Error(), "complete agent closure") {
		t.Fatalf("expected omitted closure member rejection, got %v", err)
	}
	memberPath := filepath.Join(moduleRoots["skills"], "adtech", "dashboard-generator", "README.md")
	memberBytes, err := os.ReadFile(memberPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(memberPath, append(memberBytes, []byte("\nmodified\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyAgentInstallClosure(receipt, moduleRoots, entry.Orchestration); err == nil || !strings.Contains(err.Error(), "contents have changed") {
		t.Fatalf("expected mutated closure member rejection, got %v", err)
	}
}

func TestAgentCompileBlocksUnavailableRequiredCapability(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	indexes := writeAgentTestIndexes(t, repositoryRoot)
	manifest, err := registry.ValidatePackageManifest(filepath.Join(repositoryRoot, "agents", "marketing", "campaign-qa-supervisor", "agent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	entry := registry.ProjectManifest(manifest)
	targetRoot := filepath.Join(t.TempDir(), "agents")
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	destination, err := InstallSkill(filepath.Join(repositoryRoot, "agents", filepath.FromSlash(entry.ID)), targetRoot, entry.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = InstallAgentDependencies(entry, "generic", targetRoot, map[string]string{
		"skills": filepath.Join(repositoryRoot, "skills"), "agents": filepath.Join(repositoryRoot, "agents"), "tools": filepath.Join(repositoryRoot, "tools-mcp"),
	}, indexes, false)
	if err != nil {
		t.Fatal(err)
	}
	contractPath, err := CompileAgentRuntimePackage(destination, "generic", targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := os.ReadFile(contractPath)
	if !strings.Contains(string(payload), `"status": "blocked"`) || !strings.Contains(string(payload), "campaign_config_reader") {
		t.Fatalf("unavailable required capability was not actionable: %s", payload)
	}
}

func TestReleasedAgentArchiveInstallsSelfContainedClosure(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	archive := filepath.Join(repositoryRoot, "releases", "agents", "adtech", "bi-insights-orchestrator", "0.2.0", "package.tar.gz")
	extractRoot := t.TempDir()
	if err := extractVerifiedTarGz(archive, extractRoot); err != nil {
		t.Fatalf("extract release: %v", err)
	}
	manifest, err := registry.ValidatePackageManifest(filepath.Join(extractRoot, "agent.yaml"))
	if err != nil {
		t.Fatalf("validate released agent: %v", err)
	}
	runtimeRoot := t.TempDir()
	agentTarget := filepath.Join(runtimeRoot, "agents")
	plans, err := remoteAgentInstallPlans(extractRoot, agentTarget, "generic", []string{"node22"}, manifest)
	if err != nil {
		t.Fatalf("plan closure: %v", err)
	}
	if len(plans) != 6 {
		t.Fatalf("closure plans=%d want=6", len(plans))
	}
	if _, err := compileAgentRuntimePackage(extractRoot, "generic", agentTarget, extractRoot); err != nil {
		t.Fatalf("compile release: %v", err)
	}
	destination, err := atomicInstallClosure(plans, agentTarget, manifest.ID, false)
	if err != nil {
		t.Fatalf("commit closure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, ".runtime", "generic.json")); err != nil {
		t.Fatalf("compiled contract missing: %v", err)
	}
	for _, expected := range []string{
		filepath.Join(runtimeRoot, "skills", "adtech", "dashboard-generator", "skill.yaml"),
		filepath.Join(runtimeRoot, "tools-mcp", "analytics", "ga4-mcp-connector", "tool.yaml"),
		filepath.Join(runtimeRoot, "tools-mcp", "warehouse", "bigquery-mcp-query-runner", "tool.yaml"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("closure member missing %s: %v", expected, err)
		}
	}
}

func writeAgentTestIndexes(t *testing.T, root string) map[string]string {
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
