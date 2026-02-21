package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateHappyPath(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "marketing", "demo-skill")
	mustMkdir(t, filepath.Join(skillDir, "tests"))
	mustMkdir(t, filepath.Join(skillDir, "examples"))
	mustWrite(t, filepath.Join(skillDir, "README.md"), "# Demo")
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), `---
name: demo-skill
description: test
---

## When to use
x

## Guardrails
y
`)
	mustWrite(t, filepath.Join(skillDir, "tests", "test-prompts.md"), "1. a\n2. b\n3. c\n4. d\n5. e\n")

	issues, err := Validate(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
