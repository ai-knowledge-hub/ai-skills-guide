package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	registryPath := fs.String("registry", "registry/index.json", "registry index path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	idx, err := registry.LoadIndex(*registryPath)
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
	registryPath := fs.String("registry", "registry/index.json", "registry index path")
	text := fs.String("q", "", "free text search against id/name/description")
	tag := fs.String("tag", "", "filter by tag")
	category := fs.String("category", "", "filter by category")
	runtime := fs.String("runtime", "", "filter by runtime")
	if err := fs.Parse(args); err != nil {
		return err
	}

	idx, err := registry.LoadIndex(*registryPath)
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
	registryPath := fs.String("registry", "registry/index.json", "registry index path")
	skillsRoot := fs.String("root", "skills", "skills root directory")
	skillSpec := fs.String("skill", "", "skill id, optionally with @version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skillSpec == "" {
		return errors.New("--skill is required")
	}

	id, requestedVersion, err := parseSkillSpec(*skillSpec)
	if err != nil {
		return err
	}

	idx, err := registry.LoadIndex(*registryPath)
	if err != nil {
		return err
	}
	skill, ok := registry.FindSkill(idx, id)
	if !ok {
		return fmt.Errorf("skill not found in registry: %s", id)
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
	fmt.Printf("path: %s\n", filepath.Join(*skillsRoot, filepath.FromSlash(skill.ID)))
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
	skillsRoot := fs.String("root", "skills", "skills root directory")
	registryPath := fs.String("registry", "registry/index.json", "registry index path")
	runtimeName := fs.String("runtime", "generic", "runtime adapter: codex|claude|generic")
	target := fs.String("target", "", "destination skills directory (optional for codex/claude)")
	skillSpecFlag := fs.String("skill", "", "skill id, optionally with @version")
	force := fs.Bool("force", false, "overwrite destination if it already exists")
	if err := fs.Parse(parsedArgs); err != nil {
		return err
	}

	skillSpec := strings.TrimSpace(*skillSpecFlag)
	if skillSpec == "" {
		skillSpec = positionalSpec
	}
	if skillSpec == "" {
		return errors.New("skill is required (use --skill <id[@version]> or positional <id[@version]>)")
	}

	id, requestedVersion, err := parseSkillSpec(skillSpec)
	if err != nil {
		return err
	}

	idx, err := registry.LoadIndex(*registryPath)
	if err != nil {
		return err
	}
	skill, ok := registry.FindSkill(idx, id)
	if !ok {
		return fmt.Errorf("skill not found in registry: %s", id)
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

	sourceDir := filepath.Join(*skillsRoot, filepath.FromSlash(skill.ID))
	if stat, statErr := os.Stat(sourceDir); statErr != nil || !stat.IsDir() {
		return fmt.Errorf("local source not found for %s at %s", skill.ID, sourceDir)
	}

	rt, err := installer.ResolveRuntimeTarget(*runtimeName, *target)
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
		"skills-hub: manage local marketing/adtech skill packages",
		"",
		"Usage:",
		"  skills-hub <command> [flags]",
		"",
		"Commands:",
		"  list      List skills from registry/index.json",
		"  search    Search skills by text/tag/category/runtime",
		"  info      Show details for one skill",
		"  validate  Validate local skill structure and prompt coverage",
		"  install   Install a local skill package resolved from registry metadata",
		"",
		"Examples:",
		"  skills-hub list",
		"  skills-hub search --tag paid-media --runtime codex",
		"  skills-hub info --skill marketing/meta-google-weekly-performance-review@latest",
		"  skills-hub validate",
		"  skills-hub install marketing/meta-google-weekly-performance-review@latest --runtime codex",
		"  skills-hub install --skill marketing/meta-google-weekly-performance-review@0.1.0 --runtime generic --target ./my-agent/skills",
	}
	fmt.Println(strings.Join(lines, "\n"))
}
