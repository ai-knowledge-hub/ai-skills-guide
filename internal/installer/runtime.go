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
	runtime := strings.ToLower(strings.TrimSpace(runtimeName))
	if runtime == "" {
		runtime = "generic"
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
		return RuntimeTarget{Runtime: runtime, TargetPath: codexDefaultSkillsDir()}, nil
	case "claude":
		return RuntimeTarget{Runtime: runtime, TargetPath: claudeDefaultSkillsDir()}, nil
	case "generic", "custom", "other":
		return RuntimeTarget{}, errors.New("--target is required for generic/custom runtimes")
	default:
		return RuntimeTarget{}, fmt.Errorf("unsupported runtime: %s (supported: codex, claude, generic)", runtime)
	}
}

func codexDefaultSkillsDir() string {
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return filepath.Join(codexHome, "skills")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex/skills"
	}
	return filepath.Join(home, ".codex", "skills")
}

func claudeDefaultSkillsDir() string {
	if claudeHome := strings.TrimSpace(os.Getenv("CLAUDE_HOME")); claudeHome != "" {
		return filepath.Join(claudeHome, "skills")
	}
	if claudeCodeHome := strings.TrimSpace(os.Getenv("CLAUDE_CODE_HOME")); claudeCodeHome != "" {
		return filepath.Join(claudeCodeHome, "skills")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude/skills"
	}
	return filepath.Join(home, ".claude", "skills")
}
