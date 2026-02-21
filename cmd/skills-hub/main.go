package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
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
	root := fs.String("root", "skills", "skills root directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	all, err := skills.Discover(*root)
	if err != nil {
		return err
	}
	for _, s := range all {
		status := "active"
		if s.Deprecated {
			status = "deprecated"
		}
		fmt.Printf("%s\t%s\t%s\n", s.ID, status, s.Name)
	}
	return nil
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	root := fs.String("root", "skills", "skills root directory")
	skillID := fs.String("skill", "", "skill id (category/slug)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skillID == "" {
		return errors.New("--skill is required")
	}

	all, err := skills.Discover(*root)
	if err != nil {
		return err
	}
	skill, ok := skills.FindByID(all, *skillID)
	if !ok {
		return fmt.Errorf("skill not found: %s", *skillID)
	}

	fmt.Printf("id: %s\n", skill.ID)
	fmt.Printf("path: %s\n", skill.Path)
	fmt.Printf("name: %s\n", skill.Name)
	fmt.Printf("description: %s\n", skill.Description)
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
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	root := fs.String("root", "skills", "skills root directory")
	target := fs.String("target", "", "destination skills directory")
	skillID := fs.String("skill", "", "skill id (category/slug)")
	force := fs.Bool("force", false, "overwrite destination if it already exists")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *target == "" {
		return errors.New("--target is required")
	}
	if *skillID == "" {
		return errors.New("--skill is required")
	}

	all, err := skills.Discover(*root)
	if err != nil {
		return err
	}
	skill, ok := skills.FindByID(all, *skillID)
	if !ok {
		return fmt.Errorf("skill not found: %s", *skillID)
	}

	if skill.Deprecated {
		fmt.Fprintf(os.Stderr, "warning: %s is deprecated", skill.ID)
		if skill.ReplacedBy != "" {
			fmt.Fprintf(os.Stderr, "; prefer %s", skill.ReplacedBy)
		}
		fmt.Fprintln(os.Stderr)
	}

	absTarget, err := filepath.Abs(*target)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	destination, err := installer.InstallSkill(skill.Path, absTarget, skill.ID, *force)
	if err != nil {
		return err
	}

	fmt.Printf("Installed %s to %s\n", skill.ID, destination)
	return nil
}

func printUsage() {
	lines := []string{
		"skills-hub: manage local marketing/adtech skill packages",
		"",
		"Usage:",
		"  skills-hub <command> [flags]",
		"",
		"Commands:",
		"  list      List available skills",
		"  info      Show details for one skill",
		"  validate  Validate skill structure and prompt coverage",
		"  install   Copy a skill into a target runtime directory",
		"",
		"Examples:",
		"  skills-hub list",
		"  skills-hub info --skill marketing/meta-google-weekly-performance-review",
		"  skills-hub validate",
		"  skills-hub install --skill marketing/meta-google-weekly-performance-review --target ~/.codex/skills",
	}
	fmt.Println(strings.Join(lines, "\n"))
}
