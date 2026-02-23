package registry

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParseManifest(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest %s: %w", path, err)
	}
	defer f.Close()

	var out Manifest
	scanner := bufio.NewScanner(f)
	currentScalarKey := ""
	currentListKey := ""
	inNestedMap := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(line, "  - ") {
			if currentListKey == "" {
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(line, "  - "))
			item = unquote(item)
			switch currentListKey {
			case "tags":
				out.Tags = append(out.Tags, item)
			case "runtimes":
				out.Runtimes = append(out.Runtimes, item)
			}
			continue
		}

		if strings.HasPrefix(line, "  ") {
			if currentScalarKey != "" && !strings.Contains(strings.TrimSpace(line), ":") {
				appendText := strings.TrimSpace(line)
				switch currentScalarKey {
				case "description":
					out.Description = strings.TrimSpace(out.Description + " " + appendText)
				}
				continue
			}
			if inNestedMap {
				continue
			}
		}

		if strings.HasPrefix(line, " ") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = unquote(value)

		currentScalarKey = ""
		currentListKey = ""
		inNestedMap = false

		switch key {
		case "id":
			out.ID = value
		case "name":
			out.Name = value
		case "description":
			out.Description = value
			currentScalarKey = "description"
		case "version":
			out.Version = value
		case "released_at":
			out.ReleasedAt = value
		case "category":
			out.Category = value
		case "tags":
			currentListKey = "tags"
		case "runtimes":
			currentListKey = "runtimes"
		case "deprecated":
			out.Deprecated = strings.EqualFold(value, "true")
		case "replaced_by":
			out.ReplacedBy = value
		case "author", "entrypoints", "dependencies", "verification":
			inNestedMap = true
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, fmt.Errorf("scan manifest %s: %w", path, err)
	}

	if err := validateManifestFields(out, path); err != nil {
		return Manifest{}, err
	}
	return out, nil
}

func validateManifestFields(m Manifest, path string) error {
	if m.ID == "" {
		return fmt.Errorf("manifest %s missing id", path)
	}
	if m.Name == "" {
		return fmt.Errorf("manifest %s missing name", path)
	}
	if m.Description == "" {
		return fmt.Errorf("manifest %s missing description", path)
	}
	if m.Version == "" {
		return fmt.Errorf("manifest %s missing version", path)
	}
	if m.ReleasedAt == "" {
		return fmt.Errorf("manifest %s missing released_at", path)
	}
	if len(m.Runtimes) == 0 {
		return fmt.Errorf("manifest %s missing runtimes", path)
	}
	if m.Category == "" {
		return fmt.Errorf("manifest %s missing category", path)
	}
	if len(m.Tags) == 0 {
		return fmt.Errorf("manifest %s missing tags", path)
	}
	if m.Deprecated && m.ReplacedBy == "" {
		return fmt.Errorf("manifest %s deprecated but missing replaced_by", path)
	}
	return nil
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
