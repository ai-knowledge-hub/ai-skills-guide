package skills

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var numberedPromptPattern = regexp.MustCompile(`^[0-9]+\.`)

type ValidationIssue struct {
	SkillID string
	Message string
}

func Validate(root string) ([]ValidationIssue, error) {
	all, err := Discover(root)
	if err != nil {
		return nil, err
	}

	issues := make([]ValidationIssue, 0)
	for _, skill := range all {
		skillPath := skill.Path

		requiredFiles := []string{
			"SKILL.md",
			"README.md",
			filepath.Join("tests", "test-prompts.md"),
		}
		for _, rel := range requiredFiles {
			path := filepath.Join(skillPath, rel)
			if _, err := os.Stat(path); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					issues = append(issues, ValidationIssue{SkillID: skill.ID, Message: fmt.Sprintf("missing %s", rel)})
					continue
				}
				return nil, fmt.Errorf("stat %s: %w", path, err)
			}
		}

		examplesPath := filepath.Join(skillPath, "examples")
		if stat, err := os.Stat(examplesPath); err != nil || !stat.IsDir() {
			issues = append(issues, ValidationIssue{SkillID: skill.ID, Message: "missing examples/ directory"})
		}

		skillMDPath := filepath.Join(skillPath, "SKILL.md")
		skillMD, err := os.ReadFile(skillMDPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", skillMDPath, err)
		}
		content := string(skillMD)
		if !strings.HasPrefix(content, "---") {
			issues = append(issues, ValidationIssue{SkillID: skill.ID, Message: "SKILL.md missing frontmatter"})
		}
		if !strings.Contains(strings.ToLower(content), "## when to use") {
			issues = append(issues, ValidationIssue{SkillID: skill.ID, Message: "SKILL.md missing 'When to use' section"})
		}
		if !strings.Contains(strings.ToLower(content), "## guardrails") {
			issues = append(issues, ValidationIssue{SkillID: skill.ID, Message: "SKILL.md missing 'Guardrails' section"})
		}

		promptPath := filepath.Join(skillPath, "tests", "test-prompts.md")
		promptCount, err := countNumberedPrompts(promptPath)
		if err != nil {
			return nil, err
		}
		if promptCount < 5 {
			issues = append(issues, ValidationIssue{SkillID: skill.ID, Message: fmt.Sprintf("need at least 5 numbered prompts (found %d)", promptCount)})
		}
	}

	return issues, nil
}

func countNumberedPrompts(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if numberedPromptPattern.MatchString(line) {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan %s: %w", path, err)
	}
	return count, nil
}
