package skills

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var errNoFrontmatter = errors.New("no frontmatter block")

type Frontmatter struct {
	Name        string
	Description string
	Deprecated  bool
	ReplacedBy  string
}

func parseFrontmatter(path string) (Frontmatter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseFrontmatterContent(string(content))
}

func parseFrontmatterContent(content string) (Frontmatter, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return Frontmatter{}, errNoFrontmatter
	}

	fm := Frontmatter{}
	seenClosing := false
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			seenClosing = true
			break
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")

		switch key {
		case "name":
			fm.Name = value
		case "description":
			fm.Description = value
		case "deprecated":
			parsed, parseErr := strconv.ParseBool(value)
			if parseErr != nil {
				return Frontmatter{}, fmt.Errorf("invalid deprecated value %q: %w", value, parseErr)
			}
			fm.Deprecated = parsed
		case "replaced_by":
			fm.ReplacedBy = value
		}
	}

	if !seenClosing {
		return Frontmatter{}, errNoFrontmatter
	}
	return fm, nil
}
