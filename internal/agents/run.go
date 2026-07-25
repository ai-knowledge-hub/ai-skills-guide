package agents

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RunOptions struct {
	AgentsRoot     string
	SkillsRoot     string
	ToolsRoot      string
	AgentID        string
	BindingsPath   string
	MemoryPath     string
	GovernancePath string
	ApproveLive    bool
	AuditPath      string
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
	if strings.TrimSpace(opts.BindingsPath) != "" {
		if err := loadJSON(opts.BindingsPath, &bindings); err != nil {
			return RunReport{}, fmt.Errorf("load bindings: %w", err)
		}
	} else {
		report.Warnings = append(report.Warnings, "No tool bindings file supplied")
	}

	for _, depTool := range m.DependencyTools {
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
		report.ResolvedTools[depTool] = binding
		if binding.Mode == "" {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Tool %s missing mode; expected read_only/read_write", depTool))
		}
		if strings.TrimSpace(binding.Endpoint) == "" {
			report.Status = "blocked"
			report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Tool %s missing endpoint", depTool))
		}
	}

	var memory MemoryProfile
	if strings.TrimSpace(opts.MemoryPath) != "" {
		if err := loadJSON(opts.MemoryPath, &memory); err != nil {
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
	if strings.TrimSpace(opts.GovernancePath) != "" {
		if err := loadJSON(opts.GovernancePath, &governance); err != nil {
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
				report.Status = "blocked"
				report.BlockingReasons = append(report.BlockingReasons, fmt.Sprintf("Write-capable tool blocked by governance: %s", name))
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
