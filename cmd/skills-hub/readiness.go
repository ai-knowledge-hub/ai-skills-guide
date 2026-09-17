package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/readiness"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type bindingFlag map[string]string

func (values *bindingFlag) String() string {
	if values == nil {
		return ""
	}
	items := make([]string, 0, len(*values))
	for name, reference := range *values {
		items = append(items, name+"="+reference)
	}
	return strings.Join(items, ",")
}

func (values *bindingFlag) Set(value string) error {
	name, reference, ok := strings.Cut(value, "=")
	name, reference = strings.TrimSpace(name), strings.TrimSpace(reference)
	if !ok || name == "" || reference == "" {
		return errors.New("binding must use NAME=env:VARIABLE")
	}
	if *values == nil {
		*values = make(map[string]string)
	}
	if _, exists := (*values)[name]; exists {
		return fmt.Errorf("binding %s was provided more than once", name)
	}
	(*values)[name] = reference
	return nil
}

func runAuth(args []string) error {
	if len(args) == 0 {
		return errors.New("auth requires a subcommand: configure or status")
	}
	switch args[0] {
	case "configure":
		return runAuthConfigure(args[1:])
	case "status":
		return runAuthStatus(args[1:])
	default:
		return fmt.Errorf("unsupported auth subcommand %q (supported: configure, status)", args[0])
	}
}

func runAuthConfigure(args []string) error {
	entrySpec, parsed := positionalEntry(args)
	fs := flag.NewFlagSet("auth configure", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	stateDir := fs.String("state-dir", defaultStateDir(), "non-secret authentication state directory")
	runtimeName := fs.String("runtime", "generic", "runtime adapter for packaged authentication flows")
	targetRoot := fs.String("target", "", "installed module directory for packaged authentication flows")
	method := fs.String("method", "", "declared authentication method")
	expectedAccount := fs.String("account", "", "expected provider account identifier")
	validatorCommand := fs.String("validator-command", "", "credential-free provider validation executable")
	var validatorArgs stringListFlag
	fs.Var(&validatorArgs, "validator-arg", "provider validation argument (repeatable)")
	bindings := bindingFlag{}
	fs.Var(&bindings, "binding", "runtime credential binding as NAME=env:VARIABLE (repeatable)")
	generations := bindingFlag{}
	fs.Var(&generations, "generation", "credential generation as NAME=env:GENERATION_VARIABLE (repeatable)")
	entryFlag := fs.String("entry", "", "entry id, optionally with @version")
	if err := fs.Parse(parsed); err != nil {
		return err
	}
	if strings.TrimSpace(*entryFlag) != "" {
		entrySpec = strings.TrimSpace(*entryFlag)
	}
	target, err := loadReadinessEntry(*module, *registryPath, entrySpec)
	if err != nil {
		return err
	}
	var validator *readiness.Validator
	if strings.TrimSpace(*validatorCommand) != "" {
		validator = &readiness.Validator{Command: strings.TrimSpace(*validatorCommand), Args: []string(validatorArgs)}
	}
	installed, installErr := installer.ResolveRuntimeTargetForModule(*runtimeName, target.Module, *targetRoot)
	packageDir := ""
	resolvedRuntime, resolvedTarget := "", ""
	if installErr == nil {
		packageDir = filepath.Join(installed.TargetPath, filepath.FromSlash(target.Entry.ID))
		resolvedRuntime, resolvedTarget = installed.Runtime, installed.TargetPath
	}
	profile, err := readiness.Configure(readiness.ConfigureOptions{
		StateDir: *stateDir, Module: target.Module, Entry: target.Entry, Version: target.Version,
		Method: *method, ExpectedAccount: *expectedAccount, Bindings: bindings, Generations: generations, Validator: validator,
		PackageDir: packageDir, TargetRoot: resolvedTarget, Runtime: resolvedRuntime,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Configured authentication for %s@%s\n", profile.EntryID, profile.EntryVersion)
	fmt.Printf("method: %s\n", profile.Method)
	fmt.Printf("bootstrap_mode: %s\n", profile.BootstrapMode)
	fmt.Printf("bootstrap_complete: %t\n", profile.BootstrapComplete)
	if profile.ExpectedAccount != "" {
		fmt.Printf("selected_account: %s\n", profile.ExpectedAccount)
	}
	if target.Entry.Authentication != nil {
		fmt.Printf("required_scopes: %s\n", strings.Join(target.Entry.Authentication.Scopes, ", "))
		fmt.Printf("credential_bindings: %s\n", strings.Join(target.Entry.Authentication.CredentialBindings, ", "))
	}
	if target.Entry.Authentication != nil && target.Entry.Authentication.SetupURL != "" {
		fmt.Printf("setup_url: %s\n", target.Entry.Authentication.SetupURL)
	}
	fmt.Println("credential_storage: runtime-owned references only; no credential payload was stored")
	return nil
}

func runAuthStatus(args []string) error {
	entrySpec, parsed := positionalEntry(args)
	fs := flag.NewFlagSet("auth status", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	stateDir := fs.String("state-dir", defaultStateDir(), "non-secret authentication state directory")
	entryFlag := fs.String("entry", "", "entry id, optionally with @version")
	jsonOutput := fs.Bool("json", false, "write machine-readable JSON")
	timeout := fs.Duration("timeout", 30*time.Second, "provider validation timeout")
	if err := fs.Parse(parsed); err != nil {
		return err
	}
	if strings.TrimSpace(*entryFlag) != "" {
		entrySpec = strings.TrimSpace(*entryFlag)
	}
	target, err := loadReadinessEntry(*module, *registryPath, entrySpec)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	report, err := readiness.Status(ctx, readiness.InspectOptions{
		StateDir: *stateDir, Module: target.Module, Entry: target.Entry, Version: target.Version,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else {
		printAuthReport(report)
	}
	if (report.Required || report.Selected) && !report.Authenticated {
		return fmt.Errorf("authentication is not ready: %s", report.State)
	}
	return nil
}

func runDoctor(args []string) error {
	entrySpec, parsed := positionalEntry(args)
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	stateDir := fs.String("state-dir", defaultStateDir(), "non-secret authentication state directory")
	runtimeName := fs.String("runtime", "generic", "runtime adapter: codex|claude|generic")
	targetRoot := fs.String("target", "", "installed module directory")
	entryFlag := fs.String("entry", "", "entry id, optionally with @version")
	jsonOutput := fs.Bool("json", false, "write machine-readable JSON")
	timeout := fs.Duration("timeout", 30*time.Second, "provider validation timeout")
	if err := fs.Parse(parsed); err != nil {
		return err
	}
	if strings.TrimSpace(*entryFlag) != "" {
		entrySpec = strings.TrimSpace(*entryFlag)
	}
	target, err := loadReadinessEntry(*module, *registryPath, entrySpec)
	if err != nil {
		return err
	}
	installed, err := installer.ResolveRuntimeTargetForModule(*runtimeName, target.Module, *targetRoot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	report, err := readiness.Doctor(ctx, readiness.InspectOptions{
		StateDir: *stateDir, Module: target.Module, Entry: target.Entry, Version: target.Version,
		PackageDir: filepath.Join(installed.TargetPath, filepath.FromSlash(target.Entry.ID)), TargetRoot: installed.TargetPath, Runtime: installed.Runtime,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else {
		fmt.Printf("entry: %s@%s\n", report.EntryID, report.EntryVersion)
		fmt.Printf("ready: %t\n", report.Ready)
		fmt.Printf("setup_required: %t\n", report.SetupRequired)
		fmt.Printf("expires_at: %s\n", report.ExpiresAt.UTC().Format(time.RFC3339))
		for _, check := range report.Checks {
			fmt.Printf("%s: %s", check.Name, check.Status)
			if check.Reason != "" {
				fmt.Printf(" (%s)", check.Reason)
			}
			fmt.Println()
		}
	}
	if !report.Ready {
		return errors.New("doctor found blocking readiness failures")
	}
	return nil
}

func runSmoke(args []string) error {
	entrySpec, parsed := positionalEntry(args)
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	stateDir := fs.String("state-dir", defaultStateDir(), "non-secret authentication state and evidence directory")
	runtimeName := fs.String("runtime", "generic", "runtime adapter: codex|claude|generic")
	targetRoot := fs.String("target", "", "installed module directory")
	entryFlag := fs.String("entry", "", "entry id, optionally with @version")
	timeout := fs.Duration("timeout", 2*time.Minute, "smoke test timeout")
	if err := fs.Parse(parsed); err != nil {
		return err
	}
	if strings.TrimSpace(*entryFlag) != "" {
		entrySpec = strings.TrimSpace(*entryFlag)
	}
	target, err := loadReadinessEntry(*module, *registryPath, entrySpec)
	if err != nil {
		return err
	}
	installed, err := installer.ResolveRuntimeTargetForModule(*runtimeName, target.Module, *targetRoot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	evidence, path, err := readiness.Smoke(ctx, readiness.InspectOptions{
		StateDir: *stateDir, Module: target.Module, Entry: target.Entry, Version: target.Version,
		PackageDir: filepath.Join(installed.TargetPath, filepath.FromSlash(target.Entry.ID)), TargetRoot: installed.TargetPath, Runtime: installed.Runtime,
	})
	fmt.Printf("result: %s\n", evidence.Result)
	if evidence.FailureReason != "" {
		fmt.Printf("failure_reason: %s\n", evidence.FailureReason)
	}
	if path != "" {
		fmt.Printf("evidence: %s\n", path)
	}
	return err
}

type readinessEntry struct {
	Module  string
	Entry   registry.SkillEntry
	Version string
}

func loadReadinessEntry(moduleRaw, registryPath, spec string) (readinessEntry, error) {
	module, err := normalizeModule(moduleRaw)
	if err != nil {
		return readinessEntry{}, err
	}
	if strings.TrimSpace(spec) == "" {
		return readinessEntry{}, errors.New("entry is required")
	}
	id, requestedVersion, err := parseSkillSpec(spec)
	if err != nil {
		return readinessEntry{}, err
	}
	index, err := registry.LoadIndex(resolveRegistryPath(registryPath, module))
	if err != nil {
		return readinessEntry{}, err
	}
	entry, found := registry.FindSkill(index, id)
	if !found {
		return readinessEntry{}, fmt.Errorf("entry not found in registry: %s", id)
	}
	version, err := registry.ResolveVersion(entry, requestedVersion)
	if err != nil {
		return readinessEntry{}, err
	}
	return readinessEntry{Module: module, Entry: entry, Version: version.Version}, nil
}

func positionalEntry(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}

func printAuthReport(report readiness.AuthReport) {
	fmt.Printf("entry: %s@%s\n", report.EntryID, report.EntryVersion)
	fmt.Printf("state: %s\n", report.State)
	fmt.Printf("configured: %t\n", report.Configured)
	fmt.Printf("authenticated: %t\n", report.Authenticated)
	if report.Method != "" {
		fmt.Printf("method: %s\n", report.Method)
	}
	if report.Principal != "" {
		fmt.Printf("principal: %s\n", report.Principal)
	}
	if report.Account != "" {
		fmt.Printf("account: %s\n", report.Account)
	}
	if len(report.RequiredScopes) > 0 {
		fmt.Printf("required_scopes: %s\n", strings.Join(report.RequiredScopes, ", "))
	}
	if len(report.GrantedScopes) > 0 {
		fmt.Printf("granted_scopes: %s\n", strings.Join(report.GrantedScopes, ", "))
	}
	if !report.ExpiresAt.IsZero() {
		fmt.Printf("expires_at: %s\n", report.ExpiresAt.UTC().Format(time.RFC3339))
	}
	if report.ProviderTargetFingerprint != "" {
		fmt.Printf("provider_target_fingerprint: %s\n", report.ProviderTargetFingerprint)
	}
	if report.AttestationReference != "" {
		fmt.Printf("attestation_reference: %s\n", report.AttestationReference)
	}
	if len(report.MissingBindings) > 0 {
		fmt.Printf("missing_bindings: %s\n", strings.Join(report.MissingBindings, ", "))
	}
	if len(report.MissingScopes) > 0 {
		fmt.Printf("missing_scopes: %s\n", strings.Join(report.MissingScopes, ", "))
	}
	if report.Reason != "" {
		fmt.Printf("reason: %s\n", report.Reason)
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func defaultStateDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", ".skills-hub")
	}
	return filepath.Join(base, "skills-hub")
}
