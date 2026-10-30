package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestValidateReleaseStoreRoundTripsOmittedOptionalUsability(t *testing.T) {
	const manifestText = `schema_version: "1.1"
id: engineering/roundtrip-skill
name: Roundtrip Skill
description: Retained release fixture with omitted optional usability metadata.
version: 1.0.0
released_at: "2026-09-15T08:00:00Z"
category: engineering/testing-quality
tags: [testing]
license: MIT
author:
  name: Test Maintainer
runtimes: [generic]
entrypoints:
  skill_md: SKILL.md
usability:
  availability: documentation-only
  execution: instructions
deprecated: false
`
	releaseRoot := t.TempDir()
	releaseDir := filepath.Join(
		releaseRoot, "skills", "engineering", "roundtrip-skill", "1.0.0",
	)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("create release directory: %v", err)
	}
	manifestPath := filepath.Join(releaseDir, "skill.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifestText), 0o644); err != nil {
		t.Fatalf("write retained manifest: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(releaseDir, "package.tar.gz"),
		releaseArchive(t, map[string]string{
			"skill.yaml": manifestText,
			"SKILL.md":   "# Roundtrip Skill\n",
		}),
		0o644,
	); err != nil {
		t.Fatalf("write retained archive: %v", err)
	}
	manifest, err := registry.ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("parse retained manifest: %v", err)
	}
	projection, err := json.MarshalIndent(registry.ProjectManifest(manifest), "", "  ")
	if err != nil {
		t.Fatalf("encode retained projection: %v", err)
	}
	projection = append(projection, '\n')
	if bytes.Contains(projection, []byte(`"requires_setup"`)) ||
		bytes.Contains(projection, []byte(`"limitations"`)) ||
		bytes.Contains(projection, []byte(`"executable_helpers"`)) {
		t.Fatalf("optional usability fields were not omitted:\n%s", projection)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "registry-entry.json"), projection, 0o644); err != nil {
		t.Fatalf("write retained projection: %v", err)
	}

	result, err := validateReleaseStore(releaseRoot, time.Now().UTC())
	if err != nil {
		t.Fatalf("validate retained release round trip: %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("validated %d releases, want 1", result.Count)
	}
}

func releaseArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write archive header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write archive content: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar archive: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip archive: %v", err)
	}
	return buffer.Bytes()
}
