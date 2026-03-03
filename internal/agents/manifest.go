package agents

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParseAgentManifest(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest %s: %w", path, err)
	}
	defer f.Close()

	var out Manifest
	scanner := bufio.NewScanner(f)
	currentList := ""
	section := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(line, "  ") && strings.HasSuffix(trimmed, ":") {
			key := strings.TrimSuffix(trimmed, ":")
			if section == "dependencies" {
				currentList = key
				continue
			}
			section = key
			currentList = ""
			continue
		}

		if strings.HasPrefix(line, "    - ") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "    - "))
			item = unquote(item)
			if section == "dependencies" {
				switch currentList {
				case "skills":
					out.DependencySkills = append(out.DependencySkills, item)
				case "tools":
					out.DependencyTools = append(out.DependencyTools, item)
				}
			}
			continue
		}

		if strings.HasPrefix(line, "  - ") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "  - "))
			item = unquote(item)
			switch currentList {
			case "tags":
				out.Tags = append(out.Tags, item)
			case "runtimes":
				out.Runtimes = append(out.Runtimes, item)
			}
			continue
		}

		if strings.HasPrefix(line, "  ") && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			_ = value
			if section == "dependencies" && value == "" {
				currentList = key
			}
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := unquote(strings.TrimSpace(parts[1]))
		section = ""
		currentList = ""

		switch key {
		case "id":
			out.ID = value
		case "name":
			out.Name = value
		case "description":
			out.Description = value
		case "version":
			out.Version = value
		case "released_at":
			out.ReleasedAt = value
		case "category":
			out.Category = value
		case "tags":
			currentList = "tags"
		case "runtimes":
			currentList = "runtimes"
		case "dependencies":
			section = "dependencies"
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, fmt.Errorf("scan manifest %s: %w", path, err)
	}

	if out.ID == "" || out.Name == "" || out.Version == "" {
		return Manifest{}, fmt.Errorf("manifest %s missing required id/name/version", path)
	}
	return out, nil
}

func unquote(v string) string {
	value := strings.TrimSpace(v)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
