package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/agents"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	pluginvalidate "github.com/ai-knowledge-hub/ai-skills-guide/internal/plugins"
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
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
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
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
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
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
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
	if moduleName == "plugins" {
		printPluginSummary(os.Stdout, skill)
	}
	return nil
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	module := fs.String("module", "skills", "module to validate: skills|plugins|all")
	root := fs.String("root", "skills", "module root directory")
	skillsRoot := fs.String("skills-root", "skills", "skills root directory for plugin validation")
	agentsRoot := fs.String("agents-root", "agents", "agents root directory for plugin validation")
	toolsRoot := fs.String("tools-root", "tools-mcp", "tools root directory for plugin validation")
	if err := fs.Parse(args); err != nil {
		return err
	}

	moduleName, err := normalizeValidateModule(*module)
	if err != nil {
		return err
	}

	totalIssues := 0
	if moduleName == "skills" || moduleName == "all" {
		skillRoot := *root
		if moduleName == "all" {
			skillRoot = "skills"
		}
		issues, err := skills.Validate(skillRoot)
		if err != nil {
			return err
		}
		for _, issue := range issues {
			fmt.Printf("[ERROR] %s: %s\n", issue.SkillID, issue.Message)
		}
		totalIssues += len(issues)
		if len(issues) == 0 && moduleName == "skills" {
			fmt.Println("All skills passed validation.")
		}
	}

	if moduleName == "plugins" || moduleName == "all" {
		pluginsRoot := *root
		if moduleName == "all" {
			pluginsRoot = "plugins"
		}
		issues, err := pluginvalidate.Validate(pluginvalidate.ValidateOptions{
			PluginsRoot: pluginsRoot,
			SkillsRoot:  *skillsRoot,
			AgentsRoot:  *agentsRoot,
			ToolsRoot:   *toolsRoot,
		})
		if err != nil {
			return err
		}
		for _, issue := range issues {
			fmt.Printf("[ERROR] %s: %s\n", issue.PluginID, issue.Message)
		}
		totalIssues += len(issues)
		if len(issues) == 0 && moduleName == "plugins" {
			fmt.Println("All plugins passed validation.")
		}
	}

	if totalIssues == 0 {
		if moduleName == "all" {
			fmt.Println("All skills and plugins passed validation.")
		}
		return nil
	}
	return fmt.Errorf("validation failed with %d issue(s)", totalIssues)
}

func runInstall(args []string) error {
	positionalSpec := ""
	parsedArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positionalSpec = args[0]
		parsedArgs = args[1:]
	}

	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	module := fs.String("module", "skills", "module: skills|agents|tools|plugins")
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

	var runtimeArtifacts []string
	var dependencyResult installer.DependencyInstallResult
	if moduleName == "plugins" {
		runtimeArtifacts, err = installer.PreparePluginRuntimeArtifacts(destination, rt.Runtime)
		if err != nil {
			return err
		}
		dependencyResult, err = installer.InstallPluginDependencies(
			skill,
			rt.Runtime,
			rt.TargetPath,
			map[string]string{
				"skills": resolveModuleRoot("", "skills"),
				"agents": resolveModuleRoot("", "agents"),
				"tools":  resolveModuleRoot("", "tools"),
			},
			map[string]string{
				"skills": resolveRegistryPath("", "skills"),
				"agents": resolveRegistryPath("", "agents"),
				"tools":  resolveRegistryPath("", "tools"),
			},
			*force,
		)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Installed %s@%s to %s (runtime=%s)\n", skill.ID, resolvedVersion.Version, destination, rt.Runtime)
	if moduleName == "plugins" {
		printPluginInstallNotes(os.Stdout, os.Stderr, skill, rt.Runtime, destination, runtimeArtifacts, dependencyResult)
	}
	return nil
}

func printPluginSummary(w io.Writer, entry registry.SkillEntry) {
	skills := optionalList(entry.Includes, "skills")
	agents := optionalList(entry.Includes, "agents")
	tools := optionalList(entry.Includes, "tools")
	hooks := optionalList(entry.Includes, "hooks")
	secrets := optionalList(entry.Requires, "secrets")
	approvals := optionalList(entry.Requires, "approvals")

	fmt.Fprintf(w, "includes.skills: %d\n", len(skills))
	if len(skills) > 0 {
		fmt.Fprintf(w, "includes.skills.list: %s\n", strings.Join(skills, ", "))
	}
	fmt.Fprintf(w, "includes.agents: %d\n", len(agents))
	if len(agents) > 0 {
		fmt.Fprintf(w, "includes.agents.list: %s\n", strings.Join(agents, ", "))
	}
	fmt.Fprintf(w, "includes.tools: %d\n", len(tools))
	if len(tools) > 0 {
		fmt.Fprintf(w, "includes.tools.list: %s\n", strings.Join(tools, ", "))
	}
	fmt.Fprintf(w, "includes.hooks: %d\n", len(hooks))
	if len(hooks) > 0 {
		fmt.Fprintf(w, "includes.hooks.list: %s\n", strings.Join(hooks, ", "))
	}
	if len(secrets) > 0 {
		fmt.Fprintf(w, "requires.secrets: %s\n", strings.Join(secrets, ", "))
	}
	if len(approvals) > 0 {
		fmt.Fprintf(w, "requires.approvals: %s\n", strings.Join(approvals, ", "))
	}
}

func printPluginInstallNotes(
	stdout, stderr io.Writer,
	entry registry.SkillEntry,
	runtime, destination string,
	runtimeArtifacts []string,
	deps installer.DependencyInstallResult,
) {
	if !entry.SecurityReviewed {
		fmt.Fprintf(stderr, "warning: %s is not security reviewed; inspect bundled components and setup before use\n", entry.ID)
	}
	if destination != "" {
		fmt.Fprintf(stdout, "install.root: %s\n", destination)
	}
	if len(runtimeArtifacts) > 0 {
		fmt.Fprintf(stdout, "install.runtime_artifacts: %s\n", strings.Join(runtimeArtifacts, ", "))
	}
	if runtime == "codex" || runtime == "claude" {
		fmt.Fprintf(stdout, "install.next_step: review the generated %s runtime manifest before enabling the plugin\n", runtime)
	} else if runtime != "" {
		fmt.Fprintf(stdout, "install.next_step: connect this plugin directory to your runtime manually and verify required secrets before use\n")
	}
	printPluginDependencySummary(stdout, deps)
	printPluginSummary(stdout, entry)
}

func printPluginDependencySummary(stdout io.Writer, deps installer.DependencyInstallResult) {
	fmt.Fprintf(stdout, "deps.skills.installed: %d\n", len(deps.InstalledSkills))
	if len(deps.InstalledSkills) > 0 {
		fmt.Fprintf(stdout, "deps.skills.installed.list: %s\n", strings.Join(deps.InstalledSkills, ", "))
	}
	if len(deps.SkippedSkills) > 0 {
		fmt.Fprintf(stdout, "deps.skills.skipped.list: %s\n", strings.Join(deps.SkippedSkills, ", "))
	}
	fmt.Fprintf(stdout, "deps.agents.installed: %d\n", len(deps.InstalledAgents))
	if len(deps.InstalledAgents) > 0 {
		fmt.Fprintf(stdout, "deps.agents.installed.list: %s\n", strings.Join(deps.InstalledAgents, ", "))
	}
	if len(deps.SkippedAgents) > 0 {
		fmt.Fprintf(stdout, "deps.agents.skipped.list: %s\n", strings.Join(deps.SkippedAgents, ", "))
	}
	fmt.Fprintf(stdout, "deps.tools.installed: %d\n", len(deps.InstalledTools))
	if len(deps.InstalledTools) > 0 {
		fmt.Fprintf(stdout, "deps.tools.installed.list: %s\n", strings.Join(deps.InstalledTools, ", "))
	}
	if len(deps.SkippedTools) > 0 {
		fmt.Fprintf(stdout, "deps.tools.skipped.list: %s\n", strings.Join(deps.SkippedTools, ", "))
	}
	fmt.Fprintf(stdout, "deps.hooks.packaged: %d\n", len(deps.HookPaths))
	if len(deps.HookPaths) > 0 {
		fmt.Fprintf(stdout, "deps.hooks.packaged.list: %s\n", strings.Join(deps.HookPaths, ", "))
	}
}

func optionalList[T any](set *T, field string) []string {
	if set == nil {
		return nil
	}
	switch v := any(set).(type) {
	case *registry.IncludeSet:
		switch field {
		case "skills":
			return v.Skills
		case "agents":
			return v.Agents
		case "tools":
			return v.Tools
		case "hooks":
			return v.Hooks
		}
	case *registry.RequirementSet:
		switch field {
		case "secrets":
			return v.Secrets
		case "approvals":
			return v.Approvals
		}
	}
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
		"skills-hub: manage local skill, agent, tool, and plugin packages",
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
		"  skills-hub info --module plugins --entry marketing/performance-reporting-plugin@latest",
		"  skills-hub validate",
		"  skills-hub install marketing/meta-google-weekly-performance-review@latest --runtime codex",
		"  skills-hub install --module agents --entry marketing/weekly-performance-supervisor@latest --runtime codex",
		"  skills-hub install --module plugins --entry marketing/performance-reporting-plugin@latest --runtime codex",
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
	case "plugins", "plugin":
		return "plugins", nil
	default:
		return "", fmt.Errorf("unsupported module: %s (supported: skills, agents, tools, plugins)", raw)
	}
}

func normalizeValidateModule(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "skills"
	}
	switch value {
	case "skills", "skill":
		return "skills", nil
	case "plugins", "plugin":
		return "plugins", nil
	case "all":
		return "all", nil
	default:
		return "", fmt.Errorf("unsupported validate module: %s (supported: skills, plugins, all)", raw)
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
	case "plugins":
		return "registry/plugins-index.json"
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
	case "plugins":
		return "plugins"
	default:
		return "skills"
	}
}
