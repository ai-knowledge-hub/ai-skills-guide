package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func LoadIndex(path string) (Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Index{}, fmt.Errorf("read registry index %s: %w", path, err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, fmt.Errorf("parse registry index %s: %w", path, err)
	}
	return idx, nil
}

func FindSkill(idx Index, id string) (SkillEntry, bool) {
	for _, s := range idx.Skills {
		if s.ID == id {
			return s, true
		}
	}
	return SkillEntry{}, false
}

func ResolveVersion(skill SkillEntry, requested string) (VersionEntry, error) {
	version := strings.TrimSpace(requested)
	if version == "" || version == "latest" {
		version = skill.Latest
	}
	for _, v := range skill.Versions {
		if v.Version == version {
			return v, nil
		}
	}
	return VersionEntry{}, fmt.Errorf("version %s not found for %s", version, skill.ID)
}

type SearchQuery struct {
	Text     string
	Tag      string
	Category string
	Runtime  string
}

func Search(idx Index, q SearchQuery) []SkillEntry {
	text := strings.ToLower(strings.TrimSpace(q.Text))
	tag := strings.ToLower(strings.TrimSpace(q.Tag))
	category := strings.ToLower(strings.TrimSpace(q.Category))
	runtime := strings.ToLower(strings.TrimSpace(q.Runtime))

	out := make([]SkillEntry, 0)
	for _, s := range idx.Skills {
		if text != "" {
			haystack := strings.ToLower(s.ID + " " + s.Name + " " + s.Description)
			if !strings.Contains(haystack, text) {
				continue
			}
		}
		if tag != "" && !containsFold(s.Tags, tag) {
			continue
		}
		if category != "" && strings.ToLower(s.Category) != category {
			continue
		}
		if runtime != "" && !containsFold(s.Runtimes, runtime) {
			continue
		}
		out = append(out, s)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func containsFold(items []string, needle string) bool {
	for _, item := range items {
		if strings.EqualFold(item, needle) {
			return true
		}
	}
	return false
}
