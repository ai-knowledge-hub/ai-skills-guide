package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type DependencyInstallResult struct {
	InstalledSkills []string
	SkippedSkills   []string
	InstalledAgents []string
	SkippedAgents   []string
	InstalledTools  []string
	SkippedTools    []string
	HookPaths       []string
}

func InstallPluginDependencies(
	entry registry.SkillEntry,
	runtimeName string,
	pluginTargetRoot string,
	moduleRoots map[string]string,
	registryPaths map[string]string,
	force bool,
) (DependencyInstallResult, error) {
	result := DependencyInstallResult{}
	pluginSourceDir := filepath.Join(pluginTargetRoot, filepath.FromSlash(entry.ID))
	if err := PreflightPluginDependencies(entry, runtimeName, pluginTargetRoot, pluginSourceDir, moduleRoots, registryPaths); err != nil {
		return result, err
	}

	if entry.Includes == nil {
		return result, nil
	}

	if len(entry.Includes.Skills) > 0 {
		installed, skipped, err := installDependencySet(
			entry.Includes.Skills,
			runtimeName,
			pluginTargetRoot,
			"skills",
			moduleRoots["skills"],
			force,
		)
		if err != nil {
			return result, err
		}
		result.InstalledSkills = installed
		result.SkippedSkills = skipped
	}

	if len(entry.Includes.Agents) > 0 {
		installed, skipped, err := installDependencySet(
			entry.Includes.Agents,
			runtimeName,
			pluginTargetRoot,
			"agents",
			moduleRoots["agents"],
			force,
		)
		if err != nil {
			return result, err
		}
		result.InstalledAgents = installed
		result.SkippedAgents = skipped
	}

	if len(entry.Includes.Tools) > 0 {
		installed, skipped, err := installDependencySet(
			entry.Includes.Tools,
			runtimeName,
			pluginTargetRoot,
			"tools",
			moduleRoots["tools"],
			force,
		)
		if err != nil {
			return result, err
		}
		result.InstalledTools = installed
		result.SkippedTools = skipped
	}

	for _, hookName := range entry.Includes.Hooks {
		hookPath, err := resolveHookPath(filepath.Join(pluginTargetRoot, filepath.FromSlash(entry.ID), "hooks"), hookName)
		if err != nil {
			return result, fmt.Errorf("resolve hook %s for %s: %w", hookName, entry.ID, err)
		}
		result.HookPaths = append(result.HookPaths, hookPath)
	}

	return result, nil
}

// PreflightPluginDependencies validates the complete cross-module dependency
// closure before an installer performs its first filesystem mutation.
func PreflightPluginDependencies(
	entry registry.SkillEntry,
	runtimeName string,
	pluginTargetRoot string,
	pluginSourceDir string,
	moduleRoots map[string]string,
	registryPaths map[string]string,
) error {
	if err := ValidateOperationalInstall(entry); err != nil {
		return err
	}
	if entry.Includes == nil {
		return nil
	}

	sets := []struct {
		ids        []string
		moduleName string
	}{
		{ids: entry.Includes.Skills, moduleName: "skills"},
		{ids: entry.Includes.Agents, moduleName: "agents"},
		{ids: entry.Includes.Tools, moduleName: "tools"},
	}
	for _, set := range sets {
		if len(set.ids) == 0 {
			continue
		}
		if err := preflightDependencySet(
			set.ids,
			runtimeName,
			pluginTargetRoot,
			set.moduleName,
			moduleRoots[set.moduleName],
			registryPaths[set.moduleName],
		); err != nil {
			return err
		}
	}

	for _, hookName := range entry.Includes.Hooks {
		if _, err := resolveHookPath(filepath.Join(pluginSourceDir, "hooks"), hookName); err != nil {
			return fmt.Errorf("resolve hook %s for %s: %w", hookName, entry.ID, err)
		}
	}
	return nil
}

func preflightDependencySet(
	ids []string,
	runtimeName string,
	pluginTargetRoot string,
	moduleName string,
	moduleRoot string,
	registryPath string,
) error {
	if strings.TrimSpace(moduleRoot) == "" {
		return fmt.Errorf("missing module root for %s", moduleName)
	}
	if strings.TrimSpace(registryPath) == "" {
		return fmt.Errorf("missing registry path for %s", moduleName)
	}

	idx, err := registry.LoadIndex(registryPath)
	if err != nil {
		return fmt.Errorf("load %s registry: %w", moduleName, err)
	}
	if _, err := ResolvePluginDependencyTarget(runtimeName, moduleName, pluginTargetRoot); err != nil {
		return err
	}
	for _, id := range ids {
		entry, ok := registry.FindSkill(idx, id)
		if !ok {
			return fmt.Errorf("%s dependency not found in registry: %s", moduleName, id)
		}
		if err := ValidateOperationalInstall(entry); err != nil {
			return fmt.Errorf("%s dependency: %w", moduleName, err)
		}
		sourceDir := filepath.Join(moduleRoot, filepath.FromSlash(id))
		if stat, statErr := os.Stat(sourceDir); statErr != nil || !stat.IsDir() {
			return fmt.Errorf("local source not found for %s dependency at %s", id, sourceDir)
		}
	}
	return nil
}

func installDependencySet(
	ids []string,
	runtimeName string,
	pluginTargetRoot string,
	moduleName string,
	moduleRoot string,
	force bool,
) ([]string, []string, error) {
	if strings.TrimSpace(moduleRoot) == "" {
		return nil, nil, fmt.Errorf("missing module root for %s", moduleName)
	}
	target, err := ResolvePluginDependencyTarget(runtimeName, moduleName, pluginTargetRoot)
	if err != nil {
		return nil, nil, err
	}

	installed := make([]string, 0, len(ids))
	skipped := make([]string, 0)
	for _, id := range ids {
		sourceDir := filepath.Join(moduleRoot, filepath.FromSlash(id))
		destinationDir := filepath.Join(target.TargetPath, filepath.FromSlash(id))
		if _, statErr := os.Stat(destinationDir); statErr == nil && !force {
			skipped = append(skipped, destinationDir)
			continue
		}

		destination, installErr := InstallSkill(sourceDir, target.TargetPath, id, force)
		if installErr != nil {
			return nil, nil, fmt.Errorf("install %s dependency %s: %w", moduleName, id, installErr)
		}
		installed = append(installed, destination)
	}

	return installed, skipped, nil
}

func ResolvePluginDependencyTarget(runtimeName, moduleName, pluginTargetRoot string) (RuntimeTarget, error) {
	runtime := strings.ToLower(strings.TrimSpace(runtimeName))
	pluginRootAbs, err := filepath.Abs(pluginTargetRoot)
	if err != nil {
		return RuntimeTarget{}, fmt.Errorf("resolve plugin target path: %w", err)
	}
	baseRoot := filepath.Dir(pluginRootAbs)
	moduleDir, err := normalizeModuleDir(moduleName)
	if err != nil {
		return RuntimeTarget{}, err
	}
	return RuntimeTarget{
		Runtime:    runtime,
		TargetPath: filepath.Join(baseRoot, moduleDir),
	}, nil
}

func resolveHookPath(hooksDir, hookName string) (string, error) {
	candidates := []string{
		filepath.Join(hooksDir, hookName+".md"),
		filepath.Join(hooksDir, hookName+".json"),
		filepath.Join(hooksDir, hookName+".yaml"),
	}
	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("missing hook file in %s", hooksDir)
}
