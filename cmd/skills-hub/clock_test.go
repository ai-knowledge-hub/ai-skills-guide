package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestBuiltBinaryRejectsExpiredPackageUsingRealTime(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository root")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	root := t.TempDir()
	binary := filepath.Join(root, "skills-hub")
	build := exec.Command("go", "build", "-o", binary, "./cmd/skills-hub")
	build.Dir = repository
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real CLI: %v\n%s", err, output)
	}
	const id = "analytics/ga4-mcp-connector"
	moduleRoot := filepath.Join(root, "source")
	packageDir := filepath.Join(moduleRoot, filepath.FromSlash(id))
	extractClockReleaseFixture(t, filepath.Join(repository, "releases", "tools-mcp", filepath.FromSlash(id), "0.2.0", "package.tar.gz"), packageDir)
	manifestPath := filepath.Join(packageDir, "tool.yaml")
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	payload = bytes.Replace(payload, []byte(`last_verified_at: "2026-09-17T00:00:00Z"`), []byte(`last_verified_at: "2000-01-01T00:00:00Z"`), 1)
	if err := os.WriteFile(manifestPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := registry.ValidatePackageManifestAt(manifestPath, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("historical control must be otherwise admissible: %v", err)
	}
	entry := registry.ProjectManifest(manifest)
	entry.Latest, entry.Versions = manifest.Version, []registry.VersionEntry{{Version: manifest.Version}}
	indexPath := filepath.Join(root, "tools-index.json")
	if err := registry.WriteIndex(indexPath, registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{entry}}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "runtime", "tools-mcp")
	command := exec.Command(binary, "install", "--module", "tools", "--entry", id, "--registry", indexPath, "--root", moduleRoot, "--runtime", "generic", "--target", target)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "evidence older than") {
		t.Fatalf("real CLI did not enforce current admission: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(target, filepath.FromSlash(id))); !os.IsNotExist(err) {
		t.Fatalf("rejected package changed runtime target: %v", err)
	}
}

// Use the retained release so a legitimate update to the mutable connector
// source does not change this real-clock regression's fixture or assumptions.
func extractClockReleaseFixture(t *testing.T, archive, destination string) {
	t.Helper()
	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !filepath.IsLocal(header.Name) {
			t.Fatalf("non-local fixture path %q", header.Name)
		}
		path := filepath.Join(destination, filepath.FromSlash(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size > 2<<20 {
				t.Fatalf("oversized fixture member %q", header.Name)
			}
			payload, err := io.ReadAll(io.LimitReader(reader, (2<<20)+1))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, payload, os.FileMode(header.Mode)&0o777); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unsupported fixture member %q", header.Name)
		}
	}
}
