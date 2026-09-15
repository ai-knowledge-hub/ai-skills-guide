package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	"github.com/gofrs/flock"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestInstallRemoteReleaseCleanClientAndOfflineCache(t *testing.T) {
	archive := makeArchive(t, map[string]string{
		"skill.yaml": remoteManifest("1.0.0"),
		"SKILL.md":   "# Remote fixture\n",
	})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	cacheDir := filepath.Join(t.TempDir(), "cache")
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	options := remoteOptions(server, cacheDir, targetRoot, "latest")

	result, err := InstallRemoteRelease(context.Background(), options)
	if err != nil {
		t.Fatalf("clean-client install failed: %v", err)
	}
	if result.FromCache {
		t.Fatal("first install unexpectedly reported an artifact cache hit")
	}
	assertFileContains(t, filepath.Join(result.Destination, "SKILL.md"), "Remote fixture")

	if err := os.RemoveAll(targetRoot); err != nil {
		t.Fatalf("remove first installation: %v", err)
	}
	server.Close()
	options.Offline = true
	options.Version = "1.0.0"
	result, err = InstallRemoteRelease(context.Background(), options)
	if err != nil {
		t.Fatalf("offline cached install failed: %v", err)
	}
	if !result.FromCache {
		t.Fatal("offline install did not report a verified cache hit")
	}
	assertFileContains(t, filepath.Join(result.Destination, "SKILL.md"), "Remote fixture")

	options.ID = "engineering/other-package"
	if _, err := InstallRemoteRelease(context.Background(), options); err == nil || !strings.Contains(err.Error(), "offline registry lookup failed") {
		t.Fatalf("package A cache was incorrectly reusable for package B: %v", err)
	}
}

func TestInstallRemoteReleaseRejectsChecksumMismatchWithoutRuntimeMutation(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	wrongChecksum := strings.Repeat("0", sha256.Size*2)
	server := newReleaseServer(t, archive, "1.0.0", &wrongChecksum)
	defer server.Close()
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0")

	_, err := InstallRemoteRelease(context.Background(), options)
	if err == nil || !strings.Contains(err.Error(), "checksum validation failed") {
		t.Fatalf("expected checksum rejection, got %v", err)
	}
	assertPathAbsent(t, targetRoot)
}

func TestInstallRemoteReleaseRejectsUnsupportedRuntimeFromSelectedManifest(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0")
	options.Runtime = "codex"

	_, err := InstallRemoteRelease(context.Background(), options)
	if err == nil || !strings.Contains(err.Error(), "does not declare runtime codex") {
		t.Fatalf("expected runtime admission rejection, got %v", err)
	}
	assertPathAbsent(t, targetRoot)
}

func TestInstallRemoteReleaseRejectsUnsupportedHostPlatformBeforeMutation(t *testing.T) {
	hostPlatform, err := currentHostPlatform()
	if err != nil {
		t.Fatal(err)
	}
	unsupportedPlatform := "windows"
	if hostPlatform == unsupportedPlatform {
		unsupportedPlatform = "linux"
	}
	manifest := strings.Replace(
		remoteUsableNowManifest("1.0.0", time.Now().UTC()),
		"supported_platforms: [linux]",
		"supported_platforms: ["+unsupportedPlatform+"]",
		1,
	)
	archive := makeArchive(t, map[string]string{
		"skill.yaml": manifest, "SKILL.md": "# Fixture\n", "bin/tool": "fixture\n",
		"checksums.txt": "fixture\n", "sbom.cdx.json": "{}\n",
	})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")

	_, err = InstallRemoteRelease(context.Background(), remoteOptions(
		server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0",
	))
	if err == nil || !strings.Contains(err.Error(), "host platform "+hostPlatform) {
		t.Fatalf("expected host-platform compatibility rejection, got %v", err)
	}
	assertPathAbsent(t, targetRoot)
}

func TestInstallRemoteReleaseRequiresDeclaredExecutionRuntime(t *testing.T) {
	manifest := strings.Replace(
		strings.Replace(
			remoteUsableNowManifest("1.0.0", time.Now().UTC()),
			"supported_platforms: [linux]",
			"supported_platforms: [linux, macos, windows]",
			1,
		),
		"supported_runtimes: [native]",
		"supported_runtimes: [node22]",
		1,
	)
	archive := makeArchive(t, map[string]string{
		"skill.yaml": manifest, "SKILL.md": "# Fixture\n", "bin/tool": "fixture\n",
		"checksums.txt": "fixture\n", "sbom.cdx.json": "{}\n",
	})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	options := remoteOptions(
		server,
		filepath.Join(t.TempDir(), "cache"),
		filepath.Join(t.TempDir(), "runtime", "skills"),
		"1.0.0",
	)

	if _, err := InstallRemoteRelease(context.Background(), options); err == nil || !strings.Contains(err.Error(), "--execution-runtime") {
		t.Fatalf("expected execution-runtime compatibility rejection, got %v", err)
	}
	assertPathAbsent(t, options.TargetRoot)
	options.ExecutionRuntimes = []string{"node22"}
	if _, err := InstallRemoteRelease(context.Background(), options); err != nil {
		t.Fatalf("explicit compatible execution runtime was rejected: %v", err)
	}
}

func TestInstallRemoteReleaseRejectsTemplateOnlyArchiveManifest(t *testing.T) {
	manifest := strings.Replace(
		remoteManifest("1.0.0"),
		"availability: documentation-only",
		"availability: template-only",
		1,
	)
	archive := makeArchive(t, map[string]string{"skill.yaml": manifest, "SKILL.md": "# Fixture\n"})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")

	_, err := InstallRemoteRelease(context.Background(), remoteOptions(
		server,
		filepath.Join(t.TempDir(), "cache"),
		targetRoot,
		"1.0.0",
	))
	if err == nil || !strings.Contains(err.Error(), "package admission failed") || !strings.Contains(err.Error(), "template-only") {
		t.Fatalf("expected archive-manifest admission rejection, got %v", err)
	}
	assertPathAbsent(t, targetRoot)
}

func TestInstallRemoteReleasePreservesCurrentDeprecationPolicy(t *testing.T) {
	manifest := strings.Replace(
		remoteUsableNowManifest("1.0.0", time.Now().UTC()),
		"supported_platforms: [linux]",
		"supported_platforms: [linux, macos, windows]",
		1,
	)
	manifest = strings.Replace(
		manifest,
		"  quickstart: bin/tool\n",
		"  quickstart: bin/tool\n  executable_helpers:\n    - entrypoint: bin/tool\n      availability: usable-now\n      execution: local-tool\n      limitations: [Deprecated fixture helper.]\n",
		1,
	)
	archive := makeArchive(t, map[string]string{
		"skill.yaml": manifest, "SKILL.md": "# Deprecated fixture\n", "bin/tool": "fixture\n",
		"checksums.txt": "fixture\n", "sbom.cdx.json": "{}\n",
	})
	server := newReleaseServerWithWrapper(t, archive, "1.0.0", nil, func(original http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/registry.json" {
				original.ServeHTTP(w, request)
				return
			}
			recorder := httptest.NewRecorder()
			original.ServeHTTP(recorder, request)
			var index registry.Index
			if err := json.Unmarshal(recorder.Body.Bytes(), &index); err != nil {
				t.Fatalf("decode fixture registry: %v", err)
			}
			index.Skills[0].Deprecated = true
			index.Skills[0].ReplacedBy = "engineering/replacement-fixture"
			index.Skills[0].Readiness = "deprecated"
			if err := json.NewEncoder(w).Encode(index); err != nil {
				t.Errorf("encode fixture registry: %v", err)
			}
		})
	})
	defer server.Close()

	result, err := InstallRemoteRelease(context.Background(), remoteOptions(
		server,
		filepath.Join(t.TempDir(), "cache"),
		filepath.Join(t.TempDir(), "runtime", "skills"),
		"1.0.0",
	))
	if err != nil {
		t.Fatalf("deprecated release should follow warning policy, got %v", err)
	}
	if !result.Registry.Deprecated || result.Registry.Readiness != "deprecated" {
		t.Fatalf("current lifecycle policy was discarded: %#v", result.Registry)
	}
	if result.Registry.ReplacedBy != "engineering/replacement-fixture" {
		t.Fatalf("current replacement was discarded: %#v", result.Registry)
	}
	if result.Registry.Usability.Availability != "not-verified" || result.Registry.Usability.Source != "inferred" {
		t.Fatalf("deprecated release retained positive usability: %#v", result.Registry.Usability)
	}
	if len(result.Registry.Usability.ExecutableHelpers) != 1 || result.Registry.Usability.ExecutableHelpers[0].Availability != "not-verified" {
		t.Fatalf("deprecated remote helper retained positive usability: %#v", result.Registry.Usability.ExecutableHelpers)
	}
}

func TestInstallRemotePluginReleaseActivatesValidatedDependencyClosure(t *testing.T) {
	pluginManifest := remotePluginManifest("1.0.0", time.Now().UTC())
	const dependencyID = "engineering/closure-skill"
	dependencyManifest := strings.Replace(
		remoteManifest("1.0.0"),
		"id: engineering/remote-fixture",
		"id: "+dependencyID,
		1,
	)
	archive := makeArchive(t, map[string]string{
		"plugin.yaml": pluginManifest,
		"plugin.json": "{}\n",
		"bundled/skills/engineering/closure-skill/skill.yaml": dependencyManifest,
		"bundled/skills/engineering/closure-skill/SKILL.md":   "# Bundled closure skill\n",
	})
	server := newPluginReleaseServer(t, archive, pluginManifest)
	defer server.Close()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	pluginRoot := filepath.Join(runtimeRoot, "plugins")

	result, err := InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "plugins",
		Runtime:     "generic",
		ID:          "engineering/closure-plugin",
		Version:     "1.0.0",
		TargetRoot:  pluginRoot,
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("install remote plugin closure: %v", err)
	}
	assertFileContains(t, filepath.Join(result.Destination, "plugin.yaml"), "engineering/closure-plugin")
	assertFileContains(
		t,
		filepath.Join(runtimeRoot, "skills", filepath.FromSlash(dependencyID), "SKILL.md"),
		"Bundled closure skill",
	)
}

func TestInstallRemotePluginReleaseRejectsTemplateDependencyBeforeMutation(t *testing.T) {
	pluginManifest := remotePluginManifest("1.0.0", time.Now().UTC())
	dependencyManifest := strings.Replace(
		strings.Replace(remoteManifest("1.0.0"), "id: engineering/remote-fixture", "id: engineering/closure-skill", 1),
		"availability: documentation-only",
		"availability: template-only",
		1,
	)
	archive := makeArchive(t, map[string]string{
		"plugin.yaml": pluginManifest,
		"plugin.json": "{}\n",
		"bundled/skills/engineering/closure-skill/skill.yaml": dependencyManifest,
		"bundled/skills/engineering/closure-skill/SKILL.md":   "# Template closure skill\n",
	})
	server := newPluginReleaseServer(t, archive, pluginManifest)
	defer server.Close()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")

	_, err := InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "plugins",
		Runtime:     "generic",
		ID:          "engineering/closure-plugin",
		Version:     "1.0.0",
		TargetRoot:  filepath.Join(runtimeRoot, "plugins"),
		HTTPClient:  server.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "template-only") {
		t.Fatalf("expected template-only dependency rejection, got %v", err)
	}
	assertPathAbsent(t, filepath.Join(runtimeRoot, "plugins", "engineering", "closure-plugin"))
	assertPathAbsent(t, filepath.Join(runtimeRoot, "skills", "engineering", "closure-skill"))
}

func TestInstallRemotePluginReleaseRejectsIncompatibleDependencyPlatformBeforeMutation(t *testing.T) {
	hostPlatform, err := currentHostPlatform()
	if err != nil {
		t.Fatal(err)
	}
	unsupportedPlatform := "windows"
	if hostPlatform == unsupportedPlatform {
		unsupportedPlatform = "linux"
	}
	pluginManifest := remotePluginManifest("1.0.0", time.Now().UTC())
	dependencyManifest := strings.Replace(
		strings.Replace(
			remoteUsableNowManifest("1.0.0", time.Now().UTC()),
			"id: engineering/remote-fixture",
			"id: engineering/closure-skill",
			1,
		),
		"supported_platforms: [linux]",
		"supported_platforms: ["+unsupportedPlatform+"]",
		1,
	)
	archive := makeArchive(t, map[string]string{
		"plugin.yaml": pluginManifest,
		"plugin.json": "{}\n",
		"bundled/skills/engineering/closure-skill/skill.yaml":    dependencyManifest,
		"bundled/skills/engineering/closure-skill/SKILL.md":      "# Executable closure skill\n",
		"bundled/skills/engineering/closure-skill/bin/tool":      "fixture\n",
		"bundled/skills/engineering/closure-skill/checksums.txt": "fixture\n",
		"bundled/skills/engineering/closure-skill/sbom.cdx.json": "{}\n",
	})
	server := newPluginReleaseServer(t, archive, pluginManifest)
	defer server.Close()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")

	_, err = InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "plugins",
		Runtime:     "generic",
		ID:          "engineering/closure-plugin",
		Version:     "1.0.0",
		TargetRoot:  filepath.Join(runtimeRoot, "plugins"),
		HTTPClient:  server.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "host platform "+hostPlatform) {
		t.Fatalf("expected bundled dependency platform rejection, got %v", err)
	}
	assertPathAbsent(t, filepath.Join(runtimeRoot, "plugins", "engineering", "closure-plugin"))
	assertPathAbsent(t, filepath.Join(runtimeRoot, "skills", "engineering", "closure-skill"))
}

func TestInstallRemotePluginReleaseRollsBackWholeClosureOnActivationFailure(t *testing.T) {
	pluginManifest := remotePluginManifest("1.0.0", time.Now().UTC())
	dependencyManifest := strings.Replace(remoteManifest("1.0.0"), "id: engineering/remote-fixture", "id: engineering/closure-skill", 1)
	archive := makeArchive(t, map[string]string{
		"plugin.yaml": pluginManifest,
		"plugin.json": "{}\n",
		"bundled/skills/engineering/closure-skill/skill.yaml": dependencyManifest,
		"bundled/skills/engineering/closure-skill/SKILL.md":   "# Bundled closure skill\n",
	})
	server := newPluginReleaseServer(t, archive, pluginManifest)
	defer server.Close()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	oldDependency := filepath.Join(runtimeRoot, "skills", "engineering", "closure-skill", "old.txt")
	if err := os.MkdirAll(filepath.Dir(oldDependency), 0o755); err != nil {
		t.Fatalf("create previous dependency: %v", err)
	}
	if err := os.WriteFile(oldDependency, []byte("previous\n"), 0o644); err != nil {
		t.Fatalf("write previous dependency: %v", err)
	}
	pluginDestination := filepath.Join(runtimeRoot, "plugins", "engineering", "closure-plugin")
	closureInstallTransactionFault = func(point string, index int) error {
		if point == "after-activation" && index == 1 {
			assertPathAbsent(t, pluginDestination)
			return errors.New("injected closure activation failure")
		}
		return nil
	}
	t.Cleanup(func() { closureInstallTransactionFault = nil })

	_, err := InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "plugins",
		Runtime:     "generic",
		ID:          "engineering/closure-plugin",
		Version:     "1.0.0",
		TargetRoot:  filepath.Join(runtimeRoot, "plugins"),
		Force:       true,
		HTTPClient:  server.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "injected closure activation failure") {
		t.Fatalf("expected injected closure failure, got %v", err)
	}
	assertPathAbsent(t, pluginDestination)
	assertFileContains(t, oldDependency, "previous")
}

func TestInstallRemotePluginReleaseRejectsInPlaceClosureUpgrade(t *testing.T) {
	pluginManifest := remotePluginManifest("1.0.0", time.Now().UTC())
	dependencyManifest := strings.Replace(remoteManifest("1.0.0"), "id: engineering/remote-fixture", "id: engineering/closure-skill", 1)
	archive := makeArchive(t, map[string]string{
		"plugin.yaml": pluginManifest,
		"plugin.json": "{}\n",
		"bundled/skills/engineering/closure-skill/skill.yaml": dependencyManifest,
		"bundled/skills/engineering/closure-skill/SKILL.md":   "# Bundled closure skill\n",
	})
	server := newPluginReleaseServer(t, archive, pluginManifest)
	defer server.Close()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	oldPlugin := filepath.Join(runtimeRoot, "plugins", "engineering", "closure-plugin", "old.txt")
	oldDependency := filepath.Join(runtimeRoot, "skills", "engineering", "closure-skill", "old.txt")
	for _, oldPath := range []string{oldPlugin, oldDependency} {
		if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
			t.Fatalf("create previous closure member: %v", err)
		}
		if err := os.WriteFile(oldPath, []byte("previous\n"), 0o644); err != nil {
			t.Fatalf("write previous closure member: %v", err)
		}
	}

	_, err := InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "plugins",
		Runtime:     "generic",
		ID:          "engineering/closure-plugin",
		Version:     "1.0.0",
		TargetRoot:  filepath.Join(runtimeRoot, "plugins"),
		Force:       true,
		HTTPClient:  server.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "atomic activation pointer") {
		t.Fatalf("expected unsafe in-place closure upgrade rejection, got %v", err)
	}
	assertFileContains(t, oldPlugin, "previous")
	assertFileContains(t, oldDependency, "previous")
}

func TestRecoverInterruptedClosureInstallBlocksWhenRecordedLockIsBusy(t *testing.T) {
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	pluginDestination := filepath.Join(runtimeRoot, "plugins", "engineering", "closure-plugin")
	dependencyDestination := filepath.Join(runtimeRoot, "skills", "engineering", "closure-skill")
	operations := make([]closureInstallOperation, 0, 2)
	for _, destination := range []string{pluginDestination, dependencyDestination} {
		parent := filepath.Dir(destination)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			t.Fatalf("create transaction parent: %v", err)
		}
		stage, err := os.MkdirTemp(parent, ".skills-hub-stage-")
		if err != nil {
			t.Fatalf("create transaction stage: %v", err)
		}
		operations = append(operations, closureInstallOperation{Destination: destination, Stage: stage})
	}
	markerPath := closureTransactionMarker(runtimeRoot, "engineering/closure-plugin")
	if err := writeClosureInstallTransaction(markerPath, closureInstallTransaction{
		Phase: "prepared", Operations: operations,
	}); err != nil {
		t.Fatalf("write interrupted closure transaction: %v", err)
	}

	dependencyLock := flock.New(filepath.Join(
		filepath.Dir(dependencyDestination),
		"."+filepath.Base(dependencyDestination)+".skills-hub.lock",
	))
	locked, err := dependencyLock.TryLock()
	if err != nil || !locked {
		t.Fatalf("hold dependency lock: locked=%v err=%v", locked, err)
	}

	err = recoverInterruptedClosureInstalls(runtimeRoot)
	if err == nil || !strings.Contains(err.Error(), "is busy") {
		t.Fatalf("expected busy recovery blocker, got %v", err)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("busy recovery removed its durable marker: %v", err)
	}
	if _, err := os.Stat(operations[0].Stage); err != nil {
		t.Fatalf("busy recovery mutated transaction state without every lock: %v", err)
	}
	if err := dependencyLock.Unlock(); err != nil {
		t.Fatalf("release dependency lock: %v", err)
	}
	if err := recoverInterruptedClosureInstalls(runtimeRoot); err != nil {
		t.Fatalf("recover closure after lock release: %v", err)
	}
	assertPathAbsent(t, markerPath)
}

func TestInstallRemoteReleaseRejectsManifestSchemaViolations(t *testing.T) {
	for _, test := range []struct {
		name     string
		manifest string
	}{
		{
			name:     "unknown property",
			manifest: remoteManifest("1.0.0") + "unknown_contract_field: true\n",
		},
		{
			name: "unknown availability",
			manifest: strings.Replace(
				remoteManifest("1.0.0"),
				"availability: documentation-only",
				"availability: future-ready",
				1,
			),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive := makeArchive(t, map[string]string{"skill.yaml": test.manifest, "SKILL.md": "# Fixture\n"})
			server := newReleaseServer(t, archive, "1.0.0", nil)
			defer server.Close()
			targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")

			_, err := InstallRemoteRelease(context.Background(), remoteOptions(
				server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0",
			))
			if err == nil || !strings.Contains(err.Error(), "does not satisfy skill.schema.json") {
				t.Fatalf("expected canonical schema rejection, got %v", err)
			}
			assertPathAbsent(t, targetRoot)
		})
	}
}

func TestRetainedReleaseAllowsExpiredEvidenceButCurrentAdmissionRejectsIt(t *testing.T) {
	manifest := remoteUsableNowManifest("1.0.0", time.Now().UTC().Add(-365*24*time.Hour))
	files := map[string]string{
		"skill.yaml":    manifest,
		"SKILL.md":      "# Historical fixture\n",
		"bin/tool":      "fixture\n",
		"checksums.txt": "fixture\n",
		"sbom.cdx.json": "{}\n",
	}
	releaseDir := t.TempDir()
	archivePath := filepath.Join(releaseDir, "package.tar.gz")
	manifestPath := filepath.Join(releaseDir, "skill.yaml")
	if err := os.WriteFile(archivePath, makeArchive(t, files), 0o644); err != nil {
		t.Fatalf("write historical archive: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write historical manifest: %v", err)
	}

	if _, err := ValidateRetainedRelease(
		archivePath, manifestPath, "skills", "engineering/remote-fixture", "1.0.0",
	); err != nil {
		t.Fatalf("expired evidence made immutable history unpublishable: %v", err)
	}

	currentPackage := filepath.Join(t.TempDir(), "package")
	for name, content := range files {
		filePath := filepath.Join(currentPackage, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatalf("create current package: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0o755); err != nil {
			t.Fatalf("write current package: %v", err)
		}
	}
	if _, err := registry.ValidatePackageManifest(filepath.Join(currentPackage, "skill.yaml")); err == nil || !strings.Contains(err.Error(), "evidence older than") {
		t.Fatalf("current install admission accepted expired evidence: %v", err)
	}
}

func TestInstallRemoteReleaseUsesSelectedVersionMetadata(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "historical release\n"})
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/registry.json" {
			artifactSum := sha256.Sum256(archive)
			manifestSum := sha256.Sum256([]byte(remoteManifest("1.0.0")))
			version := registry.VersionEntry{
				Version: "1.0.0", ManifestURL: server.URL + "/manifests/1.0.0.yaml",
				ManifestSHA256: hex.EncodeToString(manifestSum[:]), ArtifactURL: server.URL + "/artifacts/1.0.0.tar.gz",
				SHA256: hex.EncodeToString(artifactSum[:]),
			}
			latestMetadata := remoteRegistryEntry("2.0.0", []registry.VersionEntry{version})
			latestMetadata.Runtimes = []string{"codex"}
			latestMetadata.Usability.Availability = "template-only"
			payload, _ := json.Marshal(registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{latestMetadata}})
			_, _ = w.Write(payload)
			return
		}
		if request.URL.Path == "/artifacts/1.0.0.tar.gz" {
			_, _ = w.Write(archive)
			return
		}
		http.NotFound(w, request)
	})
	server.StartTLS()
	defer server.Close()

	result, err := InstallRemoteRelease(context.Background(), remoteOptions(
		server, filepath.Join(t.TempDir(), "cache"), filepath.Join(t.TempDir(), "skills"), "1.0.0",
	))
	if err != nil {
		t.Fatalf("install historical release using selected metadata: %v", err)
	}
	if result.Registry.Usability.Availability != "documentation-only" || !supportsRuntime(result.Registry.Runtimes, "generic") {
		t.Fatalf("result used latest package metadata instead of selected manifest: %#v", result.Registry)
	}
}

func TestInstallRemoteReleaseRejectsManifestDigestMismatch(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), filepath.Join(t.TempDir(), "skills"), "1.0.0")
	// A transport wrapper alters only the registry's manifest digest while the
	// artifact checksum remains valid.
	baseTransport := options.HTTPClient.Transport
	options.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response, err := baseTransport.RoundTrip(request)
		if err != nil || request.URL.Path != "/registry.json" {
			return response, err
		}
		defer response.Body.Close()
		var index registry.Index
		if err := json.NewDecoder(response.Body).Decode(&index); err != nil {
			return nil, err
		}
		index.Skills[0].Versions[0].ManifestSHA256 = strings.Repeat("0", sha256.Size*2)
		data, _ := json.Marshal(index)
		response.Body = io.NopCloser(bytes.NewReader(data))
		response.ContentLength = int64(len(data))
		return response, nil
	})}
	_, err := InstallRemoteRelease(context.Background(), options)
	if err == nil || !strings.Contains(err.Error(), "manifest validation failed") {
		t.Fatalf("expected manifest digest rejection, got %v", err)
	}
}

func TestInstallRemoteReleaseRejectsLegacyRegistryWithoutVersionBinding(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), filepath.Join(t.TempDir(), "skills"), "1.0.0")
	baseTransport := options.HTTPClient.Transport
	options.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response, err := baseTransport.RoundTrip(request)
		if err != nil || request.URL.Path != "/registry.json" {
			return response, err
		}
		defer response.Body.Close()
		var index registry.Index
		if err := json.NewDecoder(response.Body).Decode(&index); err != nil {
			return nil, err
		}
		index.RegistryVersion = "1.2"
		index.Skills[0].Versions[0].ManifestSHA256 = ""
		data, _ := json.Marshal(index)
		response.Body = io.NopCloser(bytes.NewReader(data))
		response.ContentLength = int64(len(data))
		return response, nil
	})}
	_, err := InstallRemoteRelease(context.Background(), options)
	if err == nil || !strings.Contains(err.Error(), "no version-bound manifest digest") {
		t.Fatalf("expected legacy registry rejection, got %v", err)
	}
}

func TestFailedReleasePreservesLastKnownGoodOfflineRegistry(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	var poisoned atomic.Bool
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var overrides map[string]string
		if poisoned.Load() {
			overrides = map[string]string{"1.0.0": strings.Repeat("0", sha256.Size*2)}
		}
		releaseHandler(t, server.URL, map[string][]byte{"1.0.0": archive}, overrides).ServeHTTP(w, request)
	})
	server.StartTLS()

	cacheDir := filepath.Join(t.TempDir(), "cache")
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	options := remoteOptions(server, cacheDir, targetRoot, "1.0.0")
	if _, err := InstallRemoteRelease(context.Background(), options); err != nil {
		t.Fatalf("seed last-known-good registry: %v", err)
	}
	if err := os.RemoveAll(targetRoot); err != nil {
		t.Fatalf("remove initial install: %v", err)
	}

	poisoned.Store(true)
	if _, err := InstallRemoteRelease(context.Background(), options); err == nil || !strings.Contains(err.Error(), "checksum validation failed") {
		t.Fatalf("expected failed candidate release, got %v", err)
	}
	server.Close()
	options.Offline = true
	result, err := InstallRemoteRelease(context.Background(), options)
	if err != nil {
		t.Fatalf("last-known-good offline install failed: %v", err)
	}
	if !result.FromCache {
		t.Fatal("offline recovery did not use the verified artifact cache")
	}
	assertFileContains(t, filepath.Join(result.Destination, "SKILL.md"), "Fixture")
}

func TestInstallRemoteReleaseRejectsTraversalAndSymlinkArchives(t *testing.T) {
	tests := []struct {
		name    string
		archive []byte
	}{
		{name: "parent traversal", archive: makeRawArchive(t, []tar.Header{{Name: "../escape", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg}}, []string{"x"})},
		{name: "symlink", archive: makeRawArchive(t, []tar.Header{{Name: "link", Linkname: "/tmp/escape", Typeflag: tar.TypeSymlink}}, []string{""})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newReleaseServer(t, test.archive, "1.0.0", nil)
			defer server.Close()
			targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
			_, err := InstallRemoteRelease(context.Background(), remoteOptions(server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0"))
			if err == nil || !strings.Contains(err.Error(), "archive validation failed") {
				t.Fatalf("expected unsafe archive rejection, got %v", err)
			}
			assertPathAbsent(t, targetRoot)
		})
	}
}

func TestInstallRemoteReleaseRejectsInterruptedDownload(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	server := newReleaseServerWithWrapper(t, archive, "1.0.0", nil, func(original http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/artifacts/1.0.0.tar.gz" {
				w.Header().Set("Content-Length", "100000")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(archive[:min(16, len(archive))])
				return
			}
			original.ServeHTTP(w, request)
		})
	})
	defer server.Close()
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	_, err := InstallRemoteRelease(context.Background(), remoteOptions(server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0"))
	if err == nil || !strings.Contains(err.Error(), "partial download was discarded") {
		t.Fatalf("expected interrupted download error, got %v", err)
	}
	assertPathAbsent(t, targetRoot)
}

func TestInstallRemoteReleaseRejectsCrossOriginRedirect(t *testing.T) {
	archive := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "# Fixture\n"})
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) }))
	defer other.Close()
	server := newRedirectingReleaseServer(t, other.URL, archive)
	defer server.Close()
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	_, err := InstallRemoteRelease(context.Background(), remoteOptions(server, filepath.Join(t.TempDir(), "cache"), targetRoot, "1.0.0"))
	if err == nil || !strings.Contains(err.Error(), "redirect origin") {
		t.Fatalf("expected redirect-origin rejection, got %v", err)
	}
	assertPathAbsent(t, targetRoot)
}

func TestInstallRemoteReleaseForceUpgradeAndFailedUpgradePreservesPrevious(t *testing.T) {
	archiveV1 := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("1.0.0"), "SKILL.md": "version one\n"})
	archiveV2 := makeArchive(t, map[string]string{"skill.yaml": remoteManifest("2.0.0"), "SKILL.md": "version two\n"})
	archives := map[string][]byte{"1.0.0": archiveV1, "2.0.0": archiveV2}
	server := newMultiVersionServer(t, archives)
	defer server.Close()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")

	if _, err := InstallRemoteRelease(context.Background(), remoteOptions(server, cacheDir, targetRoot, "1.0.0")); err != nil {
		t.Fatalf("install v1: %v", err)
	}
	upgrade := remoteOptions(server, cacheDir, targetRoot, "2.0.0")
	upgrade.Force = true
	if _, err := InstallRemoteRelease(context.Background(), upgrade); err != nil {
		t.Fatalf("upgrade to v2: %v", err)
	}
	destination := filepath.Join(targetRoot, "engineering", "remote-fixture")
	assertFileContains(t, filepath.Join(destination, "SKILL.md"), "version two")

	reinstall := remoteOptions(server, cacheDir, targetRoot, "2.0.0")
	if _, err := InstallRemoteRelease(context.Background(), reinstall); err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("expected explicit-force reinstall rejection, got %v", err)
	}
	reinstall.Force = true
	result, err := InstallRemoteRelease(context.Background(), reinstall)
	if err != nil {
		t.Fatalf("force reinstall v2: %v", err)
	}
	if !result.FromCache {
		t.Fatal("force reinstall did not reuse the verified artifact cache")
	}

	archives["2.0.0"] = makeArchive(t, map[string]string{"skill.yaml": strings.Replace(remoteManifest("2.0.0"), "id: engineering/remote-fixture", "id: engineering/substituted", 1), "SKILL.md": "bad replacement\n"})
	server.CloseClientConnections()
	badServer := newMultiVersionServer(t, map[string][]byte{"1.0.0": archiveV1, "2.0.0": archives["2.0.0"]})
	defer badServer.Close()
	badUpgrade := remoteOptions(badServer, filepath.Join(t.TempDir(), "new-cache"), targetRoot, "2.0.0")
	badUpgrade.Force = true
	if _, err := InstallRemoteRelease(context.Background(), badUpgrade); err == nil || !strings.Contains(err.Error(), "identity validation failed") {
		t.Fatalf("expected substituted identity rejection, got %v", err)
	}
	assertFileContains(t, filepath.Join(destination, "SKILL.md"), "version two")
}

func TestAtomicInstallTreeRecoversCrashAfterBackupRename(t *testing.T) {
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	id := "engineering/recoverable"
	destination := filepath.Join(targetRoot, filepath.FromSlash(id))
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatalf("create active package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "version.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write active package: %v", err)
	}
	newSource := filepath.Join(t.TempDir(), "new")
	if err := os.MkdirAll(newSource, 0o755); err != nil {
		t.Fatalf("create staged source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newSource, "version.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write staged source: %v", err)
	}
	installTransactionFault = func(point string) error {
		if point == "after-backup-rename" {
			return errors.New("simulated process loss")
		}
		return nil
	}
	t.Cleanup(func() { installTransactionFault = nil })
	_, err := atomicInstallTree(newSource, targetRoot, id, true)
	installTransactionFault = nil
	if err == nil || !strings.Contains(err.Error(), "simulated process loss") {
		t.Fatalf("expected simulated interruption, got %v", err)
	}
	assertPathAbsent(t, destination)
	if err := recoverInterruptedInstalls(targetRoot); err != nil {
		t.Fatalf("startup recovery failed: %v", err)
	}
	assertFileContains(t, filepath.Join(destination, "version.txt"), "old")

	_, err = atomicInstallTree(newSource, targetRoot, id, false)
	if err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("expected recovered active package to block non-force install, got %v", err)
	}
	assertFileContains(t, filepath.Join(destination, "version.txt"), "old")
	marker := filepath.Join(filepath.Dir(destination), ".recoverable.skills-hub-transaction.json")
	assertPathAbsent(t, marker)
}

func TestAtomicInstallTreeReconcilesCrashAfterActivationRename(t *testing.T) {
	targetRoot := filepath.Join(t.TempDir(), "runtime", "skills")
	id := "engineering/activated"
	destination := filepath.Join(targetRoot, filepath.FromSlash(id))
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatalf("create active package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "version.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write active package: %v", err)
	}
	newSource := filepath.Join(t.TempDir(), "new")
	if err := os.MkdirAll(newSource, 0o755); err != nil {
		t.Fatalf("create staged source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newSource, "version.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write staged source: %v", err)
	}
	installTransactionFault = func(point string) error {
		if point == "after-activation-rename" {
			return errors.New("simulated process loss")
		}
		return nil
	}
	t.Cleanup(func() { installTransactionFault = nil })
	_, err := atomicInstallTree(newSource, targetRoot, id, true)
	installTransactionFault = nil
	if err == nil || !strings.Contains(err.Error(), "simulated process loss") {
		t.Fatalf("expected simulated interruption, got %v", err)
	}
	assertFileContains(t, filepath.Join(destination, "version.txt"), "new")
	if err := recoverInterruptedInstalls(targetRoot); err != nil {
		t.Fatalf("startup reconciliation failed: %v", err)
	}

	_, err = atomicInstallTree(newSource, targetRoot, id, false)
	if err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("expected reconciled package to block non-force install, got %v", err)
	}
	assertFileContains(t, filepath.Join(destination, "version.txt"), "new")
	marker := filepath.Join(filepath.Dir(destination), ".activated.skills-hub-transaction.json")
	assertPathAbsent(t, marker)
}

func TestInstallRemoteReleasePreservesExecutableMode(t *testing.T) {
	manifest := remoteManifest("1.0.0")
	skillDoc := "# Fixture\n"
	script := "#!/bin/sh\nexit 0\n"
	archive := makeRawArchive(t, []tar.Header{
		{Name: "skill.yaml", Mode: 0o644, Size: int64(len(manifest)), Typeflag: tar.TypeReg},
		{Name: "SKILL.md", Mode: 0o644, Size: int64(len(skillDoc)), Typeflag: tar.TypeReg},
		{Name: "scripts/check.sh", Mode: 0o755, Size: int64(len(script)), Typeflag: tar.TypeReg},
	}, []string{manifest, skillDoc, script})
	server := newReleaseServer(t, archive, "1.0.0", nil)
	defer server.Close()
	result, err := InstallRemoteRelease(context.Background(), remoteOptions(
		server,
		filepath.Join(t.TempDir(), "cache"),
		filepath.Join(t.TempDir(), "skills"),
		"1.0.0",
	))
	if err != nil {
		t.Fatalf("install executable fixture: %v", err)
	}
	info, err := os.Stat(filepath.Join(result.Destination, "scripts", "check.sh"))
	if err != nil {
		t.Fatalf("stat installed script: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("installed script is not executable: mode=%s", info.Mode())
	}
}

func TestInstallRemoteReleaseAcceptsPublishedArchive(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	webRoot := filepath.Join(repositoryRoot, "apps", "web")
	releaseStore := filepath.Join(t.TempDir(), "releases")
	publicRoot := filepath.Join(t.TempDir(), "public")
	runPublisher(t, webRoot, releaseStore, publicRoot)

	const packageID = "engineering/implementation-strategy"
	index, err := registry.LoadIndex(filepath.Join(publicRoot, "registry", "skills-index.json"))
	if err != nil {
		t.Fatalf("load published registry: %v", err)
	}
	entry, found := registry.FindSkill(index, packageID)
	if !found {
		t.Fatalf("published registry missing %s", packageID)
	}
	release, err := registry.ResolveVersion(entry, "latest")
	if err != nil {
		t.Fatalf("resolve published version: %v", err)
	}
	archivePath := filepath.Join(publicRoot, "artifacts", filepath.FromSlash(packageID), release.Version+".tar.gz")
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read published archive: %v", err)
	}
	runPublisher(t, webRoot, releaseStore, publicRoot)
	republishedArchive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read republished archive: %v", err)
	}
	if !bytes.Equal(archive, republishedArchive) {
		t.Fatal("two consecutive publications produced different release bytes")
	}

	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/registry.json":
			servedEntry := entry
			servedEntry.Versions = append([]registry.VersionEntry(nil), entry.Versions...)
			for index := range servedEntry.Versions {
				servedEntry.Versions[index].ArtifactURL = server.URL + "/artifact.tar.gz"
				servedEntry.Versions[index].ManifestURL = server.URL + "/manifest.yaml"
			}
			payload, marshalErr := json.Marshal(registry.Index{RegistryVersion: index.RegistryVersion, Skills: []registry.SkillEntry{servedEntry}})
			if marshalErr != nil {
				t.Errorf("marshal served published registry: %v", marshalErr)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(payload)
		case "/artifact.tar.gz":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, request)
		}
	})
	server.StartTLS()
	defer server.Close()

	result, err := InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "skills",
		Runtime:     "generic",
		ID:          packageID,
		Version:     release.Version,
		TargetRoot:  filepath.Join(t.TempDir(), "skills"),
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("install actual publisher output: %v", err)
	}
	assertFileContains(t, filepath.Join(result.Destination, "SKILL.md"), "Implementation Strategy")
}

func TestPublisherRetainsHistoryAndRejectsVersionMutation(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	webRoot := filepath.Join(repositoryRoot, "apps", "web")
	releaseStore := filepath.Join(t.TempDir(), "releases")
	publicRoot := filepath.Join(t.TempDir(), "public")
	runPublisher(t, webRoot, releaseStore, publicRoot)

	const packageID = "engineering/implementation-strategy"
	index, err := registry.LoadIndex(filepath.Join(publicRoot, "registry", "skills-index.json"))
	if err != nil {
		t.Fatalf("load first publication: %v", err)
	}
	entry, _ := registry.FindSkill(index, packageID)
	current := entry.Versions[0]
	currentDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(packageID), current.Version)
	historicalDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(packageID), "0.0.1")
	if err := os.MkdirAll(historicalDir, 0o755); err != nil {
		t.Fatalf("create historical release: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(currentDir, "skill.yaml"))
	if err != nil {
		t.Fatalf("read current release manifest: %v", err)
	}
	manifest = []byte(strings.Replace(string(manifest), "version: "+current.Version, "version: 0.0.1", 1))
	if err := os.WriteFile(filepath.Join(historicalDir, "skill.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write historical manifest: %v", err)
	}
	currentArchive, err := os.ReadFile(filepath.Join(currentDir, "package.tar.gz"))
	if err != nil {
		t.Fatalf("read current release archive: %v", err)
	}
	archive := rewriteArchiveManifest(t, currentArchive, manifest, nil, nil)
	if err := os.WriteFile(filepath.Join(historicalDir, "package.tar.gz"), archive, 0o644); err != nil {
		t.Fatalf("write historical archive: %v", err)
	}
	projection, err := os.ReadFile(filepath.Join(currentDir, "registry-entry.json"))
	if err != nil {
		t.Fatalf("read current release projection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(historicalDir, "registry-entry.json"), projection, 0o644); err != nil {
		t.Fatalf("write historical release projection: %v", err)
	}
	runPublisher(t, webRoot, releaseStore, publicRoot)
	retained, err := registry.LoadIndex(filepath.Join(publicRoot, "registry", "skills-index.json"))
	if err != nil {
		t.Fatalf("load retained publication: %v", err)
	}
	retainedEntry, _ := registry.FindSkill(retained, packageID)
	if len(retainedEntry.Versions) != 2 {
		t.Fatalf("expected historical and current releases, got %#v", retainedEntry.Versions)
	}
	if _, err := os.Stat(filepath.Join(publicRoot, "artifacts", filepath.FromSlash(packageID), "0.0.1.tar.gz")); err != nil {
		t.Fatalf("historical artifact was not restored: %v", err)
	}

	if err := os.WriteFile(filepath.Join(currentDir, "package.tar.gz"), []byte("mutated"), 0o644); err != nil {
		t.Fatalf("mutate retained release fixture: %v", err)
	}
	output, err := publisherCommand(webRoot, releaseStore, publicRoot).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Immutable release collision") {
		t.Fatalf("expected immutable collision rejection, got %v\n%s", err, output)
	}
}

func TestPublisherAcceptsEquivalentGzipEncoding(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	webRoot := filepath.Join(repositoryRoot, "apps", "web")
	releaseStore := filepath.Join(t.TempDir(), "releases")
	publicRoot := filepath.Join(t.TempDir(), "public")
	runPublisher(t, webRoot, releaseStore, publicRoot)

	archivePath := filepath.Join(
		releaseStore,
		"skills",
		"engineering",
		"implementation-strategy",
		"0.1.0",
		"package.tar.gz",
	)
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read generated release archive: %v", err)
	}
	reencoded := recompressGzip(t, archive, gzip.BestSpeed)
	if bytes.Equal(archive, reencoded) {
		t.Fatal("expected alternate gzip settings to produce different release bytes")
	}
	if err := os.WriteFile(archivePath, reencoded, 0o644); err != nil {
		t.Fatalf("write equivalently encoded release archive: %v", err)
	}

	runPublisher(t, webRoot, releaseStore, publicRoot)
}

func TestPublisherRejectsUnsafeRetainedArchive(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	webRoot := filepath.Join(repositoryRoot, "apps", "web")
	releaseStore := filepath.Join(t.TempDir(), "releases")
	publicRoot := filepath.Join(t.TempDir(), "public")
	runPublisher(t, webRoot, releaseStore, publicRoot)

	const packageID = "engineering/implementation-strategy"
	currentDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(packageID), "0.1.0")
	maliciousDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(packageID), "0.0.1")
	if err := os.MkdirAll(maliciousDir, 0o755); err != nil {
		t.Fatalf("create malicious retained release: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(currentDir, "skill.yaml"))
	if err != nil {
		t.Fatalf("read current manifest: %v", err)
	}
	manifest = []byte(strings.Replace(string(manifest), "version: 0.1.0", "version: 0.0.1", 1))
	if err := os.WriteFile(filepath.Join(maliciousDir, "skill.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write malicious manifest: %v", err)
	}
	projection, err := os.ReadFile(filepath.Join(currentDir, "registry-entry.json"))
	if err != nil {
		t.Fatalf("read current projection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(maliciousDir, "registry-entry.json"), projection, 0o644); err != nil {
		t.Fatalf("write malicious projection: %v", err)
	}
	currentArchive, err := os.ReadFile(filepath.Join(currentDir, "package.tar.gz"))
	if err != nil {
		t.Fatalf("read current archive: %v", err)
	}
	extraHeader := &tar.Header{Name: "../escape", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg}
	archive := rewriteArchiveManifest(t, currentArchive, manifest, extraHeader, []byte("x"))
	if err := os.WriteFile(filepath.Join(maliciousDir, "package.tar.gz"), archive, 0o644); err != nil {
		t.Fatalf("write malicious archive: %v", err)
	}
	output, err := publisherCommand(webRoot, releaseStore, publicRoot).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "unsafe archive path") {
		t.Fatalf("expected retained archive rejection, got %v\n%s", err, output)
	}
}

func TestPublisherKeepsRemovedPackageReachable(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	toolRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	webRoot := filepath.Join(toolRoot, "apps", "web")
	fixtureRoot := t.TempDir()
	publicRoot := filepath.Join(t.TempDir(), "public")
	releaseStore := filepath.Join(t.TempDir(), "releases")
	const packageID = "engineering/implementation-strategy"
	if err := copyTree(
		filepath.Join(toolRoot, "skills", filepath.FromSlash(packageID)),
		filepath.Join(fixtureRoot, "skills", filepath.FromSlash(packageID)),
	); err != nil {
		t.Fatalf("copy package fixture: %v", err)
	}
	fullIndex, err := registry.LoadIndex(filepath.Join(toolRoot, "registry", "skills-index.json"))
	if err != nil {
		t.Fatalf("load source registry: %v", err)
	}
	entry, found := registry.FindSkill(fullIndex, packageID)
	if !found {
		t.Fatalf("source registry missing %s", packageID)
	}
	writePublisherIndexes(t, fixtureRoot, []registry.SkillEntry{entry})
	runPublisherForRepository(t, webRoot, fixtureRoot, toolRoot, releaseStore, publicRoot)

	currentDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(packageID), entry.Latest)
	prereleaseVersion := entry.Latest + "-rc.1"
	prereleaseDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(packageID), prereleaseVersion)
	if err := os.MkdirAll(prereleaseDir, 0o755); err != nil {
		t.Fatalf("create prerelease fixture: %v", err)
	}
	currentManifest, err := os.ReadFile(filepath.Join(currentDir, "skill.yaml"))
	if err != nil {
		t.Fatalf("read stable manifest: %v", err)
	}
	prereleaseManifest := []byte(strings.Replace(string(currentManifest), "version: "+entry.Latest, "version: "+prereleaseVersion, 1))
	currentArchive, err := os.ReadFile(filepath.Join(currentDir, "package.tar.gz"))
	if err != nil {
		t.Fatalf("read stable archive: %v", err)
	}
	prereleaseArchive := rewriteArchiveManifest(t, currentArchive, prereleaseManifest, nil, nil)
	for name, data := range map[string][]byte{
		"skill.yaml":     prereleaseManifest,
		"package.tar.gz": prereleaseArchive,
	} {
		if err := os.WriteFile(filepath.Join(prereleaseDir, name), data, 0o644); err != nil {
			t.Fatalf("write prerelease %s: %v", name, err)
		}
	}
	stableProjection, err := os.ReadFile(filepath.Join(currentDir, "registry-entry.json"))
	if err != nil {
		t.Fatalf("read stable projection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(prereleaseDir, "registry-entry.json"), stableProjection, 0o644); err != nil {
		t.Fatalf("write prerelease projection: %v", err)
	}

	const staleID = "engineering/stale-release"
	staleManifest := strings.Replace(
		remoteUsableNowManifest("1.0.0", time.Now().UTC().Add(-365*24*time.Hour)),
		"id: engineering/remote-fixture",
		"id: "+staleID,
		1,
	)
	writeRetainedSkillRelease(t, releaseStore, staleID, "1.0.0", staleManifest)

	if err := os.RemoveAll(filepath.Join(fixtureRoot, "skills", filepath.FromSlash(packageID))); err != nil {
		t.Fatalf("remove current package source: %v", err)
	}
	writePublisherIndexes(t, fixtureRoot, nil)
	runPublisherForRepository(t, webRoot, fixtureRoot, toolRoot, releaseStore, publicRoot)

	published, err := registry.LoadIndex(filepath.Join(publicRoot, "registry", "skills-index.json"))
	if err != nil {
		t.Fatalf("load rebuilt registry: %v", err)
	}
	retained, found := registry.FindSkill(published, packageID)
	if !found || len(retained.Versions) != 2 {
		t.Fatalf("removed package releases became unreachable: %#v", retained)
	}
	if retained.Latest != entry.Latest {
		t.Fatalf("stable release did not outrank prerelease: latest=%s versions=%#v", retained.Latest, retained.Versions)
	}
	if _, err := os.Stat(filepath.Join(publicRoot, "artifacts", filepath.FromSlash(packageID), retained.Latest+".tar.gz")); err != nil {
		t.Fatalf("retained package artifact is not published: %v", err)
	}
	stale, found := registry.FindSkill(published, staleID)
	if !found {
		t.Fatalf("stale historical release was not published")
	}
	if stale.Usability.Availability != "not-verified" || stale.Usability.Source != "inferred" {
		t.Fatalf("expired historical readiness remained advertised: %#v", stale.Usability)
	}
}

func TestPublisherBuildsSelfContainedPluginClosure(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	toolRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	webRoot := filepath.Join(toolRoot, "apps", "web")
	fixtureRoot := t.TempDir()
	publicRoot := filepath.Join(t.TempDir(), "public")
	releaseStore := filepath.Join(t.TempDir(), "releases")

	const skillID = "engineering/closure-skill"
	skillManifest := strings.Replace(remoteManifest("1.0.0"), "id: engineering/remote-fixture", "id: "+skillID, 1)
	skillDir := filepath.Join(fixtureRoot, "skills", filepath.FromSlash(skillID))
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create closure skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "skill.yaml"), []byte(skillManifest), 0o644); err != nil {
		t.Fatalf("write closure skill manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Closure skill\n"), 0o644); err != nil {
		t.Fatalf("write closure skill: %v", err)
	}

	const pluginID = "engineering/closure-plugin"
	pluginManifest := remotePluginManifest("1.0.0", time.Now().UTC())
	pluginDir := filepath.Join(fixtureRoot, "plugins", filepath.FromSlash(pluginID))
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("create closure plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(pluginManifest), 0o644); err != nil {
		t.Fatalf("write closure plugin manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write closure plugin descriptor: %v", err)
	}

	skill, err := registry.ParseManifest(filepath.Join(skillDir, "skill.yaml"))
	if err != nil {
		t.Fatalf("parse closure skill: %v", err)
	}
	plugin, err := registry.ParseManifest(filepath.Join(pluginDir, "plugin.yaml"))
	if err != nil {
		t.Fatalf("parse closure plugin: %v", err)
	}
	writePublisherModuleIndexes(t, fixtureRoot, map[string][]registry.SkillEntry{
		"index.json":         {sourceEntryForPublisher(skill)},
		"skills-index.json":  {sourceEntryForPublisher(skill)},
		"plugins-index.json": {sourceEntryForPublisher(plugin)},
	})
	runPublisherForRepository(t, webRoot, fixtureRoot, toolRoot, releaseStore, publicRoot)

	archivePath := filepath.Join(publicRoot, "artifacts", filepath.FromSlash(pluginID), "1.0.0.tar.gz")
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read self-contained plugin archive: %v", err)
	}
	entries := archiveEntryNames(t, archive)
	wantManifest := "bundled/skills/engineering/closure-skill/skill.yaml"
	if _, ok := entries[wantManifest]; !ok {
		t.Fatalf("plugin archive omitted declared dependency %s", wantManifest)
	}

	server := newPluginReleaseServer(t, archive, pluginManifest)
	defer server.Close()
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	result, err := InstallRemoteRelease(context.Background(), RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    filepath.Join(t.TempDir(), "cache"),
		Module:      "plugins",
		Runtime:     "generic",
		ID:          pluginID,
		Version:     "1.0.0",
		TargetRoot:  filepath.Join(runtimeRoot, "plugins"),
		HTTPClient:  server.Client(),
	})
	if err != nil {
		t.Fatalf("install publisher-produced plugin archive: %v", err)
	}
	assertFileContains(t, filepath.Join(result.Destination, "plugin.yaml"), pluginID)
	assertFileContains(
		t,
		filepath.Join(runtimeRoot, "skills", filepath.FromSlash(skillID), "SKILL.md"),
		"Closure skill",
	)
}

func sourceEntryForPublisher(manifest registry.Manifest) registry.SkillEntry {
	entry := registry.ProjectManifest(manifest)
	entry.Latest = manifest.Version
	entry.Versions = []registry.VersionEntry{{
		Version: manifest.Version, ReleasedAt: manifest.ReleasedAt,
		ManifestURL:    "https://skills.ai-knowledge-hub.org/manifest.yaml",
		ArtifactURL:    "https://skills.ai-knowledge-hub.org/artifact.tar.gz",
		ManifestSHA256: strings.Repeat("0", sha256.Size*2), SHA256: strings.Repeat("0", sha256.Size*2),
	}}
	return entry
}

func writePublisherModuleIndexes(t *testing.T, root string, entries map[string][]registry.SkillEntry) {
	t.Helper()
	for _, name := range []string{"index.json", "skills-index.json", "agents-index.json", "tools-index.json", "plugins-index.json"} {
		if err := registry.WriteIndex(filepath.Join(root, "registry", name), registry.Index{
			RegistryVersion: "1.3", GeneratedAt: "2026-09-14T00:00:00Z", Skills: entries[name],
		}); err != nil {
			t.Fatalf("write publisher fixture index %s: %v", name, err)
		}
	}
}

func archiveEntryNames(t *testing.T, archive []byte) map[string]struct{} {
	t.Helper()
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("open archive gzip stream: %v", err)
	}
	defer gzipReader.Close()
	entries := make(map[string]struct{})
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read archive entry: %v", err)
		}
		entries[header.Name] = struct{}{}
	}
	return entries
}

func writeRetainedSkillRelease(t *testing.T, releaseStore, id, version, manifest string) {
	t.Helper()
	releaseDir := filepath.Join(releaseStore, "skills", filepath.FromSlash(id), version)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("create retained release: %v", err)
	}
	files := map[string]string{
		"skill.yaml": manifest, "SKILL.md": "# Retained fixture\n", "bin/tool": "fixture\n",
		"checksums.txt": "fixture\n", "sbom.cdx.json": "{}\n",
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "skill.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write retained manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "package.tar.gz"), makeArchive(t, files), 0o644); err != nil {
		t.Fatalf("write retained archive: %v", err)
	}
	parsed, err := registry.ParseManifest(filepath.Join(releaseDir, "skill.yaml"))
	if err != nil {
		t.Fatalf("parse retained manifest: %v", err)
	}
	projection, err := json.MarshalIndent(registry.ProjectManifest(parsed), "", "  ")
	if err != nil {
		t.Fatalf("marshal retained projection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "registry-entry.json"), append(projection, '\n'), 0o644); err != nil {
		t.Fatalf("write retained projection: %v", err)
	}
}

func writePublisherIndexes(t *testing.T, root string, skills []registry.SkillEntry) {
	t.Helper()
	indexes := map[string][]registry.SkillEntry{
		"index.json": skills, "skills-index.json": skills,
		"agents-index.json": nil, "tools-index.json": nil, "plugins-index.json": nil,
	}
	for name, entries := range indexes {
		if err := registry.WriteIndex(filepath.Join(root, "registry", name), registry.Index{
			RegistryVersion: "1.3", GeneratedAt: "2026-09-14T00:00:00Z", Skills: entries,
		}); err != nil {
			t.Fatalf("write publisher index %s: %v", name, err)
		}
	}
}

func runPublisher(t *testing.T, webRoot, releaseStore, publicRoot string) {
	t.Helper()
	command := publisherCommand(webRoot, releaseStore, publicRoot)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("publish web artifacts: %v\n%s", err, output)
	}
}

func publisherCommand(webRoot, releaseStore, publicRoot string) *exec.Cmd {
	command := exec.Command("node", "scripts/prepare-public-assets.mjs")
	command.Dir = webRoot
	command.Env = append(os.Environ(), "RELEASE_STORE_ROOT="+releaseStore, "PUBLIC_ROOT="+publicRoot)
	return command
}

func runPublisherForRepository(t *testing.T, webRoot, repositoryRoot, validatorRoot, releaseStore, publicRoot string) {
	t.Helper()
	command := exec.Command("node", "scripts/prepare-public-assets.mjs")
	command.Dir = webRoot
	command.Env = append(
		os.Environ(),
		"REPOSITORY_ROOT="+repositoryRoot,
		"RELEASE_VALIDATOR_ROOT="+validatorRoot,
		"RELEASE_STORE_ROOT="+releaseStore,
		"PUBLIC_ROOT="+publicRoot,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("publish fixture repository: %v\n%s", err, output)
	}
}

func remoteOptions(server *httptest.Server, cacheDir, targetRoot, version string) RemoteInstallOptions {
	return RemoteInstallOptions{
		RegistryURL: server.URL + "/registry.json",
		CacheDir:    cacheDir,
		Module:      "skills",
		Runtime:     "generic",
		ID:          "engineering/remote-fixture",
		Version:     version,
		TargetRoot:  targetRoot,
		HTTPClient:  server.Client(),
	}
}

func newReleaseServer(t *testing.T, archive []byte, version string, checksumOverride *string) *httptest.Server {
	t.Helper()
	return newReleaseServerWithWrapper(t, archive, version, checksumOverride, nil)
}

func newPluginReleaseServer(t *testing.T, archive []byte, manifest string) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/registry.json":
			artifactSum := sha256.Sum256(archive)
			manifestSum := sha256.Sum256([]byte(manifest))
			entry := registry.SkillEntry{
				SchemaVersion: "2.1",
				ID:            "engineering/closure-plugin",
				Name:          "Closure Plugin",
				Description:   "Self-contained plugin used to prove remote closure installation.",
				Category:      "engineering-plugins/maintenance",
				Latest:        "1.0.0",
				Runtimes:      []string{"generic"},
				Tags:          []string{"remote-install"},
				Usability: registry.UsabilityMetadata{
					Availability: "usable-now",
					Execution:    "bundle",
					Source:       "verified",
				},
				Versions: []registry.VersionEntry{{
					Version:        "1.0.0",
					ManifestURL:    server.URL + "/manifests/1.0.0.yaml",
					ManifestSHA256: hex.EncodeToString(manifestSum[:]),
					ArtifactURL:    server.URL + "/artifacts/1.0.0.tar.gz",
					SHA256:         hex.EncodeToString(artifactSum[:]),
				}},
			}
			payload, err := json.Marshal(registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{entry}})
			if err != nil {
				t.Errorf("marshal plugin registry: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(payload)
		case "/artifacts/1.0.0.tar.gz":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, request)
		}
	})
	server.StartTLS()
	return server
}

func newMultiVersionServer(t *testing.T, archives map[string][]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		releaseHandler(t, server.URL, archives, nil).ServeHTTP(w, request)
	})
	server.StartTLS()
	return server
}

func newReleaseServerWithWrapper(t *testing.T, archive []byte, version string, checksumOverride *string, wrap func(http.Handler) http.Handler) *httptest.Server {
	t.Helper()
	archives := map[string][]byte{version: archive}
	overrides := map[string]string(nil)
	if checksumOverride != nil {
		overrides = map[string]string{version: *checksumOverride}
	}
	server := httptest.NewUnstartedServer(nil)
	base := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		releaseHandler(t, server.URL, archives, overrides).ServeHTTP(w, request)
	})
	var handler http.Handler = base
	if wrap != nil {
		handler = wrap(base)
	}
	server.Config.Handler = handler
	server.StartTLS()
	return server
}

func newRedirectingReleaseServer(t *testing.T, redirectURL string, archive []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		redirectingReleaseHandler(t, server.URL, redirectURL, archive).ServeHTTP(w, request)
	})
	server.StartTLS()
	return server
}

func releaseHandler(t *testing.T, serverURL string, archives map[string][]byte, overrides map[string]string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/registry.json" {
			versions := make([]registry.VersionEntry, 0, len(archives))
			latest := ""
			for version, archive := range archives {
				sum := sha256.Sum256(archive)
				checksum := hex.EncodeToString(sum[:])
				if overrides != nil && overrides[version] != "" {
					checksum = overrides[version]
				}
				manifestBytes := manifestFromArchive(archive)
				if len(manifestBytes) == 0 {
					manifestBytes = []byte(remoteManifest(version))
				}
				manifestSum := sha256.Sum256(manifestBytes)
				versions = append(versions, registry.VersionEntry{
					Version: version, ManifestURL: serverURL + "/manifests/" + version + ".yaml",
					ManifestSHA256: hex.EncodeToString(manifestSum[:]), ArtifactURL: serverURL + "/artifacts/" + version + ".tar.gz", SHA256: checksum,
				})
				if version > latest {
					latest = version
				}
			}
			payload, err := json.Marshal(registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{
				remoteRegistryEntry(latest, versions),
			}})
			if err != nil {
				t.Errorf("marshal registry: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(payload)
			return
		}
		for version, archive := range archives {
			if request.URL.Path == "/artifacts/"+version+".tar.gz" || request.URL.Path == "/artifact.tar.gz" {
				_, _ = w.Write(archive)
				return
			}
		}
		http.NotFound(w, request)
	})
}

func redirectingReleaseHandler(t *testing.T, ownURL, redirectURL string, archive []byte) http.Handler {
	t.Helper()
	sum := sha256.Sum256(archive)
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/registry.json" {
			manifestSum := sha256.Sum256([]byte(remoteManifest("1.0.0")))
			payload, _ := json.Marshal(registry.Index{RegistryVersion: "1.3", Skills: []registry.SkillEntry{
				remoteRegistryEntry("1.0.0", []registry.VersionEntry{{Version: "1.0.0", ManifestURL: ownURL + "/manifests/1.0.0.yaml", ManifestSHA256: hex.EncodeToString(manifestSum[:]), ArtifactURL: ownURL + "/redirect", SHA256: hex.EncodeToString(sum[:])}}),
			}})
			_, _ = w.Write(payload)
			return
		}
		if request.URL.Path == "/redirect" {
			http.Redirect(w, request, redirectURL+"/artifact.tar.gz", http.StatusFound)
			return
		}
		http.NotFound(w, request)
	})
}

func remoteRegistryEntry(latest string, versions []registry.VersionEntry) registry.SkillEntry {
	return registry.SkillEntry{
		SchemaVersion: "1.1",
		ID:            "engineering/remote-fixture",
		Name:          "Remote Fixture",
		Description:   "Remote package fixture used to prove verified installation behavior.",
		Category:      "engineering/testing-quality",
		Latest:        latest,
		Versions:      versions,
		Runtimes:      []string{"generic"},
		Tags:          []string{"remote-install"},
		Readiness:     "experimental",
		Usability: registry.UsabilityMetadata{
			Availability: "documentation-only",
			Execution:    "instructions",
			Source:       "declared",
		},
	}
}

func remoteManifest(version string) string {
	return `schema_version: "1.1"
id: engineering/remote-fixture
name: Remote Fixture
description: Remote package fixture used to prove verified installation behavior.
version: ` + version + `
released_at: "2026-09-14T00:00:00Z"
category: engineering/testing-quality
tags: [remote-install]
license: MIT
author:
  name: Test Maintainer
runtimes: [generic]
entrypoints:
  skill_md: SKILL.md
usability:
  availability: documentation-only
  execution: instructions
deprecated: false
`
}

func remoteUsableNowManifest(version string, verifiedAt time.Time) string {
	return `schema_version: "2.1"
id: engineering/remote-fixture
name: Remote Fixture
description: Remote package fixture used to prove verified installation behavior.
version: ` + version + `
released_at: "2026-09-14T00:00:00Z"
category: engineering/testing-quality
tags: [remote-install]
license: MIT
author:
  name: Test Maintainer
runtimes: [generic]
entrypoints:
  skill_md: SKILL.md
  scripts_dir: bin
usability:
  availability: usable-now
  execution: local-tool
  quickstart: bin/tool
execution:
  kind: cli
  command: [bin/tool]
  smoke_test: [bin/tool, --smoke]
  supported_platforms: [linux]
  supported_runtimes: [native]
artifact:
  self_contained: true
  dependency_lock: null
  checksums: checksums.txt
  sbom: sbom.cdx.json
authentication:
  status: none
  methods: [none]
  credential_bindings: []
  scopes: []
  credential_storage: No credentials.
  validation: Confirm no authentication challenge.
  revocation: Not applicable.
verification:
  evidence: [evidence://tests/remote-fixture/smoke]
  last_verified_at: "` + verifiedAt.Format(time.RFC3339) + `"
deprecated: false
`
}

func remotePluginManifest(version string, verifiedAt time.Time) string {
	return `schema_version: "2.1"
id: engineering/closure-plugin
name: Closure Plugin
description: Self-contained plugin used to prove deterministic dependency packaging.
version: ` + version + `
released_at: "2026-09-14T00:00:00Z"
category: engineering-plugins/maintenance
tags: [remote-install]
license: MIT
author:
  name: Test Maintainer
runtimes: [generic]
entrypoints:
  spec: plugin.json
includes:
  skills: [engineering/closure-skill]
usability:
  availability: usable-now
  execution: bundle
execution:
  kind: bundle
  supported_platforms: [linux, macos, windows]
  supported_runtimes: [generic]
artifact:
  self_contained: true
  dependency_lock: null
  checksums: null
  sbom: null
authentication:
  status: none
  methods: [none]
  credential_bindings: []
  scopes: []
  credential_storage: No credentials.
  validation: Confirm no authentication challenge.
  revocation: Not applicable.
verification:
  evidence: [evidence://tests/closure-plugin]
  last_verified_at: "` + verifiedAt.Format(time.RFC3339) + `"
deprecated: false
`
}

func makeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	headers := make([]tar.Header, 0, len(files))
	contents := make([]string, 0, len(files))
	for name, content := range files {
		headers = append(headers, tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg})
		contents = append(contents, content)
	}
	return makeRawArchive(t, headers, contents)
}

func makeRawArchive(t *testing.T, headers []tar.Header, contents []string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for index := range headers {
		header := headers[index]
		if err := tarWriter.WriteHeader(&header); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if contents[index] != "" {
			if _, err := tarWriter.Write([]byte(contents[index])); err != nil {
				t.Fatalf("write tar content: %v", err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buffer.Bytes()
}

func recompressGzip(t *testing.T, archive []byte, level int) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("open gzip archive: %v", err)
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip payload: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close gzip reader: %v", err)
	}

	var output bytes.Buffer
	writer, err := gzip.NewWriterLevel(&output, level)
	if err != nil {
		t.Fatalf("create gzip writer: %v", err)
	}
	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("write gzip payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return output.Bytes()
}

func rewriteArchiveManifest(t *testing.T, archive, manifest []byte, extraHeader *tar.Header, extraData []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("open source archive: %v", err)
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read source archive: %v", err)
		}
		data, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read source archive member: %v", err)
		}
		copyHeader := *header
		if path.Base(header.Name) == "skill.yaml" {
			data = manifest
			copyHeader.Size = int64(len(data))
		}
		if err := tarWriter.WriteHeader(&copyHeader); err != nil {
			t.Fatalf("write rewritten archive header: %v", err)
		}
		if len(data) > 0 {
			if _, err := tarWriter.Write(data); err != nil {
				t.Fatalf("write rewritten archive member: %v", err)
			}
		}
	}
	if extraHeader != nil {
		if err := tarWriter.WriteHeader(extraHeader); err != nil {
			t.Fatalf("write extra archive header: %v", err)
		}
		if _, err := tarWriter.Write(extraData); err != nil {
			t.Fatalf("write extra archive data: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close rewritten tar archive: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close rewritten gzip archive: %v", err)
	}
	return output.Bytes()
}

func manifestFromArchive(archive []byte) []byte {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			return nil
		}
		if path.Base(header.Name) != "skill.yaml" {
			continue
		}
		data, err := io.ReadAll(tarReader)
		if err != nil {
			return nil
		}
		return data
	}
}

func assertPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path %s to be absent, got %v", path, err)
	}
}

func assertFileContains(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), expected) {
		t.Fatalf("%s does not contain %q", path, expected)
	}
}
