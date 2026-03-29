package plugins

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

var numberedPromptPattern = regexp.MustCompile(`^[0-9]+\.`)

type ValidationIssue struct {
	PluginID string
	Message  string
}

type ValidateOptions struct {
	PluginsRoot string
	SkillsRoot  string
	AgentsRoot  string
	ToolsRoot   string
}

func Validate(opts ValidateOptions) ([]ValidationIssue, error) {
	pluginDirs, err := discoverPluginDirs(opts.PluginsRoot)
	if err != nil {
		return nil, err
	}

	issues := make([]ValidationIssue, 0)
	for _, pluginDir := range pluginDirs {
		pluginID := filepath.ToSlash(strings.TrimPrefix(pluginDir, opts.PluginsRoot+string(filepath.Separator)))
		if pluginID == pluginDir {
			pluginID = filepath.ToSlash(strings.TrimPrefix(pluginDir, opts.PluginsRoot))
		}

		requiredFiles := []string{
			"README.md",
			"plugin.yaml",
			"plugin.json",
			filepath.Join("tests", "test-prompts.md"),
		}
		for _, rel := range requiredFiles {
			path := filepath.Join(pluginDir, rel)
			if _, err := os.Stat(path); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					issues = append(issues, ValidationIssue{PluginID: pluginID, Message: fmt.Sprintf("missing %s", rel)})
					continue
				}
				return nil, fmt.Errorf("stat %s: %w", path, err)
			}
		}

		examplesPath := filepath.Join(pluginDir, "examples")
		if stat, err := os.Stat(examplesPath); err != nil || !stat.IsDir() {
			issues = append(issues, ValidationIssue{PluginID: pluginID, Message: "missing examples/ directory"})
		}

		manifestPath := filepath.Join(pluginDir, "plugin.yaml")
		manifest, err := registry.ParseManifest(manifestPath)
		if err != nil {
			return nil, err
		}

		promptCount, err := countNumberedPrompts(filepath.Join(pluginDir, "tests", "test-prompts.md"))
		if err != nil {
			return nil, err
		}
		if promptCount < 5 {
			issues = append(issues, ValidationIssue{PluginID: manifest.ID, Message: fmt.Sprintf("need at least 5 numbered prompts (found %d)", promptCount)})
		}

		for _, id := range manifest.Includes.Skills {
			if !entryDirExists(opts.SkillsRoot, id) {
				issues = append(issues, ValidationIssue{PluginID: manifest.ID, Message: fmt.Sprintf("missing bundled skill: %s", id)})
			}
		}
		for _, id := range manifest.Includes.Agents {
			if !entryDirExists(opts.AgentsRoot, id) {
				issues = append(issues, ValidationIssue{PluginID: manifest.ID, Message: fmt.Sprintf("missing bundled agent: %s", id)})
			}
		}
		for _, id := range manifest.Includes.Tools {
			if !entryDirExists(opts.ToolsRoot, id) {
				issues = append(issues, ValidationIssue{PluginID: manifest.ID, Message: fmt.Sprintf("missing bundled tool: %s", id)})
			}
		}

		if len(manifest.Includes.Hooks) > 0 {
			hooksDir := filepath.Join(pluginDir, "hooks")
			if stat, err := os.Stat(hooksDir); err != nil || !stat.IsDir() {
				issues = append(issues, ValidationIssue{PluginID: manifest.ID, Message: "missing hooks/ directory"})
			} else {
				for _, hookName := range manifest.Includes.Hooks {
					if !hookExists(hooksDir, hookName) {
						issues = append(issues, ValidationIssue{PluginID: manifest.ID, Message: fmt.Sprintf("missing packaged hook file for: %s", hookName)})
					}
				}
			}
		}
	}

	return issues, nil
}

func discoverPluginDirs(root string) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}

	dirs := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.Count(filepath.ToSlash(rel), "/") == 1 {
			dirs = append(dirs, path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk plugins root %s: %w", root, err)
	}
	return dirs, nil
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

func entryDirExists(root, id string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	path := filepath.Join(root, filepath.FromSlash(id))
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func hookExists(hooksDir, hookName string) bool {
	candidates := []string{
		filepath.Join(hooksDir, hookName+".md"),
		filepath.Join(hooksDir, hookName+".json"),
		filepath.Join(hooksDir, hookName+".yaml"),
	}
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return true
		}
	}
	return false
}
