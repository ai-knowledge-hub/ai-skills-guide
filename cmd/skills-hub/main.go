package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/agents"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/skills"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "list":
		err = runList(args)
	case "search":
		err = runSearch(args)
	case "info":
		err = runInfo(args)
	case "validate":
		err = runValidate(args)
	case "install":
		err = runInstall(args)
	case "run-agent":
		err = runAgent(args)
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	moduleName, err := normalizeModule(*module)
	if err != nil {
		return err
	}

	idx, err := registry.LoadIndex(resolveRegistryPath(*registryPath, moduleName))
	if err != nil {
		return err
	}
	for _, s := range idx.Skills {
		status := "active"
		if s.Deprecated {
			status = "deprecated"
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", s.ID, status, s.Latest, s.Name)
	}
	return nil
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	text := fs.String("q", "", "free text search against id/name/description")
	tag := fs.String("tag", "", "filter by tag")
	category := fs.String("category", "", "filter by category")
	runtime := fs.String("runtime", "", "filter by runtime")
	if err := fs.Parse(args); err != nil {
		return err
	}
	moduleName, err := normalizeModule(*module)
	if err != nil {
		return err
	}

	idx, err := registry.LoadIndex(resolveRegistryPath(*registryPath, moduleName))
	if err != nil {
		return err
	}
	results := registry.Search(idx, registry.SearchQuery{
		Text:     *text,
		Tag:      *tag,
		Category: *category,
		Runtime:  *runtime,
	})
	for _, s := range results {
		fmt.Printf("%s\t%s\t%s\n", s.ID, s.Category, s.Latest)
	}
	fmt.Printf("Found %d skill(s).\n", len(results))
	return nil
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	entriesRoot := fs.String("root", "", "module root directory (defaults by module)")
	skillSpec := fs.String("skill", "", "skill id, optionally with @version")
	entrySpec := fs.String("entry", "", "entry id, optionally with @version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	moduleName, err := normalizeModule(*module)
	if err != nil {
		return err
	}
	selectedSpec := strings.TrimSpace(*entrySpec)
	if selectedSpec == "" {
		selectedSpec = strings.TrimSpace(*skillSpec)
	}
	if selectedSpec == "" {
		return errors.New("--entry is required (or use legacy --skill)")
	}

	id, requestedVersion, err := parseSkillSpec(selectedSpec)
	if err != nil {
		return err
	}

	idx, err := registry.LoadIndex(resolveRegistryPath(*registryPath, moduleName))
	if err != nil {
		return err
	}
	skill, ok := registry.FindSkill(idx, id)
	if !ok {
		return fmt.Errorf("entry not found in registry: %s", id)
	}
	resolvedVersion, err := registry.ResolveVersion(skill, requestedVersion)
	if err != nil {
		return err
	}

	versions := make([]string, 0, len(skill.Versions))
	for _, v := range skill.Versions {
		versions = append(versions, v.Version)
	}
	sort.Strings(versions)

	fmt.Printf("id: %s\n", skill.ID)
	fmt.Printf("path: %s\n", filepath.Join(resolveModuleRoot(*entriesRoot, moduleName), filepath.FromSlash(skill.ID)))
	fmt.Printf("name: %s\n", skill.Name)
	fmt.Printf("description: %s\n", skill.Description)
	fmt.Printf("category: %s\n", skill.Category)
	fmt.Printf("latest: %s\n", skill.Latest)
	fmt.Printf("selected_version: %s\n", resolvedVersion.Version)
	fmt.Printf("versions: %s\n", strings.Join(versions, ", "))
	fmt.Printf("runtimes: %s\n", strings.Join(skill.Runtimes, ", "))
	fmt.Printf("deprecated: %t\n", skill.Deprecated)
	if skill.ReplacedBy != "" {
		fmt.Printf("replaced_by: %s\n", skill.ReplacedBy)
	}
	return nil
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	root := fs.String("root", "skills", "skills root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := skills.Validate(*root)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		fmt.Println("All skills passed validation.")
		return nil
	}

	for _, issue := range issues {
		fmt.Printf("[ERROR] %s: %s\n", issue.SkillID, issue.Message)
	}
	return fmt.Errorf("validation failed with %d issue(s)", len(issues))
}

func runInstall(args []string) error {
	positionalSpec := ""
	parsedArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positionalSpec = args[0]
		parsedArgs = args[1:]
	}

	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools")
	entriesRoot := fs.String("root", "", "module root directory (defaults by module)")
	registryPath := fs.String("registry", "", "registry index path (defaults by module)")
	runtimeName := fs.String("runtime", "generic", "runtime adapter: codex|claude|generic")
	target := fs.String("target", "", "destination module directory (optional for codex/claude)")
	skillSpecFlag := fs.String("skill", "", "skill id, optionally with @version")
	entrySpecFlag := fs.String("entry", "", "entry id, optionally with @version")
	force := fs.Bool("force", false, "overwrite destination if it already exists")
	if err := fs.Parse(parsedArgs); err != nil {
		return err
	}
	moduleName, err := normalizeModule(*module)
	if err != nil {
		return err
	}

	skillSpec := strings.TrimSpace(*entrySpecFlag)
	if skillSpec == "" {
		skillSpec = strings.TrimSpace(*skillSpecFlag)
	}
	if skillSpec == "" {
		skillSpec = positionalSpec
	}
	if skillSpec == "" {
		return errors.New("entry is required (use --entry <id[@version]>, --skill, or positional <id[@version]>)")
	}

	id, requestedVersion, err := parseSkillSpec(skillSpec)
	if err != nil {
		return err
	}

	idx, err := registry.LoadIndex(resolveRegistryPath(*registryPath, moduleName))
	if err != nil {
		return err
	}
	skill, ok := registry.FindSkill(idx, id)
	if !ok {
		return fmt.Errorf("entry not found in registry: %s", id)
	}
	resolvedVersion, err := registry.ResolveVersion(skill, requestedVersion)
	if err != nil {
		return err
	}

	if skill.Deprecated {
		fmt.Fprintf(os.Stderr, "warning: %s is deprecated", skill.ID)
		if skill.ReplacedBy != "" {
			fmt.Fprintf(os.Stderr, "; prefer %s", skill.ReplacedBy)
		}
		fmt.Fprintln(os.Stderr)
	}

	sourceDir := filepath.Join(resolveModuleRoot(*entriesRoot, moduleName), filepath.FromSlash(skill.ID))
	if stat, statErr := os.Stat(sourceDir); statErr != nil || !stat.IsDir() {
		return fmt.Errorf("local source not found for %s at %s", skill.ID, sourceDir)
	}

	rt, err := installer.ResolveRuntimeTargetForModule(*runtimeName, moduleName, *target)
	if err != nil {
		return err
	}
	destination, err := installer.InstallSkill(sourceDir, rt.TargetPath, skill.ID, *force)
	if err != nil {
		return err
	}

	fmt.Printf("Installed %s@%s to %s (runtime=%s)\n", skill.ID, resolvedVersion.Version, destination, rt.Runtime)
	return nil
}

func parseSkillSpec(spec string) (string, string, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return "", "", errors.New("empty skill spec")
	}

	parts := strings.SplitN(trimmed, "@", 2)
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", "", fmt.Errorf("invalid skill spec: %s", spec)
	}
	version := "latest"
	if len(parts) == 2 {
		version = strings.TrimSpace(parts[1])
		if version == "" {
			return "", "", fmt.Errorf("invalid version in skill spec: %s", spec)
		}
	}
	return id, version, nil
}

func printUsage() {
	lines := []string{
		"skills-hub: manage local marketing/adtech skill, agent, and tool packages",
		"",
		"Usage:",
		"  skills-hub <command> [flags]",
		"",
		"Commands:",
		"  list      List entries from module registry index",
		"  search    Search entries by text/tag/category/runtime",
		"  info      Show details for one entry",
		"  validate  Validate local skill structure and prompt coverage",
		"  install   Install a local module entry resolved from registry metadata",
		"  run-agent Run production preflight checks for an installed agent package",
		"",
		"Examples:",
		"  skills-hub list",
		"  skills-hub search --tag paid-media --runtime codex",
		"  skills-hub info --skill marketing/meta-google-weekly-performance-review@latest",
		"  skills-hub info --module agents --entry marketing/weekly-performance-supervisor@latest",
		"  skills-hub validate",
		"  skills-hub install marketing/meta-google-weekly-performance-review@latest --runtime codex",
		"  skills-hub install --module agents --entry marketing/weekly-performance-supervisor@latest --runtime codex",
		"  skills-hub install --skill marketing/meta-google-weekly-performance-review@0.1.0 --runtime generic --target ./my-agent/skills",
		"  skills-hub run-agent --agent marketing/weekly-performance-supervisor --bindings agents/marketing/weekly-performance-supervisor/config/tool-bindings.example.json --approve-live",
	}
	fmt.Println(strings.Join(lines, "\n"))
}

func runAgent(args []string) error {
	fs := flag.NewFlagSet("run-agent", flag.ContinueOnError)
	agentID := fs.String("agent", "", "agent id in <domain>/<slug> format")
	agentsRoot := fs.String("agents-root", "agents", "agents root directory")
	skillsRoot := fs.String("skills-root", "skills", "skills root directory")
	toolsRoot := fs.String("tools-root", "tools-mcp", "tools root directory")
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

	report, err := agents.RunPreflight(agents.RunOptions{
		AgentsRoot:     *agentsRoot,
		SkillsRoot:     *skillsRoot,
		ToolsRoot:      *toolsRoot,
		AgentID:        strings.TrimSpace(*agentID),
		BindingsPath:   strings.TrimSpace(*bindingsPath),
		MemoryPath:     strings.TrimSpace(*memoryPath),
		GovernancePath: strings.TrimSpace(*governancePath),
		ApproveLive:    *approveLive,
		AuditPath:      strings.TrimSpace(*auditPath),
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

func normalizeModule(raw string) (string, error) {
	module := strings.ToLower(strings.TrimSpace(raw))
	if module == "" {
		module = "skills"
	}
	switch module {
	case "skills", "skill":
		return "skills", nil
	case "agents", "agent":
		return "agents", nil
	case "tools", "tool", "tools-mcp":
		return "tools", nil
	default:
		return "", fmt.Errorf("unsupported module: %s (supported: skills, agents, tools)", raw)
	}
}

func resolveRegistryPath(explicit, module string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	switch module {
	case "skills":
		return "registry/skills-index.json"
	case "agents":
		return "registry/agents-index.json"
	case "tools":
		return "registry/tools-index.json"
	default:
		return "registry/index.json"
	}
}

func resolveModuleRoot(explicit, module string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	switch module {
	case "skills":
		return "skills"
	case "agents":
		return "agents"
	case "tools":
		return "tools-mcp"
	default:
		return "skills"
	}
}
