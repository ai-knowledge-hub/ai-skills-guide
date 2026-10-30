package readiness

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	manifestschemas "github.com/ai-knowledge-hub/ai-skills-guide/shared/schemas"
	"github.com/gofrs/flock"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const profileSchema = "skills-hub.auth-profile/v1"
const evidenceSchema = "skills-hub.smoke-evidence/v1"

var supportedMethods = map[string]struct{}{
	"api-key": {}, "bearer-token": {}, "oauth-authorization-code-pkce": {},
	"oauth-device-flow": {}, "oauth-client-credentials": {}, "service-account": {},
	"workload-identity": {}, "brokered": {}, "custom": {},
}

var authMethodFlows = map[string]string{
	"api-key": "credential-binding", "bearer-token": "credential-binding",
	"oauth-authorization-code-pkce": "interactive-browser", "oauth-device-flow": "device-code",
	"oauth-client-credentials": "non-interactive-service", "service-account": "non-interactive-service",
	"workload-identity": "non-interactive-workload", "brokered": "brokered", "custom": "custom",
}

var (
	smokeEvidenceSchemaOnce     sync.Once
	smokeEvidenceContract       *jsonschema.Schema
	smokeEvidenceSchemaErr      error
	atomicWriteNewBeforePublish func(string, string) error
)

type authDriverDocument struct {
	SchemaVersion  string               `json:"schema_version"`
	CredentialMode string               `json:"credential_mode"`
	Method         string               `json:"method"`
	Flow           string               `json:"flow"`
	Runtimes       []string             `json:"runtimes"`
	Bootstrap      authDriverCommandDoc `json:"bootstrap"`
	Credential     authDriverCommandDoc `json:"credential"`
	Status         authDriverCommandDoc `json:"status"`
}

type authDriverCommandDoc struct {
	Command []string `json:"command"`
}

type bootstrapRequest struct {
	Method             string   `json:"method"`
	Flow               string   `json:"flow"`
	SetupURL           string   `json:"setup_url,omitempty"`
	RequestedScopes    []string `json:"requested_scopes"`
	CredentialBindings []string `json:"credential_bindings"`
	ExpectedAccount    string   `json:"expected_account,omitempty"`
}

type bootstrapResponse struct {
	Completed bool     `json:"completed"`
	Account   string   `json:"account"`
	Scopes    []string `json:"scopes"`
}

type credentialResponse struct {
	Value      string `json:"value"`
	Generation string `json:"generation"`
}

// BindingSource names a runtime-owned credential without containing it.
type BindingSource struct {
	Type                string `json:"type"`
	Reference           string `json:"reference"`
	GenerationReference string `json:"generation_reference"`
	Revision            uint64 `json:"revision"`
}

type Validator struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	SHA256  string   `json:"sha256"`
}

type DriverCommand struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	SHA256  string   `json:"sha256"`
}

type AuthDriver struct {
	SchemaVersion  string        `json:"schema_version"`
	CredentialMode string        `json:"credential_mode,omitempty"`
	Method         string        `json:"method"`
	Flow           string        `json:"flow"`
	PackageDir     string        `json:"package_dir"`
	TargetRoot     string        `json:"target_root"`
	Runtime        string        `json:"runtime"`
	Bootstrap      DriverCommand `json:"bootstrap"`
	Credential     DriverCommand `json:"credential"`
	Status         DriverCommand `json:"status"`
}

// Profile contains only non-secret configuration. Credential payloads remain
// in the runtime-owned source named by each binding.
type Profile struct {
	SchemaVersion     string                   `json:"schema_version"`
	Module            string                   `json:"module"`
	EntryID           string                   `json:"entry_id"`
	EntryVersion      string                   `json:"entry_version"`
	Method            string                   `json:"method"`
	ExpectedAccount   string                   `json:"expected_account,omitempty"`
	Bindings          map[string]BindingSource `json:"bindings"`
	Validator         *Validator               `json:"validator,omitempty"`
	Driver            *AuthDriver              `json:"driver,omitempty"`
	RequestedScopes   []string                 `json:"requested_scopes,omitempty"`
	BootstrapMode     string                   `json:"bootstrap_mode"`
	BootstrapComplete bool                     `json:"bootstrap_complete"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

type ConfigureOptions struct {
	StateDir        string
	Module          string
	Entry           registry.SkillEntry
	Version         string
	Method          string
	ExpectedAccount string
	Bindings        map[string]string
	Generations     map[string]string
	Validator       *Validator
	PackageDir      string
	TargetRoot      string
	Runtime         string
	Now             time.Time
}

type AuthReport struct {
	EntryID                   string    `json:"entry_id"`
	EntryVersion              string    `json:"entry_version"`
	Required                  bool      `json:"required"`
	Selected                  bool      `json:"selected"`
	Configured                bool      `json:"configured"`
	Authenticated             bool      `json:"authenticated"`
	State                     string    `json:"state"`
	Method                    string    `json:"method,omitempty"`
	Principal                 string    `json:"principal,omitempty"`
	Account                   string    `json:"account,omitempty"`
	RequiredScopes            []string  `json:"required_scopes,omitempty"`
	GrantedScopes             []string  `json:"granted_scopes,omitempty"`
	MissingBindings           []string  `json:"missing_bindings,omitempty"`
	MissingScopes             []string  `json:"missing_scopes,omitempty"`
	ExpiresAt                 time.Time `json:"expires_at,omitempty"`
	ProviderTargetFingerprint string    `json:"provider_target_fingerprint,omitempty"`
	AttestationReference      string    `json:"attestation_reference,omitempty"`
	Reason                    string    `json:"reason,omitempty"`
	ObservedAt                time.Time `json:"observed_at"`
}

type validatorObservation struct {
	Authenticated        bool       `json:"authenticated"`
	Revoked              bool       `json:"revoked"`
	Principal            string     `json:"principal"`
	Account              string     `json:"account"`
	Scopes               []string   `json:"scopes"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	Provider             string     `json:"provider"`
	Tier                 string     `json:"tier"`
	Endpoint             string     `json:"endpoint"`
	Region               string     `json:"region"`
	APIVersion           string     `json:"api_version"`
	AttestationReference string     `json:"attestation_reference"`
	AttestationExpiresAt *time.Time `json:"attestation_expires_at"`
}

type resolvedAuth struct {
	Report               AuthReport
	Env                  []string
	Keys                 map[string]string
	Generations          map[string]string
	GenerationReferences map[string]string
	ValidatorSHA256      string
	Driver               *AuthDriver
}

type DoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type DoctorReport struct {
	SchemaVersion         string            `json:"schema_version"`
	ContractID            string            `json:"contract_id"`
	ContractVersion       string            `json:"contract_version"`
	EvidenceScope         string            `json:"evidence_scope"`
	TestID                string            `json:"test_id"`
	TestVersion           string            `json:"test_version"`
	Producer              string            `json:"producer"`
	EntryID               string            `json:"entry_id"`
	EntryVersion          string            `json:"entry_version"`
	Module                string            `json:"module"`
	Runtime               string            `json:"runtime"`
	Platform              string            `json:"platform"`
	ArtifactSHA256        string            `json:"artifact_sha256"`
	RuntimeContractSHA256 string            `json:"runtime_contract_sha256"`
	DependencyLockSHA256  string            `json:"dependency_lock_sha256"`
	InstallSource         string            `json:"install_source"`
	EnvironmentID         string            `json:"environment_id"`
	ObservedAt            time.Time         `json:"observed_at"`
	ExpiresAt             time.Time         `json:"expires_at"`
	Ready                 bool              `json:"ready"`
	SetupRequired         bool              `json:"setup_required"`
	Authentication        AuthReport        `json:"authentication"`
	CredentialKeys        map[string]string `json:"credential_binding_keys,omitempty"`
	ConfigurationClass    string            `json:"configuration_class"`
	Result                string            `json:"result"`
	Checks                []DoctorCheck     `json:"checks"`
}

type CatalogStatus struct {
	Availability  string    `json:"availability"`
	EvidenceScope string    `json:"evidence_scope,omitempty"`
	EvidencePath  string    `json:"evidence_path,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

type SmokeEvidence struct {
	SchemaVersion               string            `json:"schema_version"`
	ContractID                  string            `json:"contract_id"`
	ContractVersion             string            `json:"contract_version"`
	EvidenceScope               string            `json:"evidence_scope"`
	TestID                      string            `json:"test_id"`
	TestVersion                 string            `json:"test_version"`
	Producer                    string            `json:"producer"`
	EntryID                     string            `json:"entry_id"`
	EntryVersion                string            `json:"entry_version"`
	ArtifactSHA256              string            `json:"artifact_sha256"`
	DependencyLockSHA256        string            `json:"dependency_lock_sha256"`
	InstallSource               string            `json:"install_source"`
	Module                      string            `json:"module"`
	Runtime                     string            `json:"runtime"`
	Platform                    string            `json:"platform"`
	Command                     []string          `json:"command"`
	CredentialKeys              map[string]string `json:"credential_binding_keys,omitempty"`
	ProviderIdentity            string            `json:"provider_principal,omitempty"`
	ProviderAccount             string            `json:"provider_account,omitempty"`
	ProviderTargetFingerprint   string            `json:"provider_target_fingerprint,omitempty"`
	AttestationReference        string            `json:"attestation_reference,omitempty"`
	PublicationClass            string            `json:"publication_class,omitempty"`
	RedactionReason             string            `json:"redaction_reason,omitempty"`
	RedactedFields              []string          `json:"redacted_fields,omitempty"`
	GrantedScopes               []string          `json:"granted_scopes,omitempty"`
	ValidatorSHA256             string            `json:"validator_sha256,omitempty"`
	ExecutionRuntimeFingerprint string            `json:"execution_runtime_fingerprint,omitempty"`
	EnvironmentID               string            `json:"environment_id"`
	ConfigurationClass          string            `json:"configuration_class"`
	Coverage                    []string          `json:"coverage"`
	StartedAt                   time.Time         `json:"started_at"`
	ObservedAt                  time.Time         `json:"observed_at"`
	ExpiresAt                   time.Time         `json:"expires_at"`
	Result                      string            `json:"result"`
	PromotionEligible           bool              `json:"promotion_eligible"`
	FailureReason               string            `json:"failure_reason,omitempty"`
}

type InspectOptions struct {
	StateDir   string
	Module     string
	Entry      registry.SkillEntry
	Version    string
	PackageDir string
	TargetRoot string
	Runtime    string
	Now        time.Time
	Clock      func() time.Time
}

func Configure(opts ConfigureOptions) (Profile, error) {
	auth, err := authentication(opts.Entry)
	if err != nil {
		return Profile{}, err
	}
	if auth.Status == "none" {
		return Profile{}, fmt.Errorf("%s does not require authentication", opts.Entry.ID)
	}
	method := strings.TrimSpace(opts.Method)
	if method == "" && len(auth.Methods) == 1 {
		method = auth.Methods[0]
	}
	if !contains(auth.Methods, method) {
		return Profile{}, fmt.Errorf("authentication method %q is not declared for %s", method, opts.Entry.ID)
	}
	if _, ok := supportedMethods[method]; !ok {
		return Profile{}, fmt.Errorf("authentication method %q is not supported by this client", method)
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	profile := Profile{
		SchemaVersion: profileSchema, Module: opts.Module, EntryID: opts.Entry.ID,
		EntryVersion: opts.Version, Method: method, ExpectedAccount: strings.TrimSpace(opts.ExpectedAccount),
		Bindings: make(map[string]BindingSource), Validator: opts.Validator, RequestedScopes: append([]string(nil), auth.Scopes...),
		BootstrapMode: authMethodFlows[method], UpdatedAt: opts.Now.UTC(),
	}
	if profile.ExpectedAccount != "" && !safeObservationValue(profile.ExpectedAccount) {
		return Profile{}, errors.New("expected account is invalid or contains credentials")
	}
	usePackagedDriver := requiresPackagedAuthDriver(method) || authDriverExists(opts.PackageDir, method)
	if usePackagedDriver {
		if profile.Validator != nil {
			return Profile{}, errors.New("packaged authentication flows use their receipt-bound status driver, not --validator-command")
		}
		driver, err := loadAuthDriver(opts, method)
		if err != nil {
			return Profile{}, err
		}
		if driver.CredentialMode == "external-env" {
			if err := configureExternalBindings(&profile, auth.CredentialBindings, opts.Bindings, opts.Generations); err != nil {
				return Profile{}, err
			}
		}
		bootstrapEnv, err := explicitBootstrapEnvironment(auth.CredentialBindings, opts.Bindings)
		if err != nil {
			return Profile{}, err
		}
		var result bootstrapResponse
		request := bootstrapRequest{Method: method, Flow: driver.Flow, SetupURL: auth.SetupURL, RequestedScopes: append([]string(nil), auth.Scopes...), CredentialBindings: append([]string(nil), auth.CredentialBindings...), ExpectedAccount: profile.ExpectedAccount}
		bootstrapContext, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := runDriverJSON(bootstrapContext, driver.Bootstrap, request, bootstrapEnv, &result); err != nil {
			return Profile{}, fmt.Errorf("authentication bootstrap did not complete: %w", err)
		}
		if !result.Completed || !safeObservationValue(result.Account) || strings.TrimSpace(result.Account) == "" {
			return Profile{}, errors.New("authentication bootstrap did not return a completed account selection")
		}
		if profile.ExpectedAccount != "" && result.Account != profile.ExpectedAccount {
			return Profile{}, errors.New("authentication bootstrap selected a different account")
		}
		if len(missing(auth.Scopes, result.Scopes)) > 0 {
			return Profile{}, errors.New("authentication bootstrap did not preserve every requested scope")
		}
		profile.ExpectedAccount = result.Account
		profile.Driver = &driver
		profile.BootstrapComplete = true
		if driver.CredentialMode != "external-env" {
			for _, name := range auth.CredentialBindings {
				profile.Bindings[name] = BindingSource{Type: "driver", Reference: name}
			}
		}
	} else {
		profile.BootstrapComplete = true
		if err := configureExternalBindings(&profile, auth.CredentialBindings, opts.Bindings, opts.Generations); err != nil {
			return Profile{}, err
		}
	}
	for name := range opts.Bindings {
		if !contains(auth.CredentialBindings, name) {
			return Profile{}, fmt.Errorf("binding %s is not declared for %s", name, opts.Entry.ID)
		}
	}
	for name := range opts.Generations {
		if !contains(auth.CredentialBindings, name) {
			return Profile{}, fmt.Errorf("generation %s is not declared for %s", name, opts.Entry.ID)
		}
	}
	if err := validateValidator(profile.Validator); err != nil {
		return Profile{}, err
	}
	if profile.Validator != nil {
		pinned, err := pinValidator(*profile.Validator)
		if err != nil {
			return Profile{}, err
		}
		profile.Validator = &pinned
	}
	saved, err := saveProfile(opts.StateDir, profile)
	if err != nil {
		return Profile{}, err
	}
	return saved, nil
}

func configureExternalBindings(profile *Profile, names []string, references, generations map[string]string) error {
	for _, name := range names {
		if !safeCredentialBindingName(name) {
			return fmt.Errorf("credential binding %s is unsafe for process environment injection", name)
		}
		reference := strings.TrimSpace(references[name])
		if reference == "" {
			return fmt.Errorf("missing runtime binding for %s; use --binding %s=env:VARIABLE", name, name)
		}
		if !strings.HasPrefix(reference, "env:") || !validEnvironmentName(strings.TrimPrefix(reference, "env:")) {
			return fmt.Errorf("binding %s must reference a runtime environment name as env:VARIABLE", name)
		}
		generation := strings.TrimSpace(generations[name])
		if !strings.HasPrefix(generation, "env:") || !validEnvironmentName(strings.TrimPrefix(generation, "env:")) {
			return fmt.Errorf("binding %s requires a non-secret generation as env:VARIABLE", name)
		}
		if strings.TrimPrefix(reference, "env:") == strings.TrimPrefix(generation, "env:") {
			return fmt.Errorf("binding %s credential and generation references must be distinct", name)
		}
		profile.Bindings[name] = BindingSource{
			Type: "env", Reference: strings.TrimPrefix(reference, "env:"),
			GenerationReference: strings.TrimPrefix(generation, "env:"),
		}
	}
	return nil
}

func Status(ctx context.Context, opts InspectOptions) (AuthReport, error) {
	resolved, err := resolveAuthentication(ctx, opts)
	return resolved.Report, err
}

func Doctor(ctx context.Context, opts InspectOptions) (DoctorReport, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	environment, err := environmentID(opts.StateDir)
	if err != nil {
		return DoctorReport{}, err
	}
	report := DoctorReport{
		SchemaVersion: "skills-hub.doctor-evidence/v1", ContractID: "operational-readiness", ContractVersion: "1.0.3", EvidenceScope: "setup",
		TestID: "skills-hub/doctor", TestVersion: "1", Producer: "skills-hub",
		EntryID: opts.Entry.ID, EntryVersion: opts.Version, Module: opts.Module, Runtime: opts.Runtime, Platform: currentPlatform(), EnvironmentID: environment,
		ArtifactSHA256: "", DependencyLockSHA256: "not-evaluated", InstallSource: "unknown",
		ObservedAt: now.UTC(), ExpiresAt: now.Add(30 * 24 * time.Hour), Ready: true,
	}
	add := func(name, status, reason string) {
		report.Checks = append(report.Checks, DoctorCheck{Name: name, Status: status, Reason: reason})
		if status == "fail" {
			report.Ready = false
		}
	}
	if info, err := os.Stat(opts.PackageDir); err != nil || !info.IsDir() {
		add("installed", "fail", "installed package directory is unavailable")
	} else {
		contractSHA256, contractErr := installer.RuntimeContractSHA256(opts.Entry, opts.Version)
		if contractErr != nil {
			return DoctorReport{}, contractErr
		}
		report.RuntimeContractSHA256 = contractSHA256
		if receipt, err := verifyInstalledClosure(opts, contractSHA256); err != nil {
			add("installed", "fail", "install receipt or installed package integrity is invalid")
		} else {
			report.ArtifactSHA256, report.InstallSource = receipt.ArtifactSHA256, receipt.Source
			add("installed", "pass", "")
		}
		if lockDigest, lockErr := dependencyLockSHA256(opts.Entry, opts.PackageDir); lockErr == nil {
			report.DependencyLockSHA256 = lockDigest
		} else {
			add("dependency-lock", "fail", "dependency lock integrity is invalid")
		}
	}
	if opts.Entry.Deprecated {
		add("lifecycle", "fail", "selected entry is deprecated")
	} else {
		add("lifecycle", "pass", "")
	}
	if opts.Entry.Execution == nil {
		add("execution-contract", "fail", "entry has no executable contract")
	} else if !contains(opts.Entry.Runtimes, opts.Runtime) {
		add("runtime", "fail", "selected runtime is not declared")
	} else if !contains(opts.Entry.Execution.SupportedPlatforms, currentPlatform()) {
		add("platform", "fail", "current platform is not declared")
	} else {
		add("platform", "pass", "")
		if len(opts.Entry.Execution.SmokeTest) == 0 {
			add("smoke-entrypoint", "fail", "smoke test is not declared")
		} else if _, _, err := resolveCommandExecutable(opts.PackageDir, opts.Entry.Execution.SmokeTest[0]); err != nil {
			add("smoke-entrypoint", "fail", "declared smoke executable is unavailable")
		} else {
			add("smoke-entrypoint", "pass", "")
		}
	}
	resolved, err := resolveAuthentication(ctx, opts)
	if err != nil {
		return DoctorReport{}, err
	}
	auth := resolved.Report
	report.Authentication = auth
	report.CredentialKeys = resolved.Keys
	if !auth.ExpiresAt.IsZero() && auth.ExpiresAt.Before(report.ExpiresAt) {
		report.ExpiresAt = auth.ExpiresAt
	}
	if !auth.Required && !auth.Selected {
		add("authentication", "pass", "not required")
	} else if auth.Authenticated {
		add("authentication", "pass", "")
	} else {
		add("authentication", "fail", auth.State)
		switch auth.State {
		case "missing", "expired", "revoked", "wrong-account", "wrong-scope", "invalid", "stale":
			report.SetupRequired = true
		}
	}
	if auth.Selected {
		report.ConfigurationClass = "authenticated:" + auth.Method
	} else {
		report.ConfigurationClass = "no-authentication"
	}
	if report.Ready {
		report.Result = "passed"
	} else {
		report.Result = "failed"
	}
	if err := persistDoctorEvidence(opts.StateDir, report); err != nil {
		return DoctorReport{}, err
	}
	return report, nil
}

// DeriveCatalogStatus consumes persisted evidence only after revalidating the
// current installation, authentication target, credential generation, runtime,
// platform, and evidence expiry. A persisted success is never itself authority.
func DeriveCatalogStatus(ctx context.Context, opts InspectOptions) (CatalogStatus, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if opts.Entry.Deprecated {
		return CatalogStatus{Availability: "deprecated", Reason: "entry lifecycle policy is deprecated"}, nil
	}
	doctor, err := Doctor(ctx, opts)
	if err != nil {
		return CatalogStatus{}, err
	}
	if !doctor.Ready {
		if doctor.SetupRequired {
			return CatalogStatus{Availability: "setup-required", Reason: "current setup checks are not ready", ExpiresAt: doctor.ExpiresAt}, nil
		}
		return CatalogStatus{Availability: "not-verified", Reason: "current installation or runtime checks failed"}, nil
	}
	contractDigest, err := installer.RuntimeContractSHA256(opts.Entry, opts.Version)
	if err != nil {
		return CatalogStatus{}, err
	}
	receipt, err := verifyInstalledClosure(opts, contractDigest)
	if err != nil {
		return CatalogStatus{Availability: "not-verified", Reason: "current installation identity is invalid"}, nil
	}
	if receipt.Source != "remote" || !validSHA256Text(receipt.ArtifactSHA256) {
		return CatalogStatus{Availability: "not-verified", Reason: "local installations cannot establish published catalog status"}, nil
	}
	lockDigest, err := dependencyLockSHA256(opts.Entry, opts.PackageDir)
	if err != nil {
		return CatalogStatus{Availability: "not-verified", Reason: "current dependency lock is invalid"}, nil
	}
	resolved, err := resolveAuthentication(ctx, opts)
	if err != nil {
		return CatalogStatus{}, err
	}
	environment, err := environmentID(opts.StateDir)
	if err != nil {
		return CatalogStatus{}, err
	}
	executablePath, executableFingerprint, err := resolveCommandExecutable(opts.PackageDir, opts.Entry.Execution.SmokeTest[0])
	if err != nil || executablePath == "" {
		return CatalogStatus{Availability: "not-verified", Reason: "current smoke runtime is unavailable"}, nil
	}
	entries, err := os.ReadDir(filepath.Join(opts.StateDir, "evidence"))
	if errors.Is(err, os.ErrNotExist) {
		return CatalogStatus{Availability: "not-verified", Reason: "no executable evidence is available"}, nil
	}
	if err != nil {
		return CatalogStatus{}, err
	}
	var selected *SmokeEvidence
	selectedPath := ""
	configurationClass := "no-authentication"
	if resolved.Report.Selected {
		configurationClass = "authenticated:" + resolved.Report.Method
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(opts.StateDir, "evidence", entry.Name())
		payload, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		if validateSmokeEvidencePayload(payload) != nil {
			continue
		}
		var candidate SmokeEvidence
		decoder := json.NewDecoder(bytes.NewReader(payload))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&candidate) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			continue
		}
		if !evidenceMatchesStableTarget(candidate, opts, environment, now) {
			continue
		}
		if candidate.Result == "passed" && !evidenceTargetsCurrent(candidate, opts, receipt, lockDigest, resolved, environment, executableFingerprint, configurationClass, now) {
			continue
		}
		if selected == nil || candidate.ObservedAt.After(selected.ObservedAt) || (candidate.ObservedAt.Equal(selected.ObservedAt) && (candidate.StartedAt.After(selected.StartedAt) || (candidate.StartedAt.Equal(selected.StartedAt) && candidate.Result == "failed" && selected.Result == "passed"))) {
			copy := candidate
			selected = &copy
			selectedPath = path
		}
	}
	if selected == nil {
		return CatalogStatus{Availability: "not-verified", Reason: "no current evidence matches the installed target"}, nil
	}
	if selected.Result != "passed" {
		return CatalogStatus{Availability: "not-verified", EvidenceScope: "executable", EvidencePath: selectedPath, Reason: "latest applicable smoke evidence failed"}, nil
	}
	if !selected.PromotionEligible || !now.Before(selected.ExpiresAt) || !selected.ExpiresAt.After(selected.ObservedAt) || selected.ExpiresAt.After(selected.ObservedAt.Add(90*24*time.Hour)) || (resolved.Report.Selected && selected.ExpiresAt.After(resolved.Report.ExpiresAt)) {
		return CatalogStatus{Availability: "not-verified", EvidenceScope: "executable", EvidencePath: selectedPath, Reason: "latest applicable smoke evidence is ineligible or expired"}, nil
	}
	return CatalogStatus{Availability: "usable-now", EvidenceScope: "executable", EvidencePath: selectedPath, ExpiresAt: selected.ExpiresAt}, nil
}

func evidenceMatchesStableTarget(candidate SmokeEvidence, opts InspectOptions, environment string, now time.Time) bool {
	if candidate.EntryID != opts.Entry.ID || candidate.EntryVersion != opts.Version || candidate.Module != opts.Module || candidate.Runtime != opts.Runtime || candidate.Platform != currentPlatform() || candidate.EnvironmentID != environment || candidate.StartedAt.IsZero() || candidate.ObservedAt.IsZero() || candidate.ExpiresAt.IsZero() || candidate.ObservedAt.Before(candidate.StartedAt) || candidate.ExpiresAt.Before(candidate.ObservedAt) || now.Before(candidate.ObservedAt) {
		return false
	}
	command := []string{}
	if opts.Entry.Execution != nil {
		command = opts.Entry.Execution.SmokeTest
	}
	return sameStringSlice(candidate.Command, command)
}

func validateSmokeEvidencePayload(payload []byte) error {
	smokeEvidenceSchemaOnce.Do(func() {
		schemaPayload, err := manifestschemas.ManifestFiles.ReadFile("smoke-evidence.schema.json")
		if err != nil {
			smokeEvidenceSchemaErr = err
			return
		}
		var schemaDocument any
		if err := json.Unmarshal(schemaPayload, &schemaDocument); err != nil {
			smokeEvidenceSchemaErr = err
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		const schemaURL = "https://skills.ai-knowledge-hub.org/schemas/smoke-evidence.schema.json"
		if err := compiler.AddResource(schemaURL, schemaDocument); err != nil {
			smokeEvidenceSchemaErr = err
			return
		}
		smokeEvidenceContract, smokeEvidenceSchemaErr = compiler.Compile(schemaURL)
	})
	if smokeEvidenceSchemaErr != nil {
		return smokeEvidenceSchemaErr
	}
	var document any
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("smoke evidence contains trailing data")
	}
	return smokeEvidenceContract.Validate(document)
}

func evidenceTargetsCurrent(candidate SmokeEvidence, opts InspectOptions, receipt installer.InstallReceipt, lockDigest string, resolved resolvedAuth, environment, executableFingerprint, configurationClass string, now time.Time) bool {
	if candidate.SchemaVersion != evidenceSchema || (candidate.Result != "passed" && candidate.Result != "failed") || candidate.EntryID != opts.Entry.ID || candidate.EntryVersion != opts.Version || candidate.Module != opts.Module || candidate.Runtime != opts.Runtime || candidate.Platform != currentPlatform() || candidate.EnvironmentID != environment || candidate.ObservedAt.Before(candidate.StartedAt) || now.Before(candidate.ObservedAt) || !sameStringSlice(candidate.Command, opts.Entry.Execution.SmokeTest) || candidate.ConfigurationClass != configurationClass {
		return false
	}
	return candidate.InstallSource == receipt.Source && strings.EqualFold(candidate.ArtifactSHA256, receipt.ArtifactSHA256) && candidate.DependencyLockSHA256 == lockDigest && sameStringMap(candidate.CredentialKeys, resolved.Keys) && candidate.ProviderTargetFingerprint == resolved.Report.ProviderTargetFingerprint && candidate.ProviderIdentity == resolved.Report.Principal && candidate.ProviderAccount == resolved.Report.Account && candidate.AttestationReference == resolved.Report.AttestationReference && sameStringSet(candidate.GrantedScopes, resolved.Report.GrantedScopes) && candidate.ValidatorSHA256 == resolved.ValidatorSHA256 && strings.EqualFold(candidate.ExecutionRuntimeFingerprint, executableFingerprint)
}

func Smoke(ctx context.Context, opts InspectOptions) (SmokeEvidence, string, error) {
	clock := inspectionClock(opts)
	now := clock()
	authOpts := opts
	authOpts.Now = now
	evidence := SmokeEvidence{
		SchemaVersion: evidenceSchema, ContractID: "operational-readiness", ContractVersion: "1.0.3",
		EvidenceScope: "executable", TestID: "skills-hub/declared-smoke", TestVersion: "1", Producer: "skills-hub",
		EntryID: opts.Entry.ID, EntryVersion: opts.Version,
		Module: opts.Module, Runtime: opts.Runtime, Platform: currentPlatform(), StartedAt: now.UTC(), Result: "failed",
		DependencyLockSHA256: "not-evaluated", InstallSource: "unknown", Command: []string{},
		ConfigurationClass: "not-evaluated", Coverage: []string{"install-receipt", "installed-tree", "declared-smoke-test"},
	}
	if opts.Entry.Execution != nil {
		evidence.Command = append(evidence.Command, opts.Entry.Execution.SmokeTest...)
	}
	persist := func() (SmokeEvidence, string, error) {
		if evidence.ObservedAt.IsZero() {
			evidence.ObservedAt = clock()
			if evidence.ObservedAt.Before(evidence.StartedAt) {
				evidence.ObservedAt = evidence.StartedAt
			}
		}
		if evidence.Result == "failed" && (evidence.ExpiresAt.IsZero() || evidence.ExpiresAt.Before(evidence.ObservedAt)) {
			evidence.ExpiresAt = evidence.ObservedAt
		}
		return persistEvidence(opts.StateDir, evidence)
	}
	environment, err := environmentID(opts.StateDir)
	if err != nil {
		return evidence, "", err
	}
	evidence.EnvironmentID = environment
	if opts.Entry.Deprecated {
		evidence.FailureReason = "entry-deprecated"
		return persist()
	}
	if opts.Entry.Execution == nil || len(opts.Entry.Execution.SmokeTest) == 0 {
		evidence.FailureReason = "smoke-test-not-declared"
		return persist()
	}
	if !contains(opts.Entry.Runtimes, opts.Runtime) {
		evidence.FailureReason = "runtime-not-supported"
		return persist()
	}
	if !contains(opts.Entry.Execution.SupportedPlatforms, currentPlatform()) {
		evidence.FailureReason = "platform-not-supported"
		return persist()
	}
	if info, err := os.Stat(opts.PackageDir); err != nil || !info.IsDir() {
		evidence.FailureReason = "package-not-installed"
		return persist()
	}
	contractSHA256, err := installer.RuntimeContractSHA256(opts.Entry, opts.Version)
	if err != nil {
		return evidence, "", err
	}
	receipt, err := verifyInstalledClosure(opts, contractSHA256)
	if err != nil {
		evidence.FailureReason = "installed-package-integrity-invalid"
		return persist()
	}
	evidence.InstallSource = receipt.Source
	evidence.ArtifactSHA256 = receipt.ArtifactSHA256
	evidence.DependencyLockSHA256, err = dependencyLockSHA256(opts.Entry, opts.PackageDir)
	if err != nil {
		evidence.FailureReason = "dependency-lock-integrity-invalid"
		return persist()
	}
	if receipt.Source == "remote" {
		selectedSHA := selectedArtifactSHA256(opts.Entry, opts.Version)
		if selectedSHA == "" || !strings.EqualFold(selectedSHA, receipt.ArtifactSHA256) {
			evidence.FailureReason = "artifact-identity-unavailable"
			return persist()
		}
	}
	executablePath, executableFingerprint, err := resolveCommandExecutable(opts.PackageDir, evidence.Command[0])
	if err != nil {
		evidence.FailureReason = "smoke-runtime-unavailable"
		return persist()
	}
	evidence.ExecutionRuntimeFingerprint = executableFingerprint
	resolved, err := resolveAuthentication(ctx, authOpts)
	if err != nil {
		return evidence, "", err
	}
	if (resolved.Report.Required || resolved.Report.Selected) && !resolved.Report.Authenticated {
		evidence.ConfigurationClass = "authentication-" + resolved.Report.State
		evidence.FailureReason = "authentication-" + resolved.Report.State
		return persist()
	}
	evidence.CredentialKeys = resolved.Keys
	evidence.ProviderIdentity = resolved.Report.Principal
	evidence.ProviderAccount = resolved.Report.Account
	evidence.ProviderTargetFingerprint = resolved.Report.ProviderTargetFingerprint
	evidence.AttestationReference = resolved.Report.AttestationReference
	evidence.GrantedScopes = append([]string(nil), resolved.Report.GrantedScopes...)
	evidence.ValidatorSHA256 = resolved.ValidatorSHA256
	if resolved.Report.Selected {
		evidence.ConfigurationClass = "authenticated:" + resolved.Report.Method
		evidence.Coverage = append(evidence.Coverage, "authoritative-provider-identity")
		evidence.ExpiresAt = resolved.Report.ExpiresAt
	} else {
		evidence.ConfigurationClass = "no-authentication"
		evidence.ExpiresAt = now.Add(90 * 24 * time.Hour)
	}
	command := exec.CommandContext(ctx, executablePath, evidence.Command[1:]...)
	command.Dir = opts.PackageDir
	command.Env = subprocessEnvironment(resolved.Env)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			evidence.FailureReason = "smoke-test-timeout"
		} else {
			evidence.FailureReason = "smoke-test-failed"
		}
		return persist()
	}
	currentFingerprint, err := fileSHA256(executablePath)
	if err != nil || !strings.EqualFold(currentFingerprint, executableFingerprint) {
		evidence.FailureReason = "smoke-runtime-changed"
		return persist()
	}
	if _, err := verifyInstalledClosure(opts, contractSHA256); err != nil {
		evidence.FailureReason = "installed-package-changed-during-smoke"
		return persist()
	}
	if !bindingGenerationsCurrent(ctx, resolved) {
		evidence.FailureReason = "credential-binding-changed-during-smoke"
		return persist()
	}
	observedAt := clock()
	if observedAt.Before(evidence.StartedAt) {
		observedAt = evidence.StartedAt
	}
	if !evidence.ExpiresAt.After(observedAt) {
		evidence.FailureReason = "authentication-expired-during-smoke"
		return persist()
	}
	evidence.Result = "passed"
	evidence.PromotionEligible = receipt.Source == "remote"
	evidence.ObservedAt = observedAt
	return persist()
}

func inspectionClock(opts InspectOptions) func() time.Time {
	if opts.Clock != nil {
		return func() time.Time { return opts.Clock().UTC() }
	}
	if !opts.Now.IsZero() {
		fixed := opts.Now.UTC()
		return func() time.Time { return fixed }
	}
	return func() time.Time { return time.Now().UTC() }
}

func verifyInstalledClosure(opts InspectOptions, runtimeContractSHA256 string) (installer.InstallReceipt, error) {
	receipt, err := installer.VerifyInstallReceipt(opts.PackageDir, opts.Module, opts.Entry.ID, opts.Version, opts.Runtime, "", runtimeContractSHA256)
	if err != nil {
		return installer.InstallReceipt{}, err
	}
	if opts.Module == "plugins" {
		if strings.TrimSpace(opts.TargetRoot) == "" {
			return installer.InstallReceipt{}, errors.New("plugin target root is required to verify the installed closure")
		}
		if err := installer.VerifyInstallClosure(opts.PackageDir, opts.TargetRoot, receipt, opts.Entry.Includes); err != nil {
			return installer.InstallReceipt{}, err
		}
	}
	return receipt, nil
}

func resolveAuthentication(ctx context.Context, opts InspectOptions) (resolvedAuth, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	report := AuthReport{EntryID: opts.Entry.ID, EntryVersion: opts.Version, ObservedAt: now.UTC()}
	auth, err := authentication(opts.Entry)
	if err != nil {
		return resolvedAuth{}, err
	}
	report.Required = auth.Status == "required"
	report.RequiredScopes = append([]string(nil), auth.Scopes...)
	if auth.Status == "none" {
		report.Configured, report.Authenticated, report.State, report.Method = true, true, "not-required", "none"
		return resolvedAuth{Report: report}, nil
	}
	profile, err := loadProfile(opts.StateDir, opts.Module, opts.Entry.ID)
	if errors.Is(err, os.ErrNotExist) {
		if report.Required {
			report.State, report.Reason = "missing", "authentication profile is not configured"
		} else {
			report.State, report.Reason = "optional-not-configured", "optional authentication was not selected"
		}
		return resolvedAuth{Report: report}, nil
	}
	if err != nil {
		return resolvedAuth{}, err
	}
	report.Selected = true
	if !contains(auth.Methods, profile.Method) {
		return resolvedAuth{}, errors.New("authentication profile method is not declared by the selected entry")
	}
	if profile.ExpectedAccount != "" && !safeObservationValue(profile.ExpectedAccount) {
		return resolvedAuth{}, errors.New("authentication profile account is invalid")
	}
	if len(profile.Bindings) != len(auth.CredentialBindings) {
		return resolvedAuth{}, errors.New("authentication profile bindings do not match the selected entry")
	}
	if !sameStringSet(profile.RequestedScopes, auth.Scopes) || !profile.BootstrapComplete || profile.BootstrapMode != authMethodFlows[profile.Method] {
		return resolvedAuth{}, errors.New("authentication profile bootstrap scope does not match the selected entry")
	}
	for _, name := range auth.CredentialBindings {
		binding, ok := profile.Bindings[name]
		if !ok || binding.Reference == binding.GenerationReference {
			return resolvedAuth{}, errors.New("authentication profile bindings do not match the selected entry")
		}
	}
	if profile.EntryVersion != opts.Version {
		report.State, report.Reason = "stale", "authentication profile targets a different entry version"
		return resolvedAuth{Report: report}, nil
	}
	if err := validatePinnedValidator(profile.Validator); err != nil {
		return resolvedAuth{}, err
	}
	report.Method = profile.Method
	resolved := resolvedAuth{
		Report: report, Keys: make(map[string]string), Generations: make(map[string]string),
		GenerationReferences: make(map[string]string),
	}
	if profile.Driver != nil {
		if profile.Driver.Method != profile.Method || profile.Driver.Flow != authMethodFlows[profile.Method] {
			return resolvedAuth{}, errors.New("authentication profile driver does not match the selected method")
		}
		if err := validatePinnedDriver(*profile.Driver); err != nil {
			return resolvedAuth{}, err
		}
		for _, binding := range profile.Bindings {
			if profile.Driver.CredentialMode == "external-env" && binding.Type != "env" {
				return resolvedAuth{}, errors.New("authentication profile bindings do not match the driver credential mode")
			}
			if profile.Driver.CredentialMode == "driver" && binding.Type != "driver" {
				return resolvedAuth{}, errors.New("authentication profile bindings do not match the driver credential mode")
			}
		}
		contractDigest, digestErr := installer.RuntimeContractSHA256(opts.Entry, opts.Version)
		if digestErr != nil {
			return resolvedAuth{}, digestErr
		}
		if _, verifyErr := verifyInstalledClosure(InspectOptions{Module: opts.Module, Entry: opts.Entry, Version: opts.Version, PackageDir: profile.Driver.PackageDir, TargetRoot: profile.Driver.TargetRoot, Runtime: profile.Driver.Runtime}, contractDigest); verifyErr != nil {
			return resolvedAuth{}, errors.New("packaged authentication driver installation is no longer valid")
		}
		resolved.Driver = profile.Driver
	}
	for _, name := range auth.CredentialBindings {
		binding, ok := profile.Bindings[name]
		if !ok {
			resolved.Report.MissingBindings = append(resolved.Report.MissingBindings, name)
			continue
		}
		if binding.Type == "driver" && profile.Driver != nil && binding.Reference == name {
			var credential credentialResponse
			if err := runDriverJSON(ctx, profile.Driver.Credential, map[string]string{"binding": name}, nil, &credential); err != nil || credential.Value == "" || !safeGeneration(credential.Generation) {
				resolved.Report.MissingBindings = append(resolved.Report.MissingBindings, name)
				continue
			}
			resolved.Env = append(resolved.Env, name+"="+credential.Value)
			resolved.Generations[name] = credential.Generation
			resolved.Keys[name] = fmt.Sprintf("%d:%s", binding.Revision, credential.Generation)
			continue
		}
		if binding.Type != "env" || !validEnvironmentName(binding.Reference) || !validEnvironmentName(binding.GenerationReference) {
			resolved.Report.MissingBindings = append(resolved.Report.MissingBindings, name)
			continue
		}
		value, ok := os.LookupEnv(binding.Reference)
		if !ok || value == "" {
			resolved.Report.MissingBindings = append(resolved.Report.MissingBindings, name)
			continue
		}
		resolved.Env = append(resolved.Env, name+"="+value)
		generation, ok := os.LookupEnv(binding.GenerationReference)
		if !ok || !safeGeneration(generation) {
			resolved.Report.MissingBindings = append(resolved.Report.MissingBindings, name)
			continue
		}
		resolved.Generations[name] = generation
		resolved.GenerationReferences[name] = binding.GenerationReference
		resolved.Keys[name] = fmt.Sprintf("%d:%s", binding.Revision, generation)
	}
	if len(resolved.Report.MissingBindings) > 0 {
		sort.Strings(resolved.Report.MissingBindings)
		resolved.Report.State, resolved.Report.Reason = "missing", "one or more runtime credential bindings are unavailable"
		return resolved, nil
	}
	resolved.Report.Configured = true
	if profile.Validator == nil && profile.Driver == nil {
		resolved.Report.State, resolved.Report.Reason = "unverified", "no authoritative validation command is configured"
		return resolved, nil
	}
	var observation validatorObservation
	if profile.Driver != nil {
		err = runDriverJSON(ctx, profile.Driver.Status, map[string]string{"method": profile.Method}, resolved.Env, &observation)
		resolved.ValidatorSHA256 = profile.Driver.Status.SHA256
	} else {
		observation, err = runValidator(ctx, *profile.Validator, resolved.Env)
		resolved.ValidatorSHA256 = profile.Validator.SHA256
	}
	if err != nil {
		resolved.Report.State, resolved.Report.Reason = "unavailable", "authoritative credential validation did not complete"
		return resolved, nil
	}
	if !bindingGenerationsCurrent(ctx, resolved) {
		resolved.Report.State, resolved.Report.Reason = "stale", "a credential binding changed during provider validation"
		return resolved, nil
	}
	resolved.Report.Principal, resolved.Report.Account = observation.Principal, observation.Account
	resolved.Report.GrantedScopes = sortedUnique(observation.Scopes)
	resolved.Report.AttestationReference = observation.AttestationReference
	if observation.Revoked {
		resolved.Report.State, resolved.Report.Reason = "revoked", "the authoritative provider reports the credential as revoked"
		return resolved, nil
	}
	if observation.ExpiresAt != nil && !observation.ExpiresAt.After(now) {
		resolved.Report.State, resolved.Report.Reason = "expired", "the credential has expired"
		return resolved, nil
	}
	if observation.AttestationExpiresAt == nil || !observation.AttestationExpiresAt.After(now) {
		resolved.Report.State, resolved.Report.Reason = "expired", "the provider identity attestation has expired"
		return resolved, nil
	}
	if profile.ExpectedAccount != "" && observation.Account != profile.ExpectedAccount {
		resolved.Report.State, resolved.Report.Reason = "wrong-account", "the verified provider account does not match the configured account"
		return resolved, nil
	}
	resolved.Report.MissingScopes = missing(auth.Scopes, observation.Scopes)
	if len(resolved.Report.MissingScopes) > 0 {
		resolved.Report.State, resolved.Report.Reason = "wrong-scope", "the verified credential does not grant every required scope"
		return resolved, nil
	}
	if !observation.Authenticated || strings.TrimSpace(observation.Principal) == "" {
		resolved.Report.State, resolved.Report.Reason = "invalid", "the authoritative provider did not verify a principal"
		return resolved, nil
	}
	fingerprint, err := providerTargetFingerprint(observation, profile.Method)
	if err != nil {
		resolved.Report.State, resolved.Report.Reason = "invalid", "the authoritative provider returned incomplete target identity"
		return resolved, nil
	}
	resolved.Report.ProviderTargetFingerprint = fingerprint
	expiresAt := now.Add(30 * 24 * time.Hour)
	if observation.ExpiresAt != nil && observation.ExpiresAt.Before(expiresAt) {
		expiresAt = observation.ExpiresAt.UTC()
	}
	if observation.AttestationExpiresAt.Before(expiresAt) {
		expiresAt = observation.AttestationExpiresAt.UTC()
	}
	resolved.Report.ExpiresAt = expiresAt
	resolved.Report.Authenticated, resolved.Report.State = true, "ready"
	return resolved, nil
}

func runValidator(ctx context.Context, validator Validator, env []string) (validatorObservation, error) {
	digest, err := fileSHA256(validator.Command)
	if err != nil || !strings.EqualFold(digest, validator.SHA256) {
		return validatorObservation{}, errors.New("configured validator identity has changed")
	}
	command := exec.CommandContext(ctx, validator.Command, validator.Args...)
	command.Env = subprocessEnvironment(env)
	command.Stderr = io.Discard
	var output limitedBuffer
	command.Stdout = &output
	if err := command.Run(); err != nil {
		return validatorObservation{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	decoder.DisallowUnknownFields()
	var observation validatorObservation
	if err := decoder.Decode(&observation); err != nil {
		return validatorObservation{}, errors.New("validator returned an invalid observation")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return validatorObservation{}, errors.New("validator returned trailing output")
	}
	if len(observation.Scopes) > 128 {
		return validatorObservation{}, errors.New("validator returned unsafe identity metadata")
	}
	metadata := []string{observation.Principal, observation.Account, observation.Provider, observation.Tier, observation.Endpoint, observation.Region, observation.APIVersion, observation.AttestationReference}
	for _, value := range append(metadata, observation.Scopes...) {
		if !safeObservationValue(value) {
			return validatorObservation{}, errors.New("validator returned unsafe identity metadata")
		}
	}
	return observation, nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(payload []byte) (int, error) {
	if b.Len()+len(payload) > 64*1024 {
		return 0, errors.New("validator output exceeds 64 KiB")
	}
	return b.Buffer.Write(payload)
}

func authentication(entry registry.SkillEntry) (registry.AuthenticationMetadata, error) {
	if entry.Authentication == nil {
		return registry.AuthenticationMetadata{}, fmt.Errorf("%s has no machine-readable authentication contract", entry.ID)
	}
	return *entry.Authentication, nil
}

func requiresPackagedAuthDriver(method string) bool {
	return method != "api-key" && method != "bearer-token"
}

func authDriverExists(packageDir, method string) bool {
	if strings.TrimSpace(packageDir) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(packageDir, "auth", method+".json"))
	return err == nil && info.Mode().IsRegular()
}

func loadAuthDriver(opts ConfigureOptions, method string) (AuthDriver, error) {
	if strings.TrimSpace(opts.PackageDir) == "" || strings.TrimSpace(opts.Runtime) == "" {
		return AuthDriver{}, fmt.Errorf("authentication method %s requires an installed package and --runtime", method)
	}
	contractDigest, err := installer.RuntimeContractSHA256(opts.Entry, opts.Version)
	if err != nil {
		return AuthDriver{}, err
	}
	if _, err := verifyInstalledClosure(InspectOptions{Module: opts.Module, Entry: opts.Entry, Version: opts.Version, PackageDir: opts.PackageDir, TargetRoot: opts.TargetRoot, Runtime: opts.Runtime}, contractDigest); err != nil {
		return AuthDriver{}, fmt.Errorf("verify packaged authentication driver: %w", err)
	}
	driverPath := filepath.Join(opts.PackageDir, "auth", method+".json")
	payload, err := os.ReadFile(driverPath)
	if err != nil {
		return AuthDriver{}, fmt.Errorf("read packaged authentication driver: %w", err)
	}
	var document authDriverDocument
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return AuthDriver{}, errors.New("packaged authentication driver is malformed")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return AuthDriver{}, errors.New("packaged authentication driver contains trailing data")
	}
	if document.SchemaVersion != "skills-hub.auth-driver/v1" || document.Method != method || document.Flow != authMethodFlows[method] || !contains(document.Runtimes, opts.Runtime) {
		return AuthDriver{}, errors.New("packaged authentication driver does not match the selected method")
	}
	bootstrap, err := pinDriverCommand(opts.PackageDir, document.Bootstrap)
	if err != nil {
		return AuthDriver{}, fmt.Errorf("bootstrap command: %w", err)
	}
	credential, err := pinDriverCommand(opts.PackageDir, document.Credential)
	if err != nil {
		return AuthDriver{}, fmt.Errorf("credential command: %w", err)
	}
	status, err := pinDriverCommand(opts.PackageDir, document.Status)
	if err != nil {
		return AuthDriver{}, fmt.Errorf("status command: %w", err)
	}
	credentialMode := document.CredentialMode
	if credentialMode == "" {
		credentialMode = "driver"
	}
	return AuthDriver{SchemaVersion: document.SchemaVersion, CredentialMode: credentialMode, Method: method, Flow: document.Flow, PackageDir: opts.PackageDir, TargetRoot: opts.TargetRoot, Runtime: opts.Runtime, Bootstrap: bootstrap, Credential: credential, Status: status}, nil
}

func pinDriverCommand(packageDir string, document authDriverCommandDoc) (DriverCommand, error) {
	if len(document.Command) == 0 {
		return DriverCommand{}, errors.New("command is missing")
	}
	relative := filepath.Clean(filepath.FromSlash(document.Command[0]))
	if filepath.IsAbs(relative) || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return DriverCommand{}, errors.New("command must be package-relative")
	}
	resolved := filepath.Join(packageDir, relative)
	inside, err := filepath.Rel(packageDir, resolved)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return DriverCommand{}, errors.New("command escapes the installed package")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return DriverCommand{}, errors.New("command is not an executable package file")
	}
	for _, argument := range document.Command[1:] {
		if strings.ContainsRune(argument, '\x00') || registry.ContainsCredentialShapedValue(argument) {
			return DriverCommand{}, errors.New("command arguments contain unsafe material")
		}
	}
	digest, err := fileSHA256(resolved)
	if err != nil {
		return DriverCommand{}, err
	}
	return DriverCommand{Command: resolved, Args: append([]string(nil), document.Command[1:]...), SHA256: digest}, nil
}

func validatePinnedDriver(driver AuthDriver) error {
	if driver.SchemaVersion != "skills-hub.auth-driver/v1" || strings.TrimSpace(driver.PackageDir) == "" || strings.TrimSpace(driver.Runtime) == "" {
		return errors.New("authentication profile driver is invalid")
	}
	if driver.CredentialMode != "" && driver.CredentialMode != "driver" && driver.CredentialMode != "external-env" {
		return errors.New("authentication profile driver is invalid")
	}
	for _, command := range []DriverCommand{driver.Bootstrap, driver.Credential, driver.Status} {
		relative, err := filepath.Rel(driver.PackageDir, command.Command)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || !validSHA256Text(command.SHA256) {
			return errors.New("authentication profile driver is invalid")
		}
	}
	return nil
}

func validSHA256Text(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func explicitBootstrapEnvironment(bindingNames []string, references map[string]string) ([]string, error) {
	environment := make([]string, 0, len(references))
	for name, reference := range references {
		if !contains(bindingNames, name) || !strings.HasPrefix(reference, "env:") || !validEnvironmentName(strings.TrimPrefix(reference, "env:")) {
			return nil, fmt.Errorf("bootstrap binding %s is not a declared env reference", name)
		}
		value, ok := os.LookupEnv(strings.TrimPrefix(reference, "env:"))
		if !ok || value == "" {
			return nil, fmt.Errorf("bootstrap binding %s is unavailable", name)
		}
		environment = append(environment, name+"="+value)
	}
	return environment, nil
}

func runDriverJSON(ctx context.Context, commandSpec DriverCommand, input any, env []string, output any) error {
	digest, err := fileSHA256(commandSpec.Command)
	if err != nil || !strings.EqualFold(digest, commandSpec.SHA256) {
		return errors.New("packaged authentication driver identity has changed")
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, commandSpec.Command, commandSpec.Args...)
	command.Env = subprocessEnvironment(env)
	command.Stdin = bytes.NewReader(payload)
	command.Stderr = io.Discard
	var buffer limitedBuffer
	command.Stdout = &buffer
	if err := command.Run(); err != nil {
		return errors.New("packaged authentication driver failed")
	}
	decoder := json.NewDecoder(bytes.NewReader(buffer.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("packaged authentication driver returned an invalid response")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("packaged authentication driver returned trailing data")
	}
	return nil
}

func validateValidator(validator *Validator) error {
	if validator == nil {
		return nil
	}
	validator.Command = strings.TrimSpace(validator.Command)
	if validator.Command == "" {
		return errors.New("validator command cannot be empty")
	}
	for _, value := range append([]string{validator.Command}, validator.Args...) {
		if strings.ContainsRune(value, '\x00') || registry.ContainsCredentialShapedValue(value) {
			return errors.New("validator command must not contain credentials")
		}
	}
	return nil
}

func pinValidator(validator Validator) (Validator, error) {
	resolved, err := exec.LookPath(validator.Command)
	if err != nil {
		return Validator{}, errors.New("validator command is not available")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return Validator{}, errors.New("validator command path cannot be resolved")
	}
	digest, err := fileSHA256(resolved)
	if err != nil {
		return Validator{}, errors.New("validator command cannot be integrity-bound")
	}
	validator.Command = resolved
	validator.SHA256 = digest
	return validator, nil
}

func validatePinnedValidator(validator *Validator) error {
	if validator == nil {
		return nil
	}
	if err := validateValidator(validator); err != nil {
		return err
	}
	if !filepath.IsAbs(validator.Command) || len(validator.SHA256) != sha256.Size*2 {
		return errors.New("authentication profile validator identity is invalid")
	}
	if _, err := hex.DecodeString(validator.SHA256); err != nil {
		return errors.New("authentication profile validator identity is invalid")
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("validator command is not a regular file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func resolveCommandExecutable(packageDir, command string) (string, string, error) {
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsRune(command, '\x00') {
		return "", "", errors.New("smoke command is invalid")
	}
	var resolved string
	var err error
	if strings.ContainsAny(command, "/\\") {
		resolved = filepath.FromSlash(command)
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(packageDir, resolved)
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return "", "", errors.New("smoke command path cannot be resolved")
		}
		root, err := filepath.Abs(packageDir)
		if err != nil {
			return "", "", errors.New("installed package path cannot be resolved")
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", "", errors.New("smoke command escapes the installed package")
		}
	} else {
		resolved, err = exec.LookPath(command)
		if err != nil {
			return "", "", errors.New("smoke command is unavailable")
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return "", "", errors.New("smoke command path cannot be resolved")
		}
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", errors.New("smoke command is not a regular executable")
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return "", "", errors.New("smoke command is not executable")
	}
	digest, err := fileSHA256(resolved)
	if err != nil {
		return "", "", errors.New("smoke command cannot be integrity-bound")
	}
	return resolved, digest, nil
}

func loadProfile(root, module, entryID string) (Profile, error) {
	path, err := profilePath(root, module, entryID)
	if err != nil {
		return Profile{}, err
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var profile Profile
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&profile); err != nil {
		return Profile{}, fmt.Errorf("parse authentication profile: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Profile{}, errors.New("authentication profile contains trailing data")
	}
	if profile.SchemaVersion != profileSchema || profile.Module != module || profile.EntryID != entryID {
		return Profile{}, errors.New("authentication profile identity is invalid")
	}
	return profile, nil
}

func saveProfile(root string, profile Profile) (Profile, error) {
	path, err := profilePath(root, profile.Module, profile.EntryID)
	if err != nil {
		return Profile{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Profile{}, fmt.Errorf("create authentication profile directory: %w", err)
	}
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return Profile{}, fmt.Errorf("lock authentication profile: %w", err)
	}
	defer lock.Unlock()
	previous, loadErr := loadProfile(root, profile.Module, profile.EntryID)
	if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
		return Profile{}, loadErr
	}
	for name, binding := range profile.Bindings {
		binding.Revision = 1
		if old, ok := previous.Bindings[name]; ok {
			binding.Revision = old.Revision + 1
		}
		profile.Bindings[name] = binding
	}
	payload, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return Profile{}, err
	}
	if err := atomicWrite(path, append(payload, '\n'), 0o600); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func persistEvidence(root string, evidence SmokeEvidence) (SmokeEvidence, string, error) {
	if evidence.ObservedAt.IsZero() {
		evidence.ObservedAt = time.Now().UTC()
	}
	if evidence.ExpiresAt.IsZero() {
		evidence.ExpiresAt = evidence.ObservedAt
	}
	key := sha256.Sum256([]byte(strings.Join([]string{
		evidence.Module, evidence.EntryID, evidence.EntryVersion, evidence.Runtime,
		evidence.Platform, evidence.ArtifactSHA256, evidence.DependencyLockSHA256,
		evidence.EnvironmentID,
		evidence.ConfigurationClass, evidence.StartedAt.UTC().Format(time.RFC3339Nano),
		evidence.ObservedAt.UTC().Format(time.RFC3339Nano), evidence.Result,
	}, "\x00")))
	dir := filepath.Join(root, "evidence")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return evidence, "", fmt.Errorf("create evidence directory: %w", err)
	}
	payload, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return evidence, "", err
	}
	if err := validateSmokeEvidencePayload(payload); err != nil {
		return evidence, "", fmt.Errorf("validate smoke evidence: %w", err)
	}
	base := hex.EncodeToString(key[:])[:24]
	path := ""
	for sequence := 0; sequence < 1000; sequence++ {
		name := base + ".json"
		if sequence > 0 {
			name = fmt.Sprintf("%s-%d.json", base, sequence)
		}
		path = filepath.Join(dir, name)
		err = atomicWriteNew(path, append(payload, '\n'), 0o600)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return evidence, "", err
		}
	}
	if err != nil {
		return evidence, "", errors.New("smoke evidence sequence is exhausted")
	}
	if evidence.Result != "passed" {
		return evidence, path, fmt.Errorf("smoke test failed: %s", evidence.FailureReason)
	}
	return evidence, path, nil
}

func persistDoctorEvidence(root string, report DoctorReport) error {
	key := sha256.Sum256([]byte(strings.Join([]string{
		report.Module, report.EntryID, report.EntryVersion, report.Runtime,
		report.Platform, report.ArtifactSHA256, report.RuntimeContractSHA256,
		report.DependencyLockSHA256, report.EnvironmentID,
		report.ConfigurationClass, report.ObservedAt.UTC().Format(time.RFC3339Nano),
	}, "\x00")))
	dir := filepath.Join(root, "doctor-evidence")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create doctor evidence directory: %w", err)
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	base := hex.EncodeToString(key[:])[:24]
	for sequence := 0; sequence < 1000; sequence++ {
		name := base + ".json"
		if sequence > 0 {
			name = fmt.Sprintf("%s-%d.json", base, sequence)
		}
		err := atomicWriteNew(filepath.Join(dir, name), append(payload, '\n'), 0o600)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	return errors.New("doctor evidence sequence is exhausted")
}

func atomicWriteNew(path string, payload []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".skills-hub-evidence-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if atomicWriteNewBeforePublish != nil {
		if err := atomicWriteNewBeforePublish(path, tempPath); err != nil {
			return err
		}
	}
	if err := os.Link(tempPath, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

func atomicWrite(path string, payload []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".skills-hub-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func profilePath(root, module, entryID string) (string, error) {
	parts := strings.Split(entryID, "/")
	if len(parts) != 2 || !safeSegment(parts[0]) || !safeSegment(parts[1]) || !safeSegment(module) {
		return "", errors.New("invalid authentication profile identity")
	}
	return filepath.Join(root, "auth", module, parts[0], parts[1]+".json"), nil
}

func safeSegment(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func validEnvironmentName(value string) bool {
	if value == "" || !((value[0] >= 'A' && value[0] <= 'Z') || value[0] == '_') {
		return false
	}
	for index := 1; index < len(value); index++ {
		char := value[index]
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func safeObservationValue(value string) bool {
	if len(value) > 512 {
		return false
	}
	for _, character := range value {
		if character < ' ' || character == 0x7f {
			return false
		}
	}
	return !registry.ContainsCredentialShapedValue(value)
}

func safeGeneration(value string) bool {
	return value != "" && len(value) <= 256 && safeObservationValue(value)
}

func safeCredentialBindingName(value string) bool {
	if !validEnvironmentName(value) {
		return false
	}
	upper := strings.ToUpper(value)
	if upper == "PATH" || upper == "HOME" || upper == "SHELL" || upper == "IFS" || upper == "ENV" || upper == "BASH_ENV" || upper == "PYTHONPATH" || upper == "NODE_OPTIONS" {
		return false
	}
	return !strings.HasPrefix(upper, "LD_") && !strings.HasPrefix(upper, "DYLD_")
}

func subprocessEnvironment(bindings []string) []string {
	// Readiness helpers execute inside a credential boundary. Do not inherit the
	// caller's environment: HOME and provider-specific variables can grant access
	// to ambient credential files even when no value is copied explicitly.
	allow := []string{"PATH", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "SYSTEMROOT", "WINDIR", "COMSPEC", "PATHEXT"}
	environment := make([]string, 0, len(allow)+len(bindings))
	seen := make(map[string]struct{}, len(allow)+len(bindings))
	for _, name := range allow {
		if value, ok := os.LookupEnv(name); ok {
			environment = append(environment, name+"="+value)
			seen[name] = struct{}{}
		}
	}
	for _, item := range bindings {
		name, _, found := strings.Cut(item, "=")
		if !found || !safeCredentialBindingName(name) {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			for index, existing := range environment {
				if strings.HasPrefix(existing, name+"=") {
					environment[index] = item
					break
				}
			}
			continue
		}
		environment = append(environment, item)
		seen[name] = struct{}{}
	}
	return environment
}

func bindingGenerationsCurrent(ctx context.Context, resolved resolvedAuth) bool {
	if resolved.Driver != nil && resolved.Driver.CredentialMode == "driver" {
		for name, expected := range resolved.Generations {
			var credential credentialResponse
			if err := runDriverJSON(ctx, resolved.Driver.Credential, map[string]string{"binding": name}, nil, &credential); err != nil || credential.Generation != expected {
				return false
			}
		}
	}
	for name, reference := range resolved.GenerationReferences {
		current, ok := os.LookupEnv(reference)
		if !ok || current != resolved.Generations[name] {
			return false
		}
	}
	return true
}

func sameStringSet(left, right []string) bool {
	a := sortedUnique(left)
	b := sortedUnique(right)
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func sameStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func currentPlatform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func selectedArtifactSHA256(entry registry.SkillEntry, version string) string {
	for _, candidate := range entry.Versions {
		if candidate.Version == version {
			return candidate.SHA256
		}
	}
	return ""
}

func dependencyLockSHA256(entry registry.SkillEntry, packageDir string) (string, error) {
	if entry.Artifact == nil {
		return "", errors.New("entry has no artifact contract")
	}
	if entry.Artifact.DependencyLock == nil {
		return "not-applicable", nil
	}
	relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(*entry.Artifact.DependencyLock)))
	if relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("dependency lock path is invalid")
	}
	return fileSHA256(filepath.Join(packageDir, relative))
}

func providerTargetFingerprint(observation validatorObservation, method string) (string, error) {
	if observation.Provider == "" || observation.Account == "" || observation.Principal == "" || observation.Region == "" || observation.APIVersion == "" || observation.AttestationReference == "" {
		return "", errors.New("provider target identity is incomplete")
	}
	if observation.Tier != "sandbox" && observation.Tier != "live" {
		return "", errors.New("provider target tier is invalid")
	}
	endpoint, err := url.Parse(observation.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.Fragment != "" {
		return "", errors.New("provider target endpoint is invalid")
	}
	canonical := struct {
		Provider   string   `json:"provider"`
		Tier       string   `json:"tier"`
		Endpoint   string   `json:"endpoint"`
		Region     string   `json:"region"`
		Account    string   `json:"account"`
		APIVersion string   `json:"api_version"`
		Method     string   `json:"method"`
		Principal  string   `json:"principal"`
		Scopes     []string `json:"scopes"`
	}{
		Provider: observation.Provider, Tier: observation.Tier, Endpoint: endpoint.String(),
		Region: observation.Region, Account: observation.Account, APIVersion: observation.APIVersion,
		Method: method, Principal: observation.Principal, Scopes: sortedUnique(observation.Scopes),
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func environmentID(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("state directory is required for environment identity")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("create state directory: %w", err)
	}
	path := filepath.Join(root, "environment-id")
	lock := flock.New(path + ".lock")
	if err := lock.Lock(); err != nil {
		return "", fmt.Errorf("lock environment identity: %w", err)
	}
	defer lock.Unlock()
	if payload, err := os.ReadFile(path); err == nil {
		identifier := strings.TrimSpace(string(payload))
		decoded, decodeErr := hex.DecodeString(identifier)
		if decodeErr != nil || len(decoded) != 16 {
			return "", errors.New("environment identity is malformed")
		}
		return identifier, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read environment identity: %w", err)
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate environment identity: %w", err)
	}
	identifier := hex.EncodeToString(random)
	if err := atomicWrite(path, []byte(identifier+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("persist environment identity: %w", err)
	}
	return identifier, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func missing(required, granted []string) []string {
	set := make(map[string]struct{}, len(granted))
	for _, value := range granted {
		set[value] = struct{}{}
	}
	var result []string
	for _, value := range required {
		if _, ok := set[value]; !ok {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
