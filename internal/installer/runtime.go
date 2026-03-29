package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RuntimeTarget struct {
	Runtime    string
	TargetPath string
}

func ResolveRuntimeTarget(runtimeName, explicitTarget string) (RuntimeTarget, error) {
	return ResolveRuntimeTargetForModule(runtimeName, "skills", explicitTarget)
}

func ResolveRuntimeTargetForModule(runtimeName, moduleName, explicitTarget string) (RuntimeTarget, error) {
	runtime := strings.ToLower(strings.TrimSpace(runtimeName))
	if runtime == "" {
		runtime = "generic"
	}
	moduleDir, err := normalizeModuleDir(moduleName)
	if err != nil {
		return RuntimeTarget{}, err
	}

	if strings.TrimSpace(explicitTarget) != "" {
		abs, err := filepath.Abs(explicitTarget)
		if err != nil {
			return RuntimeTarget{}, fmt.Errorf("resolve target path: %w", err)
		}
		return RuntimeTarget{Runtime: runtime, TargetPath: abs}, nil
	}

	switch runtime {
	case "codex":
		return RuntimeTarget{Runtime: runtime, TargetPath: codexDefaultDir(moduleDir)}, nil
	case "claude":
		return RuntimeTarget{Runtime: runtime, TargetPath: claudeDefaultDir(moduleDir)}, nil
	case "generic", "custom", "other":
		return RuntimeTarget{}, errors.New("--target is required for generic/custom runtimes")
	default:
		return RuntimeTarget{}, fmt.Errorf("unsupported runtime: %s (supported: codex, claude, generic)", runtime)
	}
}

func codexDefaultDir(moduleDir string) string {
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return filepath.Join(codexHome, moduleDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".codex", moduleDir)
	}
	return filepath.Join(home, ".codex", moduleDir)
}

func claudeDefaultDir(moduleDir string) string {
	if claudeHome := strings.TrimSpace(os.Getenv("CLAUDE_HOME")); claudeHome != "" {
		return filepath.Join(claudeHome, moduleDir)
	}
	if claudeCodeHome := strings.TrimSpace(os.Getenv("CLAUDE_CODE_HOME")); claudeCodeHome != "" {
		return filepath.Join(claudeCodeHome, moduleDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".claude", moduleDir)
	}
	return filepath.Join(home, ".claude", moduleDir)
}

func normalizeModuleDir(moduleName string) (string, error) {
	module := strings.ToLower(strings.TrimSpace(moduleName))
	if module == "" {
		module = "skills"
	}
	switch module {
	case "skills", "skill":
		return "skills", nil
	case "agents", "agent":
		return "agents", nil
	case "tools", "tool", "tools-mcp":
		return "tools-mcp", nil
	case "plugins", "plugin":
		return "plugins", nil
	default:
		return "", fmt.Errorf("unsupported module: %s (supported: skills, agents, tools, plugins)", moduleName)
	}
}
