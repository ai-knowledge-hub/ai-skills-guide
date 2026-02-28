package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const baseURL = "https://skills.ai-knowledge-hub.org"

func BuildIndex(root string) (Index, error) {
	manifests, err := findManifests(filepath.Join(root, "skills"))
	if err != nil {
		return Index{}, err
	}

	skills := make([]SkillEntry, 0, len(manifests))
	for _, manifestPath := range manifests {
		m, err := ParseManifest(manifestPath)
		if err != nil {
			return Index{}, err
		}
		skillDir := filepath.Dir(manifestPath)
		sha, err := digestSkillDir(skillDir)
		if err != nil {
			return Index{}, err
		}
		relManifest, err := filepath.Rel(root, manifestPath)
		if err != nil {
			return Index{}, fmt.Errorf("compute relative manifest path: %w", err)
		}
		relManifest = filepath.ToSlash(relManifest)

		entry := SkillEntry{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Category:    m.Category,
			Latest:      m.Version,
			Versions: []VersionEntry{{
				Version:     m.Version,
				ReleasedAt:  m.ReleasedAt,
				ManifestURL: fmt.Sprintf("%s/%s", baseURL, relManifest),
				ArtifactURL: fmt.Sprintf("%s/artifacts/%s/%s.tar.gz", baseURL, m.ID, m.Version),
				SHA256:      sha,
			}},
			Runtimes:   append([]string{}, m.Runtimes...),
			Tags:       append([]string{}, m.Tags...),
			Deprecated: m.Deprecated,
			ReplacedBy: m.ReplacedBy,
		}
		skills = append(skills, entry)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].ID < skills[j].ID
	})

	generatedAt := "1970-01-01T00:00:00Z"
	for _, skill := range skills {
		for _, version := range skill.Versions {
			if version.ReleasedAt > generatedAt {
				generatedAt = version.ReleasedAt
			}
		}
	}

	return Index{
		RegistryVersion: "1.0",
		GeneratedAt:     generatedAt,
		Skills:          skills,
	}, nil
}

func WriteIndex(path string, index Index) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write index %s: %w", path, err)
	}
	return nil
}

func findManifests(skillsRoot string) ([]string, error) {
	entries := make([]string, 0)
	err := filepath.WalkDir(skillsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "skill.yaml" {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk skills root %s: %w", skillsRoot, err)
	}
	sort.Strings(entries)
	return entries, nil
}

func digestSkillDir(skillDir string) (string, error) {
	type fileEntry struct {
		rel string
		abs string
	}
	files := make([]fileEntry, 0)
	err := filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if containsHiddenPart(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, fileEntry{rel: filepath.ToSlash(rel), abs: path})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk skill dir %s: %w", skillDir, err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].rel < files[j].rel
	})

	h := sha256.New()
	for _, f := range files {
		if _, err := io.WriteString(h, f.rel); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte{0}); err != nil {
			return "", err
		}
		data, err := os.ReadFile(f.abs)
		if err != nil {
			return "", fmt.Errorf("read skill file %s: %w", f.abs, err)
		}
		if _, err := h.Write(data); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte{0}); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func containsHiddenPart(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}
