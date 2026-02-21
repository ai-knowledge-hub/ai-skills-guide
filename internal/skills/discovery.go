package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func Discover(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read skills root %s: %w", root, err)
	}

	var out []Skill
	for _, categoryEntry := range entries {
		if !categoryEntry.IsDir() {
			continue
		}

		category := categoryEntry.Name()
		categoryPath := filepath.Join(root, category)
		skillEntries, err := os.ReadDir(categoryPath)
		if err != nil {
			return nil, fmt.Errorf("read category %s: %w", categoryPath, err)
		}
		for _, skillEntry := range skillEntries {
			if !skillEntry.IsDir() {
				continue
			}
			slug := skillEntry.Name()
			skillPath := filepath.Join(categoryPath, slug)

			fm, fmErr := parseFrontmatter(filepath.Join(skillPath, "SKILL.md"))
			if fmErr != nil {
				fm = Frontmatter{}
			}

			out = append(out, Skill{
				ID:          filepath.ToSlash(filepath.Join(category, slug)),
				Category:    category,
				Slug:        slug,
				Path:        skillPath,
				Name:        fm.Name,
				Description: fm.Description,
				Deprecated:  fm.Deprecated,
				ReplacedBy:  fm.ReplacedBy,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func FindByID(all []Skill, id string) (Skill, bool) {
	for _, skill := range all {
		if skill.ID == id {
			return skill, true
		}
	}
	return Skill{}, false
}
