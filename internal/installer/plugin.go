package installer

import (
	"errors"
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
	Closure         []InstallReceiptMember
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
		installed, skipped, members, err := installDependencySet(
			entry.Includes.Skills,
			runtimeName,
			pluginTargetRoot,
			"skills",
			moduleRoots["skills"],
			registryPaths["skills"],
			force,
		)
		if err != nil {
			return result, err
		}
		result.InstalledSkills = installed
		result.SkippedSkills = skipped
		result.Closure = append(result.Closure, members...)
	}

	if len(entry.Includes.Agents) > 0 {
		installed, skipped, members, err := installDependencySet(
			entry.Includes.Agents,
			runtimeName,
			pluginTargetRoot,
			"agents",
			moduleRoots["agents"],
			registryPaths["agents"],
			force,
		)
		if err != nil {
			return result, err
		}
		result.InstalledAgents = installed
		result.SkippedAgents = skipped
		result.Closure = append(result.Closure, members...)
	}

	if len(entry.Includes.Tools) > 0 {
		installed, skipped, members, err := installDependencySet(
			entry.Includes.Tools,
			runtimeName,
			pluginTargetRoot,
			"tools",
			moduleRoots["tools"],
			registryPaths["tools"],
			force,
		)
		if err != nil {
			return result, err
		}
		result.InstalledTools = installed
		result.SkippedTools = skipped
		result.Closure = append(result.Closure, members...)
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
		manifestName, nameErr := manifestNameForModule(moduleName)
		if nameErr != nil {
			return nameErr
		}
		manifestPath := filepath.Join(sourceDir, manifestName)
		if _, statErr := os.Stat(manifestPath); statErr == nil {
			manifest, manifestErr := registry.ValidatePackageManifest(manifestPath)
			if manifestErr != nil {
				return fmt.Errorf("validate local %s dependency %s: %w", moduleName, id, manifestErr)
			}
			if manifest.ID != id {
				return fmt.Errorf("local %s dependency identity mismatch: expected %s, got %s", moduleName, id, manifest.ID)
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("inspect local %s dependency %s manifest %s: %w", moduleName, id, manifestPath, statErr)
		} else if strings.HasPrefix(entry.SchemaVersion, "2.") {
			return fmt.Errorf("local %s dependency %s is missing required %s", moduleName, id, manifestName)
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
	registryPath string,
	force bool,
) ([]string, []string, []InstallReceiptMember, error) {
	if strings.TrimSpace(moduleRoot) == "" {
		return nil, nil, nil, fmt.Errorf("missing module root for %s", moduleName)
	}
	target, err := ResolvePluginDependencyTarget(runtimeName, moduleName, pluginTargetRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	index, err := registry.LoadIndex(registryPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load %s registry: %w", moduleName, err)
	}

	installed := make([]string, 0, len(ids))
	skipped := make([]string, 0)
	members := make([]InstallReceiptMember, 0, len(ids))
	for _, id := range ids {
		sourceDir := filepath.Join(moduleRoot, filepath.FromSlash(id))
		entry, ok := registry.FindSkill(index, id)
		if !ok {
			return nil, nil, nil, fmt.Errorf("%s dependency not found in registry: %s", moduleName, id)
		}
		selectedVersion := entry.Latest
		if selectedVersion == "" {
			manifestName, nameErr := manifestNameForModule(moduleName)
			if nameErr != nil {
				return nil, nil, nil, nameErr
			}
			manifest, manifestErr := registry.ValidatePackageManifest(filepath.Join(sourceDir, manifestName))
			if manifestErr == nil {
				entry = registry.ProjectManifest(manifest)
				selectedVersion = manifest.Version
			} else if errors.Is(manifestErr, os.ErrNotExist) || strings.Contains(manifestErr.Error(), "no such file") {
				// Legacy local registries used by older clients did not retain a
				// version projection. Keep them verifiable without inventing a
				// release identity.
				selectedVersion = "unversioned-local"
			} else {
				return nil, nil, nil, fmt.Errorf("validate local %s dependency %s: %w", moduleName, id, manifestErr)
			}
		}
		contractDigest, digestErr := RuntimeContractSHA256(entry, selectedVersion)
		if digestErr != nil {
			return nil, nil, nil, digestErr
		}
		destinationDir := filepath.Join(target.TargetPath, filepath.FromSlash(id))
		if _, statErr := os.Stat(destinationDir); statErr == nil && !force {
			receipt, verifyErr := VerifyInstallReceipt(destinationDir, moduleName, id, selectedVersion, runtimeName, "", contractDigest)
			if verifyErr != nil {
				sourceDigest, sourceErr := packageTreeSHA256(sourceDir)
				destinationDigest, destinationErr := packageTreeSHA256(destinationDir)
				if sourceErr != nil || destinationErr != nil || !strings.EqualFold(sourceDigest, destinationDigest) {
					return nil, nil, nil, fmt.Errorf("existing %s dependency %s is unverifiable; reinstall with --force: %w", moduleName, id, verifyErr)
				}
				if receiptErr := WriteInstallReceipt(destinationDir, InstallReceipt{Source: "local", Module: moduleName, ID: id, Version: selectedVersion, Runtime: runtimeName, RuntimeContractSHA256: contractDigest}); receiptErr != nil {
					return nil, nil, nil, receiptErr
				}
				receipt, verifyErr = VerifyInstallReceipt(destinationDir, moduleName, id, selectedVersion, runtimeName, "", contractDigest)
				if verifyErr != nil {
					return nil, nil, nil, verifyErr
				}
			}
			skipped = append(skipped, destinationDir)
			members = append(members, InstallReceiptMember{Module: moduleName, ID: id, Version: selectedVersion, RuntimeContractSHA256: contractDigest, TreeSHA256: receipt.TreeSHA256})
			continue
		}

		destination, installErr := InstallSkill(sourceDir, target.TargetPath, id, force)
		if installErr != nil {
			return nil, nil, nil, fmt.Errorf("install %s dependency %s: %w", moduleName, id, installErr)
		}
		if receiptErr := WriteInstallReceipt(destination, InstallReceipt{Source: "local", Module: moduleName, ID: id, Version: selectedVersion, Runtime: runtimeName, RuntimeContractSHA256: contractDigest}); receiptErr != nil {
			return nil, nil, nil, fmt.Errorf("write %s dependency receipt %s: %w", moduleName, id, receiptErr)
		}
		receipt, verifyErr := VerifyInstallReceipt(destination, moduleName, id, selectedVersion, runtimeName, "", contractDigest)
		if verifyErr != nil {
			return nil, nil, nil, verifyErr
		}
		installed = append(installed, destination)
		members = append(members, InstallReceiptMember{Module: moduleName, ID: id, Version: selectedVersion, RuntimeContractSHA256: contractDigest, TreeSHA256: receipt.TreeSHA256})
	}

	return installed, skipped, members, nil
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
