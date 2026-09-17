package readiness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	manifestschemas "github.com/ai-knowledge-hub/ai-skills-guide/shared/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const testSecret = "test-runtime-credential-value"

func TestReadinessHelperProcess(t *testing.T) {
	mode := helperProcessArg("readiness-mode")
	if mode == "" {
		return
	}
	if helperProcessArg("readiness-delay") == "1" {
		time.Sleep(150 * time.Millisecond)
	}
	if os.Getenv("PRIVATE_PROVIDER_KEY") != "" || os.Getenv("UNRELATED_AMBIENT_SECRET") != "" {
		os.Exit(7)
	}
	switch mode {
	case "bootstrap":
		_, _ = os.Stdout.WriteString(`{"completed":true,"account":"acct-1","scopes":["read","profile"]}`)
	case "credential":
		_, _ = os.Stdout.WriteString(`{"value":"` + testSecret + `","generation":"generation-1"}`)
	case "status":
		_, _ = os.Stdout.WriteString(defaultObservationJSON())
	case "validate", "both":
		if os.Getenv("PROVIDER_API_KEY") != testSecret {
			os.Exit(4)
		}
		_, _ = os.Stdout.WriteString(helperProcessArg("readiness-observation"))
	case "smoke":
		if os.Getenv("PROVIDER_API_KEY") != testSecret {
			os.Exit(5)
		}
	default:
		os.Exit(6)
	}
	os.Exit(0)
}

func helperProcessArg(name string) string {
	prefix := "--" + name + "="
	for _, argument := range os.Args {
		if strings.HasPrefix(argument, prefix) {
			return strings.TrimPrefix(argument, prefix)
		}
	}
	return ""
}

func TestConfigureStoresOnlyReferencesAndIncrementsRevision(t *testing.T) {
	root := t.TempDir()
	entry := authenticatedEntry()
	opts := ConfigureOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Method: "api-key",
		Bindings:    map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY"},
		Generations: map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY_GENERATION"},
		Validator:   helperValidator(defaultObservationJSON(), false), Now: time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC),
	}
	first, err := Configure(opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.Bindings["PROVIDER_API_KEY"].Revision != 1 {
		t.Fatalf("first revision = %d", first.Bindings["PROVIDER_API_KEY"].Revision)
	}
	second, err := Configure(opts)
	if err != nil {
		t.Fatal(err)
	}
	if second.Bindings["PROVIDER_API_KEY"].Revision != 2 {
		t.Fatalf("second revision = %d", second.Bindings["PROVIDER_API_KEY"].Revision)
	}
	path, _ := profilePath(root, "tools", entry.ID)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), testSecret) {
		t.Fatal("profile contains a credential payload")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("profile mode = %o", info.Mode().Perm())
	}
}

func TestConfigureRejectsUnsafeBindingAndCredentialInValidator(t *testing.T) {
	entry := authenticatedEntry()
	entry.Authentication.CredentialBindings = []string{"LD_PRELOAD"}
	_, err := Configure(ConfigureOptions{
		StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0", Method: "api-key",
		Bindings: map[string]string{"LD_PRELOAD": "env:PRIVATE_PROVIDER_KEY"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected unsafe binding rejection, got %v", err)
	}

	entry = authenticatedEntry()
	_, err = Configure(ConfigureOptions{
		StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0", Method: "api-key",
		Bindings:    map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY"},
		Generations: map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY_GENERATION"},
		Validator:   &Validator{Command: "provider-check", Args: []string{"ghp_123456789012345678901234567890"}},
	})
	if err == nil || !strings.Contains(err.Error(), "must not contain credentials") {
		t.Fatalf("expected validator credential rejection, got %v", err)
	}
}

func TestConfigureSupportsEveryDeclaredAuthenticationMethod(t *testing.T) {
	methods := []string{
		"api-key", "bearer-token", "oauth-authorization-code-pkce", "oauth-device-flow",
		"oauth-client-credentials", "service-account", "workload-identity", "brokered", "custom",
	}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			entry := authenticatedEntry()
			entry.Authentication.Methods = []string{method}
			opts := ConfigureOptions{
				StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0", Method: method,
				Bindings:    map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY"},
				Generations: map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY_GENERATION"}, Validator: helperValidator(defaultObservationJSON(), false),
			}
			if requiresPackagedAuthDriver(method) {
				packageDir := t.TempDir()
				writeAuthDriverFixture(t, packageDir, method)
				contractDigest, digestErr := installer.RuntimeContractSHA256(entry, "1.0.0")
				if digestErr != nil {
					t.Fatal(digestErr)
				}
				if receiptErr := installer.WriteInstallReceipt(packageDir, installer.InstallReceipt{Source: "local", Module: "tools", ID: entry.ID, Version: "1.0.0", Runtime: "generic", RuntimeContractSHA256: contractDigest}); receiptErr != nil {
					t.Fatal(receiptErr)
				}
				opts.PackageDir, opts.TargetRoot, opts.Runtime, opts.Validator = packageDir, filepath.Dir(packageDir), "generic", nil
				opts.Bindings, opts.Generations = nil, nil
			}
			profile, err := Configure(opts)
			if err != nil {
				t.Fatal(err)
			}
			if profile.Method != method {
				t.Fatalf("method = %q", profile.Method)
			}
			if requiresPackagedAuthDriver(method) {
				report, statusErr := Status(context.Background(), InspectOptions{StateDir: opts.StateDir, Module: "tools", Entry: entry, Version: "1.0.0", Now: time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)})
				if statusErr != nil || !report.Authenticated || report.Account != "acct-1" {
					t.Fatalf("packaged %s status = %#v, %v", method, report, statusErr)
				}
			}
		})
	}
}

func writeAuthDriverFixture(t *testing.T, packageDir, method string) {
	t.Helper()
	command := installHelperExecutable(t, packageDir)
	document := map[string]any{
		"schema_version": "skills-hub.auth-driver/v1", "method": method, "flow": authMethodFlows[method],
		"runtimes":   []string{"generic"},
		"bootstrap":  map[string]any{"command": []string{command, "-test.run=TestReadinessHelperProcess", "--", "--readiness-mode=bootstrap"}},
		"credential": map[string]any{"command": []string{command, "-test.run=TestReadinessHelperProcess", "--", "--readiness-mode=credential"}},
		"status":     map[string]any{"command": []string{command, "-test.run=TestReadinessHelperProcess", "--", "--readiness-mode=status"}},
	}
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(packageDir, "auth", method+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStatusDistinguishesAuthenticationStates(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		observation map[string]any
		account     string
		want        string
		ready       bool
	}{
		{name: "ready", observation: observation(true, false, "acct-1", []string{"read", "profile"}, now.Add(time.Hour)), account: "acct-1", want: "ready", ready: true},
		{name: "expired", observation: observation(true, false, "acct-1", []string{"read", "profile"}, now.Add(-time.Minute)), account: "acct-1", want: "expired"},
		{name: "revoked", observation: observation(true, true, "acct-1", []string{"read", "profile"}, now.Add(time.Hour)), account: "acct-1", want: "revoked"},
		{name: "wrong account", observation: observation(true, false, "acct-2", []string{"read", "profile"}, now.Add(time.Hour)), account: "acct-1", want: "wrong-account"},
		{name: "wrong scope", observation: observation(true, false, "acct-1", []string{"profile"}, now.Add(time.Hour)), account: "acct-1", want: "wrong-scope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			entry := authenticatedEntry()
			payload, _ := json.Marshal(test.observation)
			configureProfile(t, root, entry, test.account, string(payload), false)
			t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
			t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
			t.Setenv("UNRELATED_AMBIENT_SECRET", "must-not-be-inherited")
			report, err := Status(context.Background(), InspectOptions{
				StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Now: now,
			})
			if err != nil {
				t.Fatal(err)
			}
			if report.State != test.want || report.Authenticated != test.ready {
				t.Fatalf("status = %q authenticated=%t, want %q authenticated=%t", report.State, report.Authenticated, test.want, test.ready)
			}
		})
	}
}

func TestStatusDistinguishesMissingAndUnverified(t *testing.T) {
	entry := authenticatedEntry()
	report, err := Status(context.Background(), InspectOptions{StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0"})
	if err != nil || report.State != "missing" {
		t.Fatalf("missing profile report = %#v, %v", report, err)
	}

	root := t.TempDir()
	_, err = Configure(ConfigureOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Method: "api-key",
		Bindings:    map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY"},
		Generations: map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY_GENERATION"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
	report, err = Status(context.Background(), InspectOptions{StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0"})
	if err != nil || report.State != "unverified" || !report.Configured || report.Authenticated {
		t.Fatalf("unverified report = %#v, %v", report, err)
	}
}

func TestOptionalAuthenticationDoesNotSilentlyUseInvalidSelectedProfile(t *testing.T) {
	entry := authenticatedEntry()
	entry.Authentication.Status = "optional"
	report, err := Status(context.Background(), InspectOptions{
		StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0",
	})
	if err != nil || report.State != "optional-not-configured" || report.Selected {
		t.Fatalf("optional unselected report = %#v, %v", report, err)
	}

	root := t.TempDir()
	configureProfile(t, root, entry, "acct-1", defaultObservationJSON(), false)
	report, err = Status(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0",
	})
	if err != nil || report.State != "missing" || !report.Selected || report.Authenticated {
		t.Fatalf("optional selected invalid report = %#v, %v", report, err)
	}
}

func TestStatusRejectsGenerationChangeDuringValidation(t *testing.T) {
	root := t.TempDir()
	entry := authenticatedEntry()
	now := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(observation(true, false, "acct-1", []string{"read", "profile"}, now.Add(time.Hour)))
	configureProfile(t, root, entry, "acct-1", string(payload), true)
	t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
	done := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = os.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-2")
		close(done)
	}()
	report, err := Status(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Now: now,
	})
	<-done
	if err != nil || report.State != "stale" || report.Authenticated {
		t.Fatalf("generation change report = %#v, %v", report, err)
	}
}

func TestSmokeUsesVerifiedCredentialSnapshotAndPersistsRedactedEvidence(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(t.TempDir(), "installed")
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := authenticatedEntry()
	entry.Execution.SmokeTest = []string{installHelperExecutable(t, packageDir), "-test.run=TestReadinessHelperProcess", "--", "--readiness-mode=smoke"}
	contractSHA256, err := installer.RuntimeContractSHA256(entry, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(packageDir, installer.InstallReceipt{
		Source: "remote", Module: "tools", ID: entry.ID, Version: "1.0.0", Runtime: "generic",
		ArtifactSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: contractSHA256,
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(observation(true, false, "acct-1", []string{"read", "profile"}, now.Add(time.Hour)))
	configureProfile(t, root, entry, "acct-1", string(payload), false)
	t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
	t.Setenv("UNRELATED_AMBIENT_SECRET", "must-not-be-inherited")

	evidence, evidencePath, err := Smoke(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Result != "passed" || evidencePath == "" {
		t.Fatalf("evidence = %#v path=%q", evidence, evidencePath)
	}
	stored, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), testSecret) || strings.Contains(string(stored), "PRIVATE_PROVIDER_KEY") {
		t.Fatal("smoke evidence contains credential material or private source reference")
	}
	validateSmokeEvidence(t, stored)
	derived, err := DeriveCatalogStatus(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: evidence.ObservedAt,
	})
	if err != nil || derived.Availability != "usable-now" || derived.EvidencePath != evidencePath {
		t.Fatalf("derived catalog status = %#v, %v", derived, err)
	}
	executablePath := filepath.Join(packageDir, filepath.FromSlash(entry.Execution.SmokeTest[0]))
	executableBytes, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executablePath, []byte("mutated\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	failed, failedPath, persistErr := Smoke(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: evidence.ObservedAt.Add(2 * time.Second),
	})
	if persistErr == nil || failed.FailureReason != "installed-package-integrity-invalid" {
		t.Fatalf("early smoke failure was not recorded: %#v, %v", failed, persistErr)
	}
	if failed.ArtifactSHA256 != "" || failed.ConfigurationClass != "not-evaluated" {
		t.Fatalf("early smoke failure unexpectedly had complete target fields: %#v", failed)
	}
	if err := os.WriteFile(executablePath, executableBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	derived, err = DeriveCatalogStatus(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: failed.ObservedAt.Add(time.Second),
	})
	if err != nil || derived.Availability != "not-verified" || derived.EvidencePath != failedPath {
		t.Fatalf("later failed smoke did not demote catalog status: %#v, %v", derived, err)
	}
	if err := os.Remove(failedPath); err != nil {
		t.Fatal(err)
	}
	invalid := evidence
	invalid.StartedAt = evidence.ObservedAt.Add(3 * time.Second)
	invalid.ObservedAt = evidence.ObservedAt.Add(3 * time.Second)
	invalid.ExpiresAt = invalid.ObservedAt
	invalid.Result = "failed"
	invalid.PromotionEligible = false
	invalid.FailureReason = "smoke-test-failed"
	invalid.Producer = "untrusted-producer"
	invalidPayload, _ := json.Marshal(invalid)
	invalidPath := filepath.Join(root, "evidence", "semantically-invalid.json")
	if err := os.WriteFile(invalidPath, invalidPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	derived, err = DeriveCatalogStatus(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: invalid.ObservedAt.Add(time.Second),
	})
	if err != nil || derived.Availability != "usable-now" || derived.EvidencePath != evidencePath {
		t.Fatalf("semantically invalid evidence participated in chronology: %#v, %v", derived, err)
	}
	if err := os.Remove(invalidPath); err != nil {
		t.Fatal(err)
	}
	var forged SmokeEvidence
	if err := json.Unmarshal(stored, &forged); err != nil {
		t.Fatal(err)
	}
	forged.ExpiresAt = forged.ObservedAt.Add(91 * 24 * time.Hour)
	forgedPayload, _ := json.Marshal(forged)
	if err := os.WriteFile(evidencePath, forgedPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	derived, err = DeriveCatalogStatus(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: evidence.ObservedAt,
	})
	if err != nil || derived.Availability != "not-verified" {
		t.Fatalf("forged evidence status = %#v, %v", derived, err)
	}
}

func TestDoctorReportsScopedSetupEvidence(t *testing.T) {
	root := t.TempDir()
	packageDir := t.TempDir()
	entry := authenticatedEntry()
	entry.Execution.SmokeTest = []string{installHelperExecutable(t, packageDir), "-test.run=TestReadinessHelperProcess", "--", "--readiness-mode=smoke"}
	contractSHA256, err := installer.RuntimeContractSHA256(entry, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(packageDir, installer.InstallReceipt{
		Source: "remote", Module: "tools", ID: entry.ID, Version: "1.0.0", Runtime: "generic",
		ArtifactSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: contractSHA256,
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(observation(true, false, "acct-1", []string{"read", "profile"}, now.Add(time.Hour)))
	configureProfile(t, root, entry, "acct-1", string(payload), false)
	t.Setenv("PRIVATE_PROVIDER_KEY", testSecret)
	t.Setenv("PRIVATE_PROVIDER_KEY_GENERATION", "generation-1")
	report, err := Doctor(context.Background(), InspectOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready || report.SetupRequired || !report.Authentication.Authenticated || report.EvidenceScope != "setup" || report.EnvironmentID == "" {
		t.Fatalf("doctor report = %#v", report)
	}
	entries, err := os.ReadDir(filepath.Join(root, "doctor-evidence"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("doctor evidence was not persisted: entries=%v err=%v", entries, err)
	}
	otherTarget := report
	otherTarget.Runtime = "codex"
	if err := persistDoctorEvidence(root, otherTarget); err != nil {
		t.Fatal(err)
	}
	entries, err = os.ReadDir(filepath.Join(root, "doctor-evidence"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("doctor evidence targets overwrote one another: entries=%v err=%v", entries, err)
	}
	doctorPayload, err := os.ReadFile(filepath.Join(root, "doctor-evidence", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	validateEvidenceSchema(t, "doctor-evidence.schema.json", doctorPayload)
}

func TestFailedSmokeEvidenceIsSchemaValidAndNotPromotionEligible(t *testing.T) {
	entry := authenticatedEntry()
	entry.Execution = nil
	evidence, path, err := Smoke(context.Background(), InspectOptions{
		StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0", Runtime: "generic",
	})
	if err == nil || evidence.FailureReason != "smoke-test-not-declared" || evidence.PromotionEligible {
		t.Fatalf("failed evidence = %#v, %v", evidence, err)
	}
	payload, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	validateSmokeEvidence(t, payload)
}

func TestSmokeEvidenceContractRejectsSemanticMutations(t *testing.T) {
	evidence := SmokeEvidence{
		SchemaVersion: evidenceSchema, ContractID: "operational-readiness", ContractVersion: "1.0.3",
		EvidenceScope: "executable", TestID: "skills-hub/declared-smoke", TestVersion: "1", Producer: "skills-hub",
		EntryID: "ads/example", EntryVersion: "1.0.0", ArtifactSHA256: strings.Repeat("a", 64), DependencyLockSHA256: "not-applicable",
		InstallSource: "remote", Module: "tools", Runtime: "generic", Platform: currentPlatform(), Command: []string{"bin/example", "--smoke"},
		EnvironmentID: strings.Repeat("b", 32), ConfigurationClass: "no-authentication",
		Coverage:  []string{"install-receipt", "installed-tree", "declared-smoke-test"},
		StartedAt: time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC), ObservedAt: time.Date(2026, 9, 17, 10, 0, 1, 0, time.UTC),
		ExpiresAt: time.Date(2026, 9, 17, 11, 0, 0, 0, time.UTC), Result: "failed", PromotionEligible: false, FailureReason: "smoke-test-failed",
	}
	mutations := map[string]func(*SmokeEvidence){
		"contract version": func(value *SmokeEvidence) { value.ContractVersion = "9" },
		"evidence scope":   func(value *SmokeEvidence) { value.EvidenceScope = "setup" },
		"test identity":    func(value *SmokeEvidence) { value.TestID = "other" },
		"test version":     func(value *SmokeEvidence) { value.TestVersion = "2" },
		"producer":         func(value *SmokeEvidence) { value.Producer = "other" },
		"coverage":         func(value *SmokeEvidence) { value.Coverage = []string{"installed-tree"} },
		"failure reason":   func(value *SmokeEvidence) { value.FailureReason = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := evidence
			mutate(&candidate)
			payload, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateSmokeEvidencePayload(payload); err == nil {
				t.Fatalf("semantic mutation %q satisfied the smoke evidence contract", name)
			}
		})
	}
}

func TestAtomicWriteNewPublishesOnlyCompleteNoReplaceFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evidence.json")
	payload := []byte("{\"complete\":true}\n")
	injected := errors.New("crash-before-publish")
	atomicWriteNewBeforePublish = func(finalPath, temporaryPath string) error {
		if finalPath != path {
			t.Fatalf("unexpected final path %s", finalPath)
		}
		if _, err := os.Stat(finalPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("final path became visible before publication: %v", err)
		}
		stored, err := os.ReadFile(temporaryPath)
		if err != nil || !bytes.Equal(stored, payload) {
			t.Fatalf("temporary evidence was not complete before publication: %q, %v", stored, err)
		}
		return injected
	}
	if err := atomicWriteNew(path, payload, 0o600); !errors.Is(err, injected) {
		t.Fatalf("pre-publication fault was not returned: %v", err)
	}
	atomicWriteNewBeforePublish = nil
	t.Cleanup(func() { atomicWriteNewBeforePublish = nil })
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed publication left a final evidence path: %v", err)
	}
	if err := atomicWriteNew(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteNew(path, []byte("replacement\n"), 0o600); !errors.Is(err, os.ErrExist) {
		t.Fatalf("append-only evidence was replaceable: %v", err)
	}
	stored, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(stored, payload) {
		t.Fatalf("published evidence is incomplete or replaced: %q, %v", stored, err)
	}
}

func TestSmokeRejectsRegistryRuntimeContractDrift(t *testing.T) {
	packageDir := t.TempDir()
	entry := authenticatedEntry()
	contractSHA256, err := installer.RuntimeContractSHA256(entry, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.WriteInstallReceipt(packageDir, installer.InstallReceipt{
		Source: "remote", Module: "tools", ID: entry.ID, Version: "1.0.0", Runtime: "generic",
		ArtifactSHA256: strings.Repeat("a", 64), RuntimeContractSHA256: contractSHA256,
	}); err != nil {
		t.Fatal(err)
	}
	entry.Execution.SmokeTest = []string{"different-command", "--smoke"}
	evidence, _, err := Smoke(context.Background(), InspectOptions{
		StateDir: t.TempDir(), Module: "tools", Entry: entry, Version: "1.0.0", PackageDir: packageDir, Runtime: "generic",
	})
	if err == nil || evidence.FailureReason != "installed-package-integrity-invalid" {
		t.Fatalf("runtime contract drift evidence = %#v, %v", evidence, err)
	}
}

func authenticatedEntry() registry.SkillEntry {
	return registry.SkillEntry{
		ID: "ads/example-tool", Latest: "1.0.0", Runtimes: []string{"generic", "codex", "claude"},
		Versions: []registry.VersionEntry{{Version: "1.0.0", SHA256: strings.Repeat("a", 64)}},
		Execution: &registry.ExecutionMetadata{
			Kind: "cli", SmokeTest: []string{"example", "--smoke"}, SupportedPlatforms: []string{currentPlatform()}, SupportedRuntimes: []string{"native"},
		},
		Artifact: &registry.ArtifactMetadata{SelfContained: true},
		Authentication: &registry.AuthenticationMetadata{
			Status: "required", Methods: []string{"api-key"}, CredentialBindings: []string{"PROVIDER_API_KEY"}, Scopes: []string{"read", "profile"},
		},
	}
}

func helperValidator(observation string, delay bool) *Validator {
	args := []string{"-test.run=TestReadinessHelperProcess", "--", "--readiness-mode=validate", "--readiness-observation=" + observation}
	if delay {
		args = append(args, "--readiness-delay=1")
	}
	return &Validator{Command: os.Args[0], Args: args}
}

func configureProfile(t *testing.T, root string, entry registry.SkillEntry, account, validatorObservation string, delay bool) {
	t.Helper()
	_, err := Configure(ConfigureOptions{
		StateDir: root, Module: "tools", Entry: entry, Version: "1.0.0", Method: "api-key", ExpectedAccount: account,
		Bindings:    map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY"},
		Generations: map[string]string{"PROVIDER_API_KEY": "env:PRIVATE_PROVIDER_KEY_GENERATION"}, Validator: helperValidator(validatorObservation, delay),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func defaultObservationJSON() string {
	payload, _ := json.Marshal(observation(true, false, "acct-1", []string{"read", "profile"}, time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)))
	return string(payload)
}

func observation(authenticated, revoked bool, account string, scopes []string, expires time.Time) map[string]any {
	return map[string]any{
		"authenticated": authenticated, "revoked": revoked, "principal": "provider-user-1",
		"account": account, "scopes": scopes, "expires_at": expires.UTC(),
		"provider": "example-provider", "tier": "sandbox", "endpoint": "https://api.example.test/v1",
		"region": "global", "api_version": "v1", "attestation_reference": "provider://identity/check-1",
		"attestation_expires_at": expires.Add(48 * time.Hour).UTC(),
	}
}

func validateSmokeEvidence(t *testing.T, payload []byte) {
	t.Helper()
	validateEvidenceSchema(t, "smoke-evidence.schema.json", payload)
}

func validateEvidenceSchema(t *testing.T, schemaName string, payload []byte) {
	t.Helper()
	schemaPayload, err := manifestschemas.ManifestFiles.ReadFile(schemaName)
	if err != nil {
		t.Fatal(err)
	}
	var schemaDocument any
	if err := json.Unmarshal(schemaPayload, &schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	schemaURL := "https://skills.ai-knowledge-hub.org/schemas/" + schemaName
	if err := compiler.AddResource(schemaURL, schemaDocument); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(document); err != nil {
		t.Fatalf("persisted smoke evidence is invalid: %v", err)
	}
}

func installHelperExecutable(t *testing.T, packageDir string) string {
	t.Helper()
	relative := filepath.Join("bin", "readiness-test-helper")
	destination := filepath.Join(packageDir, relative)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	target, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(target, source); err != nil {
		target.Close()
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(relative)
}
