package agents

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/readiness"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type RunOptions struct {
	AgentsRoot           string
	SkillsRoot           string
	ToolsRoot            string
	AgentID              string
	Runtime              string
	StateDir             string
	BindingsPath         string
	MemoryPath           string
	GovernancePath       string
	ApproveLive          bool
	AuditPath            string
	ModelAttestationPath string
}

type compiledAgentContract struct {
	SchemaVersion   string             `json:"schema_version"`
	Runtime         string             `json:"runtime"`
	Agent           compiledAgentRoot  `json:"agent"`
	Status          string             `json:"status"`
	BlockingReasons []string           `json:"blocking_reasons"`
	Model           ModelRequirement   `json:"model"`
	Memory          ProfileRequirement `json:"memory"`
	Governance      ProfileRequirement `json:"governance"`
	Bindings        []compiledBinding  `json:"bindings"`
}

type modelAttestationClaims struct {
	SchemaVersion          string    `json:"schema_version"`
	Runtime                string    `json:"runtime"`
	AgentID                string    `json:"agent_id"`
	AgentVersion           string    `json:"agent_version"`
	RuntimeContractSHA256  string    `json:"runtime_contract_sha256"`
	CompiledContractSHA256 string    `json:"compiled_contract_sha256"`
	ModelIdentity          string    `json:"model_identity"`
	ModelVersion           string    `json:"model_version"`
	Capabilities           []string  `json:"capabilities"`
	IssuedAt               time.Time `json:"issued_at"`
	ExpiresAt              time.Time `json:"expires_at"`
}

type modelAttestationDocument struct {
	modelAttestationClaims
	Signature string `json:"signature"`
}

type compiledAgentRoot struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type compiledBinding struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Package       string `json:"package,omitempty"`
	Version       string `json:"version,omitempty"`
	Requirement   string `json:"requirement"`
	Access        string `json:"access,omitempty"`
	Status        string `json:"status"`
	Path          string `json:"path,omitempty"`
	Endpoint      string `json:"endpoint,omitempty"`
	Reason        string `json:"reason,omitempty"`
	ConfigureHint string `json:"configure_hint,omitempty"`
}

func RunPreflight(opts RunOptions) (RunReport, error) {
	agentDir := filepath.Join(opts.AgentsRoot, filepath.FromSlash(opts.AgentID))
	manifestPath := filepath.Join(agentDir, "agent.yaml")
	specPath := filepath.Join(agentDir, "AGENT.md")

	m, err := ParseAgentManifest(manifestPath)
	if err != nil {
		return RunReport{}, err
	}
	report := RunReport{
		AgentID:          m.ID,
		Status:           "ready",
		Checks:           []string{},
		Warnings:         []string{},
		BlockingReasons:  []string{},
		DependencyAgents: append([]string{}, m.DependencyAgents...),
		DependencySkills: append([]string{}, m.DependencySkills...),
		DependencyTools:  append([]string{}, m.DependencyTools...),
		ResolvedTools:    map[string]ToolBinding{},
		Model:            ModelReadiness{Status: "not-applicable"},
	}
	var compiled *compiledAgentContract
	var runtimeContractDigest string
	orchestrationIntegrityOK := true
	if m.Orchestration != nil {
		runtimeName := strings.ToLower(strings.TrimSpace(opts.Runtime))
		if runtimeName == "" {
			runtimeName = "generic"
		}
		installedManifest, manifestErr := registry.ValidateHistoricalPackageManifest(manifestPath)
		if manifestErr != nil {
			orchestrationIntegrityOK = false
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, "Installed agent manifest is not valid for receipt verification")
		} else {
			entry := registry.ProjectManifest(installedManifest)
			contractDigest, digestErr := installer.RuntimeContractSHA256(entry, installedManifest.Version)
			if digestErr != nil {
				return RunReport{}, fmt.Errorf("derive agent runtime contract digest: %w", digestErr)
			}
			runtimeContractDigest = contractDigest
			receipt, receiptErr := installer.VerifyInstallReceipt(agentDir, "agents", m.ID, m.Version, runtimeName, "", contractDigest)
			if receiptErr != nil {
				orchestrationIntegrityOK = false
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, "Installed agent receipt verification failed; reinstall the agent")
			} else if closureErr := installer.VerifyAgentInstallClosure(receipt, map[string]string{
				"agents": opts.AgentsRoot, "skills": opts.SkillsRoot, "tools": opts.ToolsRoot,
			}, entry.Orchestration); closureErr != nil {
				orchestrationIntegrityOK = false
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, "Installed agent dependency closure verification failed; reinstall the agent")
			}
		}
		compiledPath := filepath.Join(agentDir, ".runtime", runtimeName+".json")
		loaded, loadErr := loadCompiledAgentContract(compiledPath)
		if loadErr != nil {
			orchestrationIntegrityOK = false
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, "Compiled runtime contract unavailable or malformed; reinstall the agent for this runtime")
		} else {
			report.Model = ModelReadiness{
				Selection: loaded.Model.Selection, Status: "unverified",
				RequiredCapabilities: append([]string(nil), loaded.Model.RequiredCapabilities...),
			}
			integrityReasons := compiledAgentContractIntegrityReasons(loaded, m, opts, runtimeName)
			modelReady := false
			modelReadinessEvaluated := false
			var modelReasons []string
			if len(integrityReasons) == 0 && orchestrationIntegrityOK {
				modelReadinessEvaluated = true
				model, modelErr := verifyModelAttestation(opts.ModelAttestationPath, loaded, compiledPath, runtimeContractDigest, time.Now().UTC())
				if modelErr != nil {
					modelReasons = append(modelReasons, "Runtime model readiness is unverified: "+modelErr.Error())
				} else {
					report.Model = model
					modelReady = true
				}
			}
			readinessReasons := append(modelReasons, compiledAgentContractReadinessReasons(loaded, m, opts, modelReady || modelReadinessEvaluated)...)
			for _, reason := range append(append([]string(nil), integrityReasons...), readinessReasons...) {
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, reason)
			}
			if len(integrityReasons) > 0 {
				orchestrationIntegrityOK = false
			}
			if orchestrationIntegrityOK {
				compiled = &loaded
			}
		}
	}

	if _, err := os.Stat(specPath); err != nil {
		report.Status = "blocked"
		report.BlockingReasons = append(report.BlockingReasons, "Missing AGENT.md")
	}

	report.WorkflowSteps = extractWorkflowSteps(specPath)
	if len(report.WorkflowSteps) == 0 {
		report.Warnings = append(report.Warnings, "No workflow steps parsed from AGENT.md")
	}

	for _, dep := range m.DependencySkills {
		p := filepath.Join(opts.SkillsRoot, filepath.FromSlash(dep))
		if _, err := os.Stat(p); err != nil {
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Missing skill dependency: %s", dep))
		}
	}
	for _, dep := range m.DependencyAgents {
		p := filepath.Join(opts.AgentsRoot, filepath.FromSlash(dep))
		if _, err := os.Stat(p); err != nil {
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Missing agent dependency: %s", dep))
		}
	}

	bindings := ToolBindingsFile{Tools: map[string]ToolBinding{}}
	if compiled != nil {
		bindings = compiledToolBindings(*compiled)
	}
	if strings.TrimSpace(opts.BindingsPath) != "" {
		var overrides ToolBindingsFile
		if err := loadJSON(opts.BindingsPath, &overrides); err != nil {
			return RunReport{}, fmt.Errorf("load bindings: %w", err)
		}
		for name, override := range overrides.Tools {
			base, expected := bindings.Tools[name]
			if m.Orchestration != nil && !expected {
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Unexpected tool binding override: %s", name))
				continue
			}
			if expected {
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Tool %s is receipt-bound; compiled binding overrides are not permitted", name))
				bindings.Tools[name] = base
			} else {
				bindings.Tools[name] = override
			}
		}
	} else {
		if m.Orchestration == nil {
			report.Warnings = append(report.Warnings, "No tool bindings file supplied")
		}
	}

	toolNames := append([]string(nil), m.DependencyTools...)
	if m.Orchestration != nil {
		toolNames = toolNames[:0]
		for _, binding := range m.Orchestration.Bindings {
			if binding.Kind == "tool" {
				toolNames = append(toolNames, binding.Name)
			}
		}
	}
	for _, depTool := range toolNames {
		if strings.Contains(depTool, "/") {
			p := filepath.Join(opts.ToolsRoot, filepath.FromSlash(depTool))
			if _, err := os.Stat(p); err != nil {
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Missing tool dependency: %s", depTool))
			}
		}
		binding, ok := bindings.Tools[depTool]
		if !ok {
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Missing tool binding: %s", depTool))
			continue
		}
		binding = refreshAuthenticationStatus(opts, binding)
		report.ResolvedTools[depTool] = binding
		if blocking, warning := toolBindingReadiness(depTool, binding); blocking != "" {
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, blocking)
		} else if warning != "" {
			report.Warnings = append(report.Warnings, warning)
		}
		if binding.Mode == "" {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Tool %s missing mode; expected read_only/read_write", depTool))
		}
	}

	var memory MemoryProfile
	memoryPath := strings.TrimSpace(opts.MemoryPath)
	if memoryPath == "" && m.Orchestration != nil && m.Orchestration.Memory.Mode == "bundled" {
		memoryPath = filepath.Join(agentDir, filepath.FromSlash(m.Orchestration.Memory.Path))
	}
	if memoryPath != "" {
		if err := loadJSON(memoryPath, &memory); err != nil {
			return RunReport{}, fmt.Errorf("load memory profile: %w", err)
		}
		for _, f := range memory.ContextFiles {
			if _, err := os.Stat(f); err != nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Missing memory context file: %s", f))
			}
		}
	} else {
		report.Warnings = append(report.Warnings, "No memory profile supplied")
	}

	governance := GovernanceProfile{
		RequireApprovalForLive: true,
		AllowWriteTools:        false,
		MaxToolCalls:           100,
	}
	governancePath := strings.TrimSpace(opts.GovernancePath)
	if governancePath == "" && m.Orchestration != nil && m.Orchestration.Governance.Mode == "bundled" {
		governancePath = filepath.Join(agentDir, filepath.FromSlash(m.Orchestration.Governance.Path))
	}
	if governancePath != "" {
		if err := loadJSON(governancePath, &governance); err != nil {
			return RunReport{}, fmt.Errorf("load governance profile: %w", err)
		}
	} else {
		report.Warnings = append(report.Warnings, "No governance profile supplied; using safe defaults")
	}

	if governance.RequireApprovalForLive && !opts.ApproveLive {
		report.Status = "blocked"
		report.BlockingReasons = append(report.BlockingReasons, "Live approval is required (--approve-live not set)")
	}
	if !governance.AllowWriteTools {
		for name, binding := range report.ResolvedTools {
			if strings.EqualFold(binding.Mode, "read_write") {
				if binding.Requirement == "optional" {
					report.Warnings = append(report.Warnings, fmt.Sprintf("Optional write-capable tool disabled by governance: %s", name))
				} else {
					report.Status = "blocked"
					report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Write-capable tool blocked by governance: %s", name))
				}
			}
		}
	}

	if len(report.BlockingReasons) == 0 {
		report.Checks = append(report.Checks, "All preflight checks passed")
	}
	if strings.TrimSpace(opts.AuditPath) != "" {
		if err := writeJSON(opts.AuditPath, report); err != nil {
			return RunReport{}, fmt.Errorf("write audit report: %w", err)
		}
	}
	return report, nil
}

func loadCompiledAgentContract(path string) (compiledAgentContract, error) {
	var document compiledAgentContract
	data, err := os.ReadFile(path)
	if err != nil {
		return document, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return document, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return document, fmt.Errorf("compiled runtime contract contains trailing data")
	}
	return document, nil
}

func compiledToolBindings(document compiledAgentContract) ToolBindingsFile {
	result := ToolBindingsFile{Tools: map[string]ToolBinding{}}
	for _, binding := range document.Bindings {
		if binding.Kind != "tool" {
			continue
		}
		mode := strings.ReplaceAll(binding.Access, "-", "_")
		result.Tools[binding.Name] = ToolBinding{Package: binding.Package, Version: binding.Version, Requirement: binding.Requirement, Endpoint: binding.Endpoint, Mode: mode, Status: binding.Status, Reason: binding.Reason, ConfigureHint: binding.ConfigureHint}
	}
	return result
}

func validateCompiledAgentContract(document compiledAgentContract, manifest Manifest, opts RunOptions, runtimeName string) []string {
	reasons := compiledAgentContractIntegrityReasons(document, manifest, opts, runtimeName)
	return append(reasons, compiledAgentContractReadinessReasons(document, manifest, opts, false)...)
}

func compiledAgentContractIntegrityReasons(document compiledAgentContract, manifest Manifest, opts RunOptions, runtimeName string) []string {
	var reasons []string
	if document.SchemaVersion != "skills-hub.agent-runtime/v1" || document.Runtime != runtimeName || document.Agent.ID != manifest.ID || document.Agent.Version != manifest.Version {
		return []string{"Compiled runtime contract does not match the selected agent and runtime"}
	}
	if document.Status != "ready" && document.Status != "setup-required" && document.Status != "blocked" {
		reasons = append(reasons, "Compiled runtime contract has an invalid top-level status")
	}
	if manifest.Orchestration == nil {
		return append(reasons, "Installed agent has no orchestration contract")
	}
	if !reflect.DeepEqual(document.Model, manifest.Orchestration.Model) || !reflect.DeepEqual(document.Memory, manifest.Orchestration.Memory) || !reflect.DeepEqual(document.Governance, manifest.Orchestration.Governance) {
		reasons = append(reasons, "Compiled runtime model or profile requirements do not match the installed manifest")
	}
	expected := make(map[string]PackageBinding, len(manifest.Orchestration.Bindings))
	for _, binding := range manifest.Orchestration.Bindings {
		expected[binding.Name] = binding
	}
	if len(document.Bindings) != len(expected) {
		reasons = append(reasons, "Compiled runtime contract does not contain the exact manifest binding set")
	}
	seen := map[string]bool{}
	for _, binding := range document.Bindings {
		want, ok := expected[binding.Name]
		if !ok || seen[binding.Name] {
			reasons = append(reasons, fmt.Sprintf("Compiled runtime contract contains unexpected or duplicate binding: %s", binding.Name))
			continue
		}
		seen[binding.Name] = true
		if binding.Kind != want.Kind || binding.Package != want.Package || binding.Version != want.Version || binding.Requirement != want.Requirement || binding.Access != want.Access {
			reasons = append(reasons, fmt.Sprintf("Compiled binding %s does not match the installed manifest", binding.Name))
			continue
		}
		validStatus := binding.Status == "ready" || binding.Status == "setup-required" || binding.Status == "not-ready" || binding.Status == "missing" || binding.Status == "unavailable"
		if !validStatus {
			reasons = append(reasons, fmt.Sprintf("Compiled binding %s has invalid status", binding.Name))
		}
		if want.Availability == "unavailable" {
			if binding.Status != "unavailable" {
				reasons = append(reasons, fmt.Sprintf("Compiled binding %s loses the manifest unavailable state", binding.Name))
			}
		} else {
			root := map[string]string{"agent": opts.AgentsRoot, "skill": opts.SkillsRoot, "tool": opts.ToolsRoot}[binding.Kind]
			wantPath := filepath.Clean(filepath.Join(root, filepath.FromSlash(binding.Package)))
			if filepath.Clean(binding.Path) != wantPath {
				reasons = append(reasons, fmt.Sprintf("Compiled binding %s points outside its installed package", binding.Name))
			}
		}
		if binding.Kind == "tool" && binding.Status == "ready" && strings.TrimSpace(binding.Endpoint) == "" {
			reasons = append(reasons, fmt.Sprintf("Ready compiled tool binding %s has no receipt-bound endpoint", binding.Name))
		}
	}
	return reasons
}

func compiledAgentContractReadinessReasons(document compiledAgentContract, manifest Manifest, opts RunOptions, modelReady bool) []string {
	var reasons []string
	if document.Status == "blocked" {
		if len(document.BlockingReasons) == 0 {
			reasons = append(reasons, "Compiled runtime contract is blocked")
		} else {
			for _, reason := range document.BlockingReasons {
				if modelReady && reason == "model: trusted runtime adapter attestation is required before execution" {
					continue
				}
				reasons = append(reasons, "Compiled runtime contract is blocked: "+reason)
			}
		}
	}
	if len(document.Model.RequiredCapabilities) > 0 && !modelReady {
		reasons = append(reasons, "Runtime model readiness is unverified: no trusted runtime adapter supplied a model-bound capability attestation")
	}
	for label, profile := range map[string]ProfileRequirement{"memory": document.Memory, "governance": document.Governance} {
		if profile.Required && profile.Mode == "runtime" {
			provided := opts.MemoryPath
			if label == "governance" {
				provided = opts.GovernancePath
			}
			if strings.TrimSpace(provided) == "" {
				reasons = append(reasons, fmt.Sprintf("Required runtime %s profile is unresolved", label))
			}
		}
	}
	if manifest.Orchestration != nil {
		for _, binding := range document.Bindings {
			if binding.Requirement == "required" && binding.Kind != "tool" && binding.Status != "ready" {
				reasons = append(reasons, fmt.Sprintf("Required compiled %s binding %s is unresolved", binding.Kind, binding.Name))
			}
		}
	}
	return reasons
}

func verifyModelAttestation(path string, contract compiledAgentContract, compiledPath, runtimeContractDigest string, now time.Time) (ModelReadiness, error) {
	if len(contract.Model.RequiredCapabilities) == 0 {
		return ModelReadiness{Selection: contract.Model.Selection, Status: "not-applicable"}, nil
	}
	publicKey, trustErr := governedRuntimeModelTrustRoot(contract.Runtime)
	if trustErr != nil {
		return ModelReadiness{}, trustErr
	}
	if strings.TrimSpace(path) == "" {
		return ModelReadiness{}, fmt.Errorf("--model-attestation is required")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > 64<<10 {
		return ModelReadiness{}, fmt.Errorf("the model attestation is unavailable or exceeds 64 KiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ModelReadiness{}, fmt.Errorf("the model attestation cannot be read")
	}
	var document modelAttestationDocument
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return ModelReadiness{}, fmt.Errorf("the model attestation is malformed")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ModelReadiness{}, fmt.Errorf("the model attestation contains trailing data")
	}
	compiledBytes, err := os.ReadFile(compiledPath)
	if err != nil {
		return ModelReadiness{}, fmt.Errorf("the compiled runtime contract cannot be authenticated")
	}
	compiledDigest := sha256.Sum256(compiledBytes)
	if document.SchemaVersion != "skills-hub.model-attestation/v1" ||
		document.Runtime != contract.Runtime || document.AgentID != contract.Agent.ID || document.AgentVersion != contract.Agent.Version ||
		document.RuntimeContractSHA256 != runtimeContractDigest || document.CompiledContractSHA256 != hex.EncodeToString(compiledDigest[:]) {
		return ModelReadiness{}, fmt.Errorf("the model attestation is not bound to this installed agent and runtime contract")
	}
	if document.IssuedAt.IsZero() || document.ExpiresAt.IsZero() || document.ExpiresAt.Before(now) || document.ExpiresAt.Equal(now) || document.IssuedAt.After(now.Add(5*time.Minute)) || !document.ExpiresAt.After(document.IssuedAt) || document.ExpiresAt.Sub(document.IssuedAt) > 24*time.Hour {
		return ModelReadiness{}, fmt.Errorf("the model attestation is expired or has an invalid validity window")
	}
	if !validAttestationToken(document.ModelIdentity, 256) || !validAttestationToken(document.ModelVersion, 128) || len(document.Capabilities) == 0 || len(document.Capabilities) > 64 {
		return ModelReadiness{}, fmt.Errorf("the model attestation identity, version, or capabilities are invalid")
	}
	capabilities := make(map[string]bool, len(document.Capabilities))
	for _, capability := range document.Capabilities {
		if !validAttestationToken(capability, 128) || capabilities[capability] {
			return ModelReadiness{}, fmt.Errorf("the model attestation contains invalid or duplicate capabilities")
		}
		capabilities[capability] = true
	}
	for _, required := range contract.Model.RequiredCapabilities {
		if !capabilities[required] {
			return ModelReadiness{}, fmt.Errorf("the model attestation does not satisfy every required capability")
		}
	}
	signature, signatureErr := base64.StdEncoding.DecodeString(document.Signature)
	claims, marshalErr := json.Marshal(document.modelAttestationClaims)
	if signatureErr != nil || len(signature) != ed25519.SignatureSize || marshalErr != nil || !ed25519.Verify(publicKey, claims, signature) {
		return ModelReadiness{}, fmt.Errorf("the model attestation signature is invalid")
	}
	evidenceDigest := sha256.Sum256(data)
	return ModelReadiness{
		Selection: contract.Model.Selection, Status: "ready", Identity: document.ModelIdentity, Version: document.ModelVersion,
		EvidenceReference: "sha256:" + hex.EncodeToString(evidenceDigest[:]), RequiredCapabilities: append([]string(nil), contract.Model.RequiredCapabilities...),
		AttestedCapabilities: append([]string(nil), document.Capabilities...),
	}, nil
}

func validAttestationToken(value string, max int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > max {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("._:/@+-", character) {
			continue
		}
		return false
	}
	return true
}

func toolBindingReadiness(name string, binding ToolBinding) (blocking, warning string) {
	reason := binding.Reason
	if reason == "" && binding.ConfigureHint != "" {
		reason = binding.Status + "; next: " + binding.ConfigureHint
	} else if reason == "" {
		reason = binding.Status
	}
	if binding.Status != "" && binding.Status != "ready" {
		if binding.Requirement == "optional" {
			return "", fmt.Sprintf("Optional tool %s is not ready: %s", name, reason)
		}
		return fmt.Sprintf("Tool %s is not ready: %s", name, reason), ""
	}
	if strings.TrimSpace(binding.Endpoint) == "" {
		if binding.Requirement == "optional" {
			return "", fmt.Sprintf("Optional tool %s missing endpoint", name)
		}
		return fmt.Sprintf("Tool %s missing endpoint", name), ""
	}
	return "", ""
}

func refreshAuthenticationStatus(opts RunOptions, binding ToolBinding) ToolBinding {
	if binding.Status != "setup-required" || binding.Package == "" || binding.Version == "" || strings.TrimSpace(opts.StateDir) == "" {
		return binding
	}
	toolDir := filepath.Join(opts.ToolsRoot, filepath.FromSlash(binding.Package))
	manifest, err := registry.ValidatePackageManifest(filepath.Join(toolDir, "tool.yaml"))
	if err != nil || manifest.ID != binding.Package || manifest.Version != binding.Version {
		binding.Reason = "installed provider contract is unavailable, expired, or does not match the pinned binding"
		return binding
	}
	entry := registry.ProjectManifest(manifest)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report, statusErr := readiness.Status(ctx, readiness.InspectOptions{
		StateDir: opts.StateDir, Module: "tools", Entry: entry, Version: binding.Version,
		PackageDir: toolDir, TargetRoot: opts.ToolsRoot, Runtime: opts.Runtime, Now: time.Now().UTC(),
	})
	if statusErr == nil && report.Authenticated && report.EntryID == binding.Package && report.EntryVersion == binding.Version {
		binding.Status, binding.Reason = "ready", ""
		return binding
	}
	if report.Reason != "" {
		binding.Reason = report.Reason
	}
	return binding
}

func extractWorkflowSteps(agentSpecPath string) []string {
	f, err := os.Open(agentSpecPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	steps := []string{}
	scanner := bufio.NewScanner(f)
	inWorkflow := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inWorkflow = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), "Workflow")
			continue
		}
		if !inWorkflow {
			continue
		}
		if strings.HasPrefix(trimmed, "1.") ||
			strings.HasPrefix(trimmed, "2.") ||
			strings.HasPrefix(trimmed, "3.") ||
			strings.HasPrefix(trimmed, "4.") ||
			strings.HasPrefix(trimmed, "5.") ||
			strings.HasPrefix(trimmed, "6.") ||
			strings.HasPrefix(trimmed, "7.") ||
			strings.HasPrefix(trimmed, "8.") ||
			strings.HasPrefix(trimmed, "9.") {
			steps = append(steps, strings.TrimSpace(trimmed[2:]))
		}
	}
	return steps
}

func loadJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
