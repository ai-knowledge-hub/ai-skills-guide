package skills

import "testing"

func TestParseFrontmatterContent(t *testing.T) {
	content := `---
name: sample-skill
description: Demo
deprecated: true
replaced_by: skills/marketing/new-skill
---

# Sample`

	fm, err := parseFrontmatterContent(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Name != "sample-skill" {
		t.Fatalf("unexpected name: %s", fm.Name)
	}
	if !fm.Deprecated {
		t.Fatalf("expected deprecated=true")
	}
	if fm.ReplacedBy != "skills/marketing/new-skill" {
		t.Fatalf("unexpected replacement: %s", fm.ReplacedBy)
	}
}
