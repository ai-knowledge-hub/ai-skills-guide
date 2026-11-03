//go:build !runtime_readiness

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

// The release contract is represented independently of the command dispatcher.
// Observe a real binary, temporary destinations and unchanged source metadata.
func TestInstallerReleaseHasNoRuntimeAuthority(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	repository := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	root := t.TempDir()
	binary := filepath.Join(root, "skills-hub")
	environment := []string{}
	for _, value := range os.Environ() {
		if strings.Contains(strings.SplitN(value, "=", 2)[0], "MODEL_ATTESTATION_PUBLIC_KEY") || strings.HasPrefix(value, "GOFLAGS=") {
			continue
		}
		environment = append(environment, value)
	}
	build := exec.Command("go", "build", "-tags", "installer_release", "-o", binary, "./cmd/skills-hub")
	build.Dir, build.Env = repository, environment
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("installer build requires no model keys: %v\n%s", err, out)
	}
	run := func(args ...string) ([]byte, error) {
		command := exec.Command(binary, args...)
		command.Dir = root
		command.Env = append(append([]string{}, environment...), "HOME="+root, "CODEX_HOME="+filepath.Join(root, "codex"), "XDG_CONFIG_HOME="+filepath.Join(root, "config"), "GENERIC_MODEL_ATTESTATION_PUBLIC_KEY=invalid-fixture")
		return command.CombinedOutput()
	}
	help, err := run("help")
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"list", "search", "info", "validate", "install"} {
		if !strings.Contains(string(help), "  "+command+" ") {
			t.Fatalf("missing installer command %s:\n%s", command, help)
		}
	}
	for _, command := range []string{"auth", "doctor", "smoke", "run-agent"} {
		if strings.Contains(string(help), "  "+command+" ") {
			t.Fatalf("deferred command advertised: %s", command)
		}
		out, err := run(command, "--entry", "engineering/example@1.0.0", "--runtime", "codex", "--target", filepath.Join(root, "target"), "--state-dir", filepath.Join(root, "state"))
		if err == nil || !strings.Contains(string(out), "not available in the installation-only CLI") {
			t.Fatalf("deferred command %s: %v\n%s", command, err, out)
		}
	}
	for _, command := range []string{"list", "info"} {
		for _, flag := range []string{"--runtime", "--target", "--state-dir"} {
			out, err := run(command, flag, "unused")
			if err == nil || !strings.Contains(string(out), "flag provided but not defined") {
				t.Fatalf("readiness flag reached installer %s %s: %v\n%s", command, flag, err, out)
			}
		}
	}
	for _, name := range []string{"target", "state", "codex", "config"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("rejected operation changed %s: %v", name, err)
		}
	}
	const id = "engineering/implementation-strategy"
	source := filepath.Join(repository, "skills", filepath.FromSlash(id), "SKILL.md")
	want, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, adapter := range []string{"codex", "claude", "generic"} {
		target := filepath.Join(root, "installed", adapter)
		out, err := run("install", "--source", "local", "--module", "skills", "--registry", filepath.Join(repository, "registry", "skills-index.json"), "--root", filepath.Join(repository, "skills"), "--entry", id, "--runtime", adapter, "--target", target)
		if err != nil {
			t.Fatalf("installation without model/provider authority (%s): %v\n%s", adapter, err, out)
		}
		got, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(id), "SKILL.md"))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("installed bytes (%s): %v", adapter, err)
		}
		if !strings.Contains(string(out), "runtime operation has not been verified") {
			t.Fatalf("installation scope missing: %s", out)
		}
	}
	// An authenticated tool can be installed without resolving credentials or
	// starting its provider/runtime processes. Trap every advertised interpreter.
	toolRoot := filepath.Join(root, "source-tools")
	toolID := "analytics/ga4-mcp-connector"
	toolPackage := filepath.Join(toolRoot, filepath.FromSlash(toolID))
	extractClockReleaseFixture(t, filepath.Join(repository, "releases", "tools-mcp", filepath.FromSlash(toolID), "0.2.0", "package.tar.gz"), toolPackage)
	toolManifest, err := registry.ValidateHistoricalPackageManifest(filepath.Join(toolPackage, "tool.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	toolEntry := registry.ProjectManifest(toolManifest)
	toolEntry.Latest, toolEntry.Versions = toolManifest.Version, []registry.VersionEntry{{Version: toolManifest.Version}}
	toolIndex := filepath.Join(root, "tool-index.json")
	if err := registry.WriteIndex(toolIndex, registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{toolEntry}}); err != nil {
		t.Fatal(err)
	}
	trapDir := filepath.Join(root, "trap-bin")
	if err := os.Mkdir(trapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"node", "npx", "python", "python3"} {
		if err := os.WriteFile(filepath.Join(trapDir, name), []byte("#!/bin/sh\necho invoked > '"+filepath.Join(root, "provider-invoked")+"'\nexit 43\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	installTool := exec.CommandContext(ctx, binary, "install", "--source", "local", "--module", "tools", "--registry", toolIndex, "--root", toolRoot, "--entry", toolID+"@0.2.0", "--runtime", "generic", "--target", filepath.Join(root, "tool-target"))
	installTool.Dir = root
	installTool.Env = []string{"PATH=" + trapDir, "HOME=" + root}
	if out, err := installTool.CombinedOutput(); err != nil {
		t.Fatalf("eligible tool required provider execution/credentials: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "provider-invoked")); !os.IsNotExist(err) {
		t.Fatalf("installation invoked a runtime/provider process: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "tool-target", filepath.FromSlash(toolID), ".skills-hub-install.json")); err != nil {
		t.Fatal(err)
	}
	// A coherent positive catalogue declaration is not evidence of local readiness.
	indexPath := filepath.Join(root, "index.json")
	entry := registry.SkillEntry{ID: "engineering/example", Name: "Example", Latest: "1.0.0", Runtimes: []string{"codex"}, Versions: []registry.VersionEntry{{Version: "1.0.0"}}, Usability: registry.UsabilityMetadata{Availability: "usable-now", Execution: "local-tool", ExecutableHelpers: []registry.ExecutableHelperMetadata{{Entrypoint: "scripts/check.sh", Availability: "usable-now"}}}}
	if err := registry.WriteIndex(indexPath, registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{entry}}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(indexPath)
	for _, args := range [][]string{{"list", "--registry", indexPath}, {"info", "--registry", indexPath, "--entry", entry.ID}} {
		out, err := run(args...)
		if err != nil || strings.Contains(string(out), "usable-now") || !strings.Contains(string(out), "not-verified") {
			t.Fatalf("static declaration became runtime authority: %v\n%s", err, out)
		}
	}
	if out, err := run("search", "--registry", indexPath, "--runtime", "codex"); err != nil || !strings.Contains(string(out), entry.ID) {
		t.Fatalf("metadata runtime filter lost: %v\n%s", err, out)
	}
	after, _ := os.ReadFile(indexPath)
	if !bytes.Equal(before, after) {
		t.Fatal("display projection mutated admission metadata")
	}
	// Explicitly conflicting build profiles must not produce a public artifact.
	forbidden := filepath.Join(root, "forbidden")
	conflict := exec.Command("go", "build", "-tags", "installer_release,runtime_readiness", "-o", forbidden, "./cmd/skills-hub")
	conflict.Dir, conflict.Env = repository, environment
	if out, err := conflict.CombinedOutput(); err == nil {
		t.Fatalf("conflicting profiles built a release: %s", out)
	}
	if _, err := os.Stat(forbidden); !os.IsNotExist(err) {
		t.Fatalf("conflicting build produced an artifact: %v", err)
	}
}

func TestInstallerProviderGuidanceDoesNotAdvertiseDeferredCommands(t *testing.T) {
	entry := registry.SkillEntry{ProviderDependencies: []registry.ProviderDependencyMetadata{{Tool: "analytics/ga4-mcp-connector", Requirement: "required", Access: "read-only"}, {Tool: "warehouse/bigquery-mcp-query-runner", Requirement: "optional", Access: "read-only"}}}
	var out bytes.Buffer
	printProviderAuthNextStep(&out, entry, "1.0.0", "codex", "/tmp/unused")
	for _, want := range []string{"analytics/ga4-mcp-connector", "warehouse/bigquery-mcp-query-runner", "installed README", "has not been verified"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing setup guidance %q: %s", want, out.String())
		}
	}
	for _, command := range []string{"skills-hub auth", "skills-hub doctor", "skills-hub smoke", "skills-hub run-agent"} {
		if strings.Contains(out.String(), command) {
			t.Fatalf("unavailable next step: %s", out.String())
		}
	}
}
