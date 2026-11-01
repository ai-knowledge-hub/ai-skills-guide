package installer

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type operationClock struct{ nanos atomic.Int64 }

func (clock *operationClock) set(at time.Time) { clock.nanos.Store(at.UnixNano()) }
func (clock *operationClock) now() time.Time   { return time.Unix(0, clock.nanos.Load()).UTC() }

func timedSkillFiles(observed time.Time) map[string]string {
	manifest := strings.Replace(remoteUsableNowManifest("1.0.0", observed), "supported_platforms: [linux]", "supported_platforms: [linux, macos, windows]", 1)
	return map[string]string{
		"skill.yaml": manifest, "SKILL.md": "# New timed package\n", "bin/tool": "fixture\n",
		"checksums.txt": "fixture\n", "sbom.cdx.json": "{}\n",
	}
}

func TestRemoteAdmissionSamplesTimeAfterDownloads(t *testing.T) {
	observed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := observed.Add(90 * 24 * time.Hour) // Operational v1 executable scope.
	archive := makeArchive(t, timedSkillFiles(observed))
	for _, download := range []string{"/registry.json", "/artifacts/1.0.0.tar.gz"} {
		for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
			t.Run(download+"/"+delta.String(), func(t *testing.T) {
				var clock operationClock
				clock.set(deadline.Add(-time.Hour))
				evaluation := evaluationWithClock(time.Time{}, clock.now)
				var artifactSeen atomic.Bool
				server := newReleaseServerWithWrapper(t, archive, "1.0.0", nil, func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if strings.HasPrefix(r.URL.Path, "/artifacts/") {
							artifactSeen.Store(true)
						}
						if r.URL.Path == download {
							clock.set(deadline.Add(delta))
						}
						next.ServeHTTP(w, r)
					})
				})
				defer server.Close()
				options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), filepath.Join(t.TempDir(), "runtime", "skills"), "1.0.0")
				options.ExecutionRuntimes = []string{"native"}
				result, err := evaluation.InstallRemoteRelease(context.Background(), options)
				if !artifactSeen.Load() {
					t.Fatal("fixture did not exercise both registry and artifact downloads")
				}
				if delta < 0 {
					if err != nil {
						t.Fatalf("unexpired control rejected: %v", err)
					}
					if _, err := registry.ValidatePackageManifestAt(filepath.Join(result.Destination, "skill.yaml"), clock.now()); err != nil {
						t.Fatalf("installed control is not admissible: %v", err)
					}
				} else {
					if err == nil || !strings.Contains(err.Error(), "evidence older than") {
						t.Fatalf("download extended evidence validity: %v", err)
					}
					assertPathAbsent(t, options.TargetRoot)
				}
			})
		}
	}
}

func TestFixedHistoricalEvaluationSurvivesAdvancingTransportClock(t *testing.T) {
	observed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var clock operationClock
	clock.set(observed.Add(time.Hour))
	evaluation := evaluationWithClock(observed.Add(time.Hour), clock.now)
	server := newReleaseServerWithWrapper(t, makeArchive(t, timedSkillFiles(observed)), "1.0.0", nil, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clock.set(observed.AddDate(100, 0, 0))
			next.ServeHTTP(w, r)
		})
	})
	defer server.Close()
	options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), filepath.Join(t.TempDir(), "skills"), "1.0.0")
	options.ExecutionRuntimes = []string{"native"}
	if _, err := evaluation.InstallRemoteRelease(context.Background(), options); err != nil {
		t.Fatalf("explicit historical instant was lost: %v", err)
	}
}

func TestRemoteCommitRechecksFreshnessAndRestoresPreviousInstallation(t *testing.T) {
	observed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := observed.Add(90 * 24 * time.Hour)
	var clock operationClock
	clock.set(deadline.Add(-time.Hour))
	server := newReleaseServer(t, makeArchive(t, timedSkillFiles(observed)), "1.0.0", nil)
	defer server.Close()
	options := remoteOptions(server, filepath.Join(t.TempDir(), "cache"), filepath.Join(t.TempDir(), "runtime", "skills"), "1.0.0")
	options.ExecutionRuntimes, options.Force = []string{"native"}, true
	destination := filepath.Join(options.TargetRoot, "engineering", "remote-fixture")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "previous.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	previous := installTransactionFault
	t.Cleanup(func() { installTransactionFault = previous })
	advanced := false
	installTransactionFault = func(point string) error {
		if point == "after-backup-rename" {
			advanced = true
			clock.set(deadline)
		}
		return nil
	}
	_, err := evaluationWithClock(time.Time{}, clock.now).InstallRemoteRelease(context.Background(), options)
	if !advanced || err == nil || !strings.Contains(err.Error(), "evidence older than") {
		t.Fatalf("commit extended evidence lifetime: advanced=%v, err=%v", advanced, err)
	}
	assertFileContains(t, filepath.Join(destination, "previous.txt"), "original")
	assertPathAbsent(t, filepath.Join(destination, "SKILL.md"))
	assertPathAbsent(t, filepath.Join(filepath.Dir(destination), ".remote-fixture.skills-hub-transaction.json"))
}

func TestRemoteClosureExpiryRollsBackActivatedDependencies(t *testing.T) {
	childObserved := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deadline := childObserved.Add(90 * 24 * time.Hour)
	rootManifest := remotePluginManifest("1.0.0", childObserved.Add(24*time.Hour))
	files := map[string]string{"plugin.yaml": rootManifest, "plugin.json": "{}\n"}
	for name, content := range timedSkillFiles(childObserved) {
		files["bundled/skills/engineering/closure-skill/"+name] = strings.ReplaceAll(content, "engineering/remote-fixture", "engineering/closure-skill")
	}
	archive := makeArchive(t, files)
	for _, point := range []string{"before-activation", "after-activation"} {
		t.Run(point, func(t *testing.T) {
			var clock operationClock
			clock.set(deadline.Add(-time.Hour))
			server := newPluginReleaseServer(t, archive, rootManifest)
			defer server.Close()
			runtimeRoot := t.TempDir()
			dependency := filepath.Join(runtimeRoot, "skills", "engineering", "closure-skill")
			if err := os.MkdirAll(dependency, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dependency, "previous.txt"), []byte("original dependency"), 0o644); err != nil {
				t.Fatal(err)
			}
			previous := closureInstallTransactionFault
			t.Cleanup(func() { closureInstallTransactionFault = previous })
			advanced := false
			closureInstallTransactionFault = func(phase string, index int) error {
				if phase == point && index == 0 {
					advanced = true
					clock.set(deadline)
				}
				return nil
			}
			_, err := evaluationWithClock(time.Time{}, clock.now).InstallRemoteRelease(context.Background(), RemoteInstallOptions{
				RegistryURL: server.URL + "/registry.json", CacheDir: filepath.Join(t.TempDir(), "cache"), Module: "plugins", Runtime: "generic",
				ID: "engineering/closure-plugin", Version: "1.0.0", TargetRoot: filepath.Join(runtimeRoot, "plugins"), HTTPClient: server.Client(),
				ExecutionRuntimes: []string{"generic", "native"}, Force: true,
			})
			if !advanced || err == nil || !strings.Contains(err.Error(), "engineering/closure-skill") || !strings.Contains(err.Error(), "evidence older than") {
				t.Fatalf("child evidence outlived its deadline: advanced=%v, err=%v", advanced, err)
			}
			assertPathAbsent(t, filepath.Join(runtimeRoot, "plugins", "engineering", "closure-plugin"))
			assertFileContains(t, filepath.Join(dependency, "previous.txt"), "original dependency")
			assertPathAbsent(t, filepath.Join(dependency, "SKILL.md"))
			assertPathAbsent(t, closureTransactionMarker(runtimeRoot, "engineering/closure-plugin"))
		})
	}
}
