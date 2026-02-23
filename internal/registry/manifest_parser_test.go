package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	content := `id: marketing/example-skill
name: Example Skill
description: First sentence,
  second sentence.
version: 0.1.0
released_at: "2026-02-23T00:00:00Z"
category: marketing-tools/ads-ops
tags:
  - one
  - two
license: MIT
author:
  name: Example
runtimes:
  - codex
  - generic
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.ID != "marketing/example-skill" {
		t.Fatalf("unexpected id: %s", m.ID)
	}
	if m.Description != "First sentence, second sentence." {
		t.Fatalf("unexpected description: %s", m.Description)
	}
	if m.Category != "marketing-tools/ads-ops" {
		t.Fatalf("unexpected category: %s", m.Category)
	}
	if len(m.Tags) != 2 || m.Tags[0] != "one" || m.Tags[1] != "two" {
		t.Fatalf("unexpected tags: %#v", m.Tags)
	}
	if len(m.Runtimes) != 2 || m.Runtimes[0] != "codex" {
		t.Fatalf("unexpected runtimes: %#v", m.Runtimes)
	}
}
