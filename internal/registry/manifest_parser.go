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
	nestedMapKey := ""
	currentNestedListKey := ""

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

		if strings.HasPrefix(line, "    - ") && inNestedMap && currentNestedListKey != "" {
			item := strings.TrimSpace(strings.TrimPrefix(line, "    - "))
			item = unquote(item)
			switch nestedMapKey {
			case "dependencies":
				switch currentNestedListKey {
				case "agents":
					out.Dependencies.Agents = append(out.Dependencies.Agents, item)
				case "skills":
					out.Dependencies.Skills = append(out.Dependencies.Skills, item)
				case "tools":
					out.Dependencies.Tools = append(out.Dependencies.Tools, item)
				case "apis":
					out.Dependencies.APIs = append(out.Dependencies.APIs, item)
				case "mcp_servers":
					out.Dependencies.MCPServers = append(out.Dependencies.MCPServers, item)
				}
			case "operational":
				switch currentNestedListKey {
				case "capabilities":
					out.Operational.Capabilities = append(out.Operational.Capabilities, item)
				case "auth_required":
					out.Operational.AuthRequired = append(out.Operational.AuthRequired, item)
				case "coordinates":
					out.Operational.Coordinates = append(out.Operational.Coordinates, item)
				case "outputs":
					out.Operational.Outputs = append(out.Operational.Outputs, item)
				}
			case "includes":
				switch currentNestedListKey {
				case "skills":
					out.Includes.Skills = append(out.Includes.Skills, item)
				case "agents":
					out.Includes.Agents = append(out.Includes.Agents, item)
				case "tools":
					out.Includes.Tools = append(out.Includes.Tools, item)
				case "hooks":
					out.Includes.Hooks = append(out.Includes.Hooks, item)
				}
			case "requires":
				switch currentNestedListKey {
				case "secrets":
					out.Requires.Secrets = append(out.Requires.Secrets, item)
				case "approvals":
					out.Requires.Approvals = append(out.Requires.Approvals, item)
				}
			case "usability":
				switch currentNestedListKey {
				case "requires_setup":
					out.Usability.RequiresSetup = append(out.Usability.RequiresSetup, item)
				case "limitations":
					out.Usability.Limitations = append(out.Usability.Limitations, item)
				}
			}
			continue
		}

		if strings.HasPrefix(line, "  ") {
			if inNestedMap {
				if strings.HasSuffix(strings.TrimSpace(line), ":") {
					nestedParts := strings.SplitN(strings.TrimSpace(line), ":", 2)
					currentNestedListKey = strings.TrimSpace(nestedParts[0])
					continue
				}
				nestedParts := strings.SplitN(strings.TrimSpace(line), ":", 2)
				if len(nestedParts) == 2 {
					nestedKey := strings.TrimSpace(nestedParts[0])
					nestedValue := unquote(strings.TrimSpace(nestedParts[1]))
					switch nestedMapKey {
					case "verification":
						if nestedKey == "security_reviewed" {
							out.SecurityReviewed = strings.EqualFold(nestedValue, "true")
						}
					case "operational":
						switch nestedKey {
						case "connected_system":
							out.Operational.ConnectedSystem = nestedValue
						case "access_level":
							out.Operational.AccessLevel = nestedValue
						case "trust_boundary":
							out.Operational.TrustBoundary = nestedValue
						case "approval_boundary":
							out.Operational.ApprovalBoundary = nestedValue
						case "role":
							out.Operational.Role = nestedValue
						case "autonomy_level":
							out.Operational.AutonomyLevel = nestedValue
						case "use_when":
							out.Operational.UseWhen = nestedValue
						case "execution_mode":
							out.Operational.ExecutionMode = nestedValue
						}
					case "usability":
						switch nestedKey {
						case "availability":
							out.Usability.Availability = nestedValue
						case "execution":
							out.Usability.Execution = nestedValue
						case "quickstart":
							out.Usability.Quickstart = nestedValue
						}
					}
					currentNestedListKey = ""
				}
			}
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
		nestedMapKey = ""
		currentNestedListKey = ""

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
		case "author", "entrypoints", "dependencies", "verification", "includes", "requires", "install", "operational", "usability":
			inNestedMap = true
			nestedMapKey = key
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
