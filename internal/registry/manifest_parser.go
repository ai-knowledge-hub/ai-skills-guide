package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var credentialPatterns = []*regexp.Regexp{
	// Authorization payloads and private keys.
	regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]{24,}`),
	regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]{8,}\.[a-zA-Z0-9_-]{8,}\.[a-zA-Z0-9_-]{8,}\b`),
	regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`),
	// Provider-issued keys and tokens with stable, high-confidence prefixes.
	regexp.MustCompile(`\bgh[pousr]_[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bgithub_pat_[a-zA-Z0-9_]{20,}\b`),
	regexp.MustCompile(`\bglpat-[a-zA-Z0-9_-]{20,}\b`),
	regexp.MustCompile(`\bsk-(?:proj-)?[a-zA-Z0-9_-]{16,}\b`),
	regexp.MustCompile(`\b(?:sk|rk)_(?:live|test)_[a-zA-Z0-9]{16,}\b`),
	regexp.MustCompile(`\bSG\.[a-zA-Z0-9_-]{16,}\.[a-zA-Z0-9_-]{16,}\b`),
	regexp.MustCompile(`\bnpm_[a-zA-Z0-9]{20,}\b`),
	regexp.MustCompile(`\bpypi-AgEIcHlwaS5vcmc[a-zA-Z0-9_-]{16,}\b`),
	regexp.MustCompile(`\bshp(?:at|ca|pa|ss)_[a-fA-F0-9]{20,}\b`),
	regexp.MustCompile(`\bdop_v1_[a-fA-F0-9]{40,}\b`),
	regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{20,}\b`),
	regexp.MustCompile(`\bxox[baprs]-[0-9A-Za-z-]{12,}\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
}

// Provider-specific formats are not exhaustive. Catch high-confidence secret
// assignments without treating binding identifiers as credential payloads.
var credentialAssignmentPattern = regexp.MustCompile(`(?i)\b(?:api[_-]?key|access[_-]?key|authorization[_-]?code|client[_-]?secret|credential|device[_-]?code|password|refresh[_-]?token|secret|token)\s*[:=]\s*["']?([a-z0-9._~+/=-]{12,})`)

var bindingReferencePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

var nonSecretCredentialSentinels = map[string]struct{}{
	"externally-managed": {},
	"managed-externally": {},
	"no-credential":      {},
	"no-credentials":     {},
	"not-applicable":     {},
	"not-configured":     {},
	"not-required":       {},
	"placeholder":        {},
	"redacted":           {},
	"runtime-managed":    {},
	"unset":              {},
}

func ParseManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open manifest %s: %w", path, err)
	}

	var raw map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&raw); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return Manifest{}, fmt.Errorf("decode trailing YAML in manifest %s: %w", path, err)
		}
		return Manifest{}, fmt.Errorf("manifest %s contains multiple YAML documents", path)
	}
	if raw == nil {
		return Manifest{}, fmt.Errorf("manifest %s is empty", path)
	}

	if secretPath, found := findCredentialShapedValue(raw, "$"); found {
		return Manifest{}, fmt.Errorf("manifest %s contains a credential-shaped value at %s; use a binding name", path, secretPath)
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return Manifest{}, fmt.Errorf("normalize manifest %s: %w", path, err)
	}
	var out Manifest
	if err := json.Unmarshal(payload, &out); err != nil {
		return Manifest{}, fmt.Errorf("decode normalized manifest %s: %w", path, err)
	}

	out.executionSet = hasMap(raw, "execution")
	out.artifactSet = hasMap(raw, "artifact")
	out.authenticationSet = hasMap(raw, "authentication")
	out.verificationSet = hasMap(raw, "verification")
	out.usabilitySet = hasMap(raw, "usability")
	if verification, ok := raw["verification"].(map[string]any); ok {
		if reviewed, ok := verification["security_reviewed"].(bool); ok {
			out.SecurityReviewed = reviewed
		}
	}
	if out.SchemaVersion == "" {
		if contractField, found := firstV2ContractField(raw); found {
			return Manifest{}, fmt.Errorf("manifest %s declares v2 contract field %s without schema_version", path, contractField)
		}
	}

	if err := validateManifestFields(out, path); err != nil {
		return Manifest{}, err
	}
	return out, nil
}

func firstV2ContractField(raw map[string]any) (string, bool) {
	for _, field := range []string{"execution", "artifact", "authentication"} {
		if _, found := raw[field]; found {
			return "$." + field, true
		}
	}
	if verification, ok := raw["verification"].(map[string]any); ok {
		for _, field := range []string{"evidence", "last_verified_at"} {
			if _, found := verification[field]; found {
				return "$.verification." + field, true
			}
		}
	}
	return "", false
}

func hasMap(raw map[string]any, key string) bool {
	_, ok := raw[key].(map[string]any)
	return ok
}

func findCredentialShapedValue(value any, path string) (string, bool) {
	switch typed := value.(type) {
	case string:
		for _, pattern := range credentialPatterns {
			if pattern.MatchString(typed) {
				return path, true
			}
		}
		for _, match := range credentialAssignmentPattern.FindAllStringSubmatch(typed, -1) {
			if !isNonSecretCredentialValue(match[1]) {
				return path, true
			}
		}
	case []any:
		for index, item := range typed {
			if foundPath, found := findCredentialShapedValue(item, path+"["+strconv.Itoa(index)+"]"); found {
				return foundPath, true
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if foundPath, found := findCredentialShapedValue(typed[key], path+"."+key); found {
				return foundPath, true
			}
		}
	}
	return "", false
}

func isNonSecretCredentialValue(value string) bool {
	trimmed := strings.Trim(value, `"'.,;:()[]{}`)
	if bindingReferencePattern.MatchString(trimmed) {
		return true
	}
	normalized := strings.ToLower(strings.ReplaceAll(trimmed, "_", "-"))
	_, found := nonSecretCredentialSentinels[normalized]
	return found
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
	if m.SchemaVersion != "" {
		if m.SchemaVersion != "1.1" && m.SchemaVersion != "2.0" && m.SchemaVersion != "2.1" {
			return fmt.Errorf("manifest %s has unsupported schema_version %q", path, m.SchemaVersion)
		}
		if m.SchemaVersion == "2.0" && m.Usability.Availability == "not-verified" {
			return fmt.Errorf("manifest %s must use schema_version 2.1 for usability availability not-verified", path)
		}
		if m.SchemaVersion == "2.0" && len(m.Usability.ExecutableHelpers) > 0 {
			return fmt.Errorf("manifest %s must use schema_version 1.1 or 2.1 for usability executable_helpers", path)
		}
		if m.SchemaVersion == "1.1" {
			if m.executionSet || m.artifactSet || m.authenticationSet || len(m.Verification.Evidence) > 0 || m.Verification.LastVerifiedAt != "" {
				return fmt.Errorf("manifest %s uses v2 contract fields with schema_version 1.1", path)
			}
			return nil
		}
		if !m.executionSet || !m.artifactSet || !m.authenticationSet || !m.verificationSet {
			return fmt.Errorf("manifest %s is missing a required v2 contract section", path)
		}
		if m.Execution.Kind == "" || len(m.Execution.SupportedPlatforms) == 0 || len(m.Execution.SupportedRuntimes) == 0 {
			return fmt.Errorf("manifest %s has incomplete execution contract", path)
		}
		switch m.Execution.Kind {
		case "instructions", "bundle", "integration-template":
			if len(m.Execution.Command) > 0 || len(m.Execution.Healthcheck) > 0 || len(m.Execution.SmokeTest) > 0 {
				return fmt.Errorf("manifest %s non-executable contract cannot declare commands or checks", path)
			}
		case "script", "cli", "orchestrator":
			if len(m.Execution.Command) == 0 || len(m.Execution.SmokeTest) == 0 || len(m.Execution.Healthcheck) > 0 {
				return fmt.Errorf("manifest %s executable contract requires command and smoke_test but no healthcheck", path)
			}
		case "mcp-server", "service":
			if len(m.Execution.Command) == 0 || len(m.Execution.Healthcheck) == 0 || len(m.Execution.SmokeTest) == 0 {
				return fmt.Errorf("manifest %s service contract requires command, healthcheck, and smoke_test", path)
			}
		default:
			return fmt.Errorf("manifest %s has unsupported execution kind %q", path, m.Execution.Kind)
		}
		if m.Authentication.Status == "" || len(m.Authentication.Methods) == 0 {
			return fmt.Errorf("manifest %s has incomplete authentication contract", path)
		}
		if m.Authentication.Status == "required" && len(m.Authentication.CredentialBindings) == 0 {
			return fmt.Errorf("manifest %s required authentication has no credential binding", path)
		}
		if m.Authentication.Status == "none" && (len(m.Authentication.Methods) != 1 || m.Authentication.Methods[0] != "none") {
			return fmt.Errorf("manifest %s authentication status none requires only the none method", path)
		}
		if m.Authentication.Status != "none" && containsString(m.Authentication.Methods, "none") {
			return fmt.Errorf("manifest %s authenticated contract cannot include the none method", path)
		}
		if len(m.Verification.Evidence) == 0 || m.Verification.LastVerifiedAt == "" {
			return fmt.Errorf("manifest %s has incomplete verification contract", path)
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
