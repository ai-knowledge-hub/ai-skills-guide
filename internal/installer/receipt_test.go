package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestInstallReceiptBindsIdentityAndInstalledTree(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tool.yaml"), []byte("id: ads/example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	receipt := InstallReceipt{
		Source: "remote", Module: "tools", ID: "ads/example", Version: "1.0.0", Runtime: "generic",
		ArtifactSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: strings.Repeat("c", 64),
	}
	if err := WriteInstallReceipt(root, receipt); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyInstallReceipt(root, "tools", "ads/example", "1.0.0", "generic", strings.Repeat("a", 64), ""); err != nil {
		t.Fatalf("verify receipt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "tool.yaml"), []byte("id: ads/changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyInstallReceipt(root, "tools", "ads/example", "1.0.0", "generic", strings.Repeat("a", 64), ""); err == nil || !strings.Contains(err.Error(), "contents have changed") {
		t.Fatalf("expected installed tree mutation rejection, got %v", err)
	}
}

func TestRuntimeContractDigestChangesWithExecutableOrAuthenticationSemantics(t *testing.T) {
	entry := registry.SkillEntry{
		ID: "ads/example", Runtimes: []string{"generic"},
		Execution:      &registry.ExecutionMetadata{Kind: "cli", Command: []string{"bin/tool"}, SmokeTest: []string{"bin/tool", "--smoke"}},
		Authentication: &registry.AuthenticationMetadata{Status: "none", Methods: []string{"none"}},
	}
	original, err := RuntimeContractSHA256(entry, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	entry.Execution.SmokeTest = []string{"bin/other", "--smoke"}
	changed, err := RuntimeContractSHA256(entry, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if original == changed {
		t.Fatal("runtime contract digest did not change with smoke command")
	}
}

func TestInstallReceiptRejectsIdentityMismatch(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tool.yaml"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteInstallReceipt(root, InstallReceipt{
		Source: "local", Module: "tools", ID: "ads/example", Version: "1.0.0", Runtime: "generic", RuntimeContractSHA256: strings.Repeat("c", 64),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyInstallReceipt(root, "tools", "ads/other", "1.0.0", "generic", "", ""); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestInstallReceiptSurvivesInstallerModeNormalization(t *testing.T) {
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "bin", "tool"), []byte("fixture\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "metadata.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteInstallReceipt(source, InstallReceipt{
		Source: "remote", Module: "tools", ID: "ads/example", Version: "1.0.0", Runtime: "generic",
		ArtifactSHA256: strings.Repeat("b", 64), RuntimeContractSHA256: strings.Repeat("c", 64),
	}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "installed")
	if err := copyTree(source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyInstallReceipt(destination, "tools", "ads/example", "1.0.0", "generic", strings.Repeat("b", 64), ""); err != nil {
		t.Fatalf("verify normalized install: %v", err)
	}
}
