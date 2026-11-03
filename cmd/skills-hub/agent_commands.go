//go:build runtime_readiness

package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/agents"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
)

func runAgent(args []string) error {
	fs := flag.NewFlagSet("run-agent", flag.ContinueOnError)
	agentID := fs.String("agent", "", "agent id in <domain>/<slug> format")
	runtimeName := fs.String("runtime", "generic", "compiled runtime binding set (codex, claude, generic)")
	stateDir := fs.String("state-dir", defaultStateDir(), "non-secret provider authentication state directory")
	agentsRoot := fs.String("agents-root", "", "agents root directory override")
	skillsRoot := fs.String("skills-root", "", "skills root directory override")
	toolsRoot := fs.String("tools-root", "", "tools root directory override")
	modelAttestationPath := fs.String("model-attestation", "", "path to a signed runtime model attestation")
	bindingsPath := fs.String("bindings", "", "path to JSON tool bindings")
	memoryPath := fs.String("memory", "", "path to JSON memory profile")
	governancePath := fs.String("governance", "", "path to JSON governance profile")
	auditPath := fs.String("audit-log", "", "path to output JSON audit report")
	approveLive := fs.Bool("approve-live", false, "approve live actions for this run")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*agentID) == "" {
		return errors.New("--agent is required")
	}
	resolvedAgentsRoot, resolvedSkillsRoot, resolvedToolsRoot, err := resolveRunAgentRoots(*runtimeName, *agentsRoot, *skillsRoot, *toolsRoot)
	if err != nil {
		return err
	}

	report, err := agents.RunPreflight(agents.RunOptions{
		AgentsRoot:           resolvedAgentsRoot,
		SkillsRoot:           resolvedSkillsRoot,
		ToolsRoot:            resolvedToolsRoot,
		AgentID:              strings.TrimSpace(*agentID),
		Runtime:              strings.TrimSpace(*runtimeName),
		StateDir:             strings.TrimSpace(*stateDir),
		BindingsPath:         strings.TrimSpace(*bindingsPath),
		MemoryPath:           strings.TrimSpace(*memoryPath),
		GovernancePath:       strings.TrimSpace(*governancePath),
		ApproveLive:          *approveLive,
		AuditPath:            strings.TrimSpace(*auditPath),
		ModelAttestationPath: strings.TrimSpace(*modelAttestationPath),
	})
	if err != nil {
		return err
	}

	fmt.Printf("agent: %s\n", report.AgentID)
	fmt.Printf("status: %s\n", report.Status)
	if len(report.WorkflowSteps) > 0 {
		fmt.Printf("workflow_steps: %d\n", len(report.WorkflowSteps))
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, w := range report.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	if len(report.BlockingReasons) > 0 {
		fmt.Println("blocking_reasons:")
		for _, b := range report.BlockingReasons {
			fmt.Printf("  - %s\n", b)
		}
	}

	if report.Status != "ready" {
		return fmt.Errorf("agent preflight not ready: %s", report.Status)
	}
	return nil
}

func resolveRunAgentRoots(runtimeName, agentsRoot, skillsRoot, toolsRoot string) (string, string, string, error) {
	resolve := func(module, explicit string) (string, error) {
		target, err := installer.ResolveRuntimeTargetForModule(runtimeName, module, strings.TrimSpace(explicit))
		if err != nil {
			return "", fmt.Errorf("resolve %s root: %w", module, err)
		}
		return target.TargetPath, nil
	}
	agentsPath, err := resolve("agents", agentsRoot)
	if err != nil {
		return "", "", "", err
	}
	skillsPath, err := resolve("skills", skillsRoot)
	if err != nil {
		return "", "", "", err
	}
	toolsPath, err := resolve("tools", toolsRoot)
	if err != nil {
		return "", "", "", err
	}
	return agentsPath, skillsPath, toolsPath, nil
}
