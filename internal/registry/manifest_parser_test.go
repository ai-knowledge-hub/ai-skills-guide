package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
verification:
  security_reviewed: true
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
	if !m.SecurityReviewed {
		t.Fatal("expected security reviewed to be true")
	}
}

func TestParseManifestSchemaVersionCompatibility(t *testing.T) {
	t.Run("catalog 1.1 allows usability without v2 contract", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "skill.yaml")
		content := `schema_version: "1.1"
id: marketing/versioned-skill
name: Versioned Skill
description: Versioned catalog manifest used for compatibility testing.
version: 1.0.0
released_at: "2026-09-14T00:00:00Z"
category: marketing-tools/ads-ops
tags: [testing]
runtimes: [codex]
usability:
  availability: documentation-only
  execution: instructions
deprecated: false
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		if _, err := ParseManifest(path); err != nil {
			t.Fatalf("parse schema 1.1 manifest: %v", err)
		}
	})

	t.Run("v2.0 rejects new availability", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tool.yaml")
		content := "schema_version: \"2.0\"\n" + strings.Replace(
			unversionedV2ToolManifest,
			"availability: setup-required",
			"availability: not-verified",
			1,
		)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		if _, err := ParseManifest(path); err == nil || !strings.Contains(err.Error(), "schema_version 2.1") {
			t.Fatalf("expected version-gated availability error, got %v", err)
		}
	})

	t.Run("v2.0 rejects executable helpers", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tool.yaml")
		content := "schema_version: \"2.0\"\n" + strings.Replace(
			unversionedV2ToolManifest,
			"  execution: local-tool",
			`  execution: local-tool
  executable_helpers:
    - entrypoint: scripts/check.py
      availability: not-verified
      execution: local-tool
      limitations:
        - No current executable evidence.`,
			1,
		)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		if _, err := ParseManifest(path); err == nil || !strings.Contains(err.Error(), "executable_helpers") {
			t.Fatalf("expected version-gated executable helper error, got %v", err)
		}
	})
}

func TestParseToolManifestOperationalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.yaml")
	content := `id: analytics/example-tool
name: Example Tool
description: Example tool manifest for operational metadata parsing.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: tools-mcp/analytics
tags:
  - analytics
license: MIT
author:
  name: Example
runtimes:
  - codex
dependencies:
  mcp_servers:
    - ga4
operational:
  connected_system: Google Analytics 4
  capabilities:
    - Pull sessions by dimension.
    - Return normalized rows for dashboard input.
  auth_required:
    - GA4 property access via authenticated MCP runtime
  access_level: read-only
  trust_boundary: remote-mcp-server
  approval_boundary: Safe for read-only analytics retrieval only.
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Operational.ConnectedSystem != "Google Analytics 4" {
		t.Fatalf("unexpected connected system: %s", m.Operational.ConnectedSystem)
	}
	if len(m.Operational.Capabilities) != 2 {
		t.Fatalf("unexpected capabilities: %#v", m.Operational.Capabilities)
	}
	if len(m.Operational.AuthRequired) != 1 {
		t.Fatalf("unexpected auth requirements: %#v", m.Operational.AuthRequired)
	}
	if m.Operational.AccessLevel != "read-only" {
		t.Fatalf("unexpected access level: %s", m.Operational.AccessLevel)
	}
	if m.Operational.TrustBoundary != "remote-mcp-server" {
		t.Fatalf("unexpected trust boundary: %s", m.Operational.TrustBoundary)
	}
	if len(m.Dependencies.MCPServers) != 1 || m.Dependencies.MCPServers[0] != "ga4" {
		t.Fatalf("unexpected dependencies: %#v", m.Dependencies)
	}
}

func TestParseManifestUsability(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.yaml")
	content := `id: adtech/example-tool
name: Example Tool
description: Example tool manifest for usability parsing.
version: 0.1.0
released_at: "2026-07-25T00:00:00Z"
category: tools-mcp/adtech
tags:
  - example
license: MIT
author:
  name: Example
runtimes:
  - codex
usability:
  availability: setup-required
  execution: remote-integration
  requires_setup:
    - Configure an account-scoped key.
  limitations:
    - Dry-run mode is the default.
  quickstart: python3 scripts/client.py account
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Usability.Availability != "setup-required" || m.Usability.Execution != "remote-integration" {
		t.Fatalf("unexpected usability: %#v", m.Usability)
	}
	if len(m.Usability.RequiresSetup) != 1 || len(m.Usability.Limitations) != 1 {
		t.Fatalf("unexpected usability lists: %#v", m.Usability)
	}
}

func TestParseAgentManifestOperationalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yaml")
	content := `id: marketing/example-agent
name: Example Agent
description: Example agent manifest for operational metadata parsing.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: marketing-agents/performance
tags:
  - reporting
license: MIT
author:
  name: Example
runtimes:
  - codex
dependencies:
  agents:
    - marketing/creative-operating-system-supervisor
  skills:
    - adtech/dashboard-generator
  tools:
    - ga4_query
operational:
  role: Weekly reporting supervisor.
  coordinates:
    - Dashboard generation
    - QA gating
  autonomy_level: semi-autonomous
  approval_boundary: Require approval before external distribution.
  outputs:
    - Publish decision
    - Executive narrative
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Operational.Role != "Weekly reporting supervisor." {
		t.Fatalf("unexpected role: %s", m.Operational.Role)
	}
	if m.Operational.AutonomyLevel != "semi-autonomous" {
		t.Fatalf("unexpected autonomy level: %s", m.Operational.AutonomyLevel)
	}
	if len(m.Operational.Coordinates) != 2 {
		t.Fatalf("unexpected coordinates: %#v", m.Operational.Coordinates)
	}
	if len(m.Operational.Outputs) != 2 {
		t.Fatalf("unexpected outputs: %#v", m.Operational.Outputs)
	}
	if len(m.Dependencies.Agents) != 1 || m.Dependencies.Agents[0] != "marketing/creative-operating-system-supervisor" {
		t.Fatalf("unexpected agent dependencies: %#v", m.Dependencies)
	}
	if len(m.Dependencies.Skills) != 1 || m.Dependencies.Skills[0] != "adtech/dashboard-generator" {
		t.Fatalf("unexpected skill dependencies: %#v", m.Dependencies)
	}
}

func TestParseSkillManifestOperationalMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	content := `id: engineering/example-skill
name: Example Skill
description: Example skill manifest for operational metadata parsing.
version: 0.1.0
released_at: "2026-03-30T00:00:00Z"
category: engineering/code-maintenance
tags:
  - planning
license: MIT
author:
  name: Example
runtimes:
  - codex
dependencies:
  tools:
    - rg
operational:
  use_when: Use when repo scope analysis is needed before edits.
  execution_mode: read-only-local-inspection
  outputs:
    - Change strategy
    - Verification command list
  approval_boundary: Safe for inspection and planning before edits.
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Operational.UseWhen != "Use when repo scope analysis is needed before edits." {
		t.Fatalf("unexpected use_when: %s", m.Operational.UseWhen)
	}
	if m.Operational.ExecutionMode != "read-only-local-inspection" {
		t.Fatalf("unexpected execution mode: %s", m.Operational.ExecutionMode)
	}
	if len(m.Operational.Outputs) != 2 {
		t.Fatalf("unexpected outputs: %#v", m.Operational.Outputs)
	}
	if len(m.Dependencies.Tools) != 1 || m.Dependencies.Tools[0] != "rg" {
		t.Fatalf("unexpected tool dependencies: %#v", m.Dependencies)
	}
}

func TestParsePluginManifestIncludesAndRequires(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	content := `id: marketing/example-plugin
name: Example Plugin
description: Example plugin package for manifest parsing tests.
version: 0.1.0
released_at: "2026-03-28T00:00:00Z"
category: marketing-plugins/reporting
tags:
  - reporting
license: MIT
author:
  name: Example
runtimes:
  - codex
  - claude
entrypoints:
  spec: plugin.json
includes:
  skills:
    - marketing/meta-google-weekly-performance-review
  agents:
    - marketing/weekly-performance-supervisor
  tools:
    - analytics/ga4-mcp-connector
  hooks:
    - post-analysis-slack-summary
requires:
  secrets:
    - GA4_PROPERTY_ID
  approvals:
    - human-review-for-write-actions
verification:
  security_reviewed: false
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(m.Includes.Skills) != 1 || m.Includes.Skills[0] != "marketing/meta-google-weekly-performance-review" {
		t.Fatalf("unexpected included skills: %#v", m.Includes.Skills)
	}
	if len(m.Includes.Agents) != 1 || m.Includes.Agents[0] != "marketing/weekly-performance-supervisor" {
		t.Fatalf("unexpected included agents: %#v", m.Includes.Agents)
	}
	if len(m.Includes.Tools) != 1 || m.Includes.Tools[0] != "analytics/ga4-mcp-connector" {
		t.Fatalf("unexpected included tools: %#v", m.Includes.Tools)
	}
	if len(m.Includes.Hooks) != 1 || m.Includes.Hooks[0] != "post-analysis-slack-summary" {
		t.Fatalf("unexpected included hooks: %#v", m.Includes.Hooks)
	}
	if len(m.Requires.Secrets) != 1 || m.Requires.Secrets[0] != "GA4_PROPERTY_ID" {
		t.Fatalf("unexpected required secrets: %#v", m.Requires.Secrets)
	}
	if len(m.Requires.Approvals) != 1 || m.Requires.Approvals[0] != "human-review-for-write-actions" {
		t.Fatalf("unexpected required approvals: %#v", m.Requires.Approvals)
	}
}

func TestParseManifestV2Contracts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.yaml")
	content := `schema_version: "2.0"
id: adtech/example-v2-tool
name: Example V2 Tool
description: Example v2 tool manifest for contract metadata parsing.
version: 1.0.0
released_at: "2026-09-13T00:00:00Z"
category: tools-mcp/adtech
tags:
  - example
license: MIT
author:
  name: Example
runtimes:
  - codex
execution:
  kind: mcp-server
  command:
    - node
    - dist/server.js
  healthcheck:
    - node
    - dist/server.js
    - --healthcheck
  smoke_test:
    - node
    - dist/server.js
    - --smoke
  supported_platforms:
    - linux
    - macos
  supported_runtimes:
    - node22
artifact:
  self_contained: true
  dependency_lock: pnpm-lock.yaml
  checksums: checksums.txt
  sbom: sbom.cdx.json
authentication:
  status: required
  methods:
    - api-key
  credential_bindings:
    - PROVIDER_API_KEY
  scopes:
    - read
  setup_url: https://example.test/setup
  credential_storage: Runtime credential store.
  validation: Call the non-destructive identity endpoint.
  revocation: Revoke the key at the provider.
verification:
  security_reviewed: false
  evidence:
    - evidence://example/mcp-smoke
  last_verified_at: "2026-09-13T00:00:00Z"
deprecated: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.SchemaVersion != "2.0" || m.Execution.Kind != "mcp-server" {
		t.Fatalf("unexpected v2 execution contract: %#v", m.Execution)
	}
	if len(m.Execution.Command) != 2 || len(m.Execution.Healthcheck) != 3 || len(m.Execution.SmokeTest) != 3 {
		t.Fatalf("unexpected execution commands: %#v", m.Execution)
	}
	if m.Artifact.DependencyLock == nil || *m.Artifact.DependencyLock != "pnpm-lock.yaml" {
		t.Fatalf("unexpected artifact contract: %#v", m.Artifact)
	}
	if m.Authentication.Status != "required" || len(m.Authentication.CredentialBindings) != 1 {
		t.Fatalf("unexpected authentication contract: %#v", m.Authentication)
	}
	if len(m.Verification.Evidence) != 1 || m.Verification.LastVerifiedAt != "2026-09-13T00:00:00Z" {
		t.Fatalf("unexpected verification contract: %#v", m.Verification)
	}
}

func TestCredentialScannerRejectsSecretsInAllowedFields(t *testing.T) {
	fixturePath := filepath.Join(
		"..", "..", "shared", "schemas", "fixtures", "manifest-v2",
		"embedded-secrets.allowed-fields.json",
	)
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(values) != 7 {
		t.Fatalf("expected seven allowed-field secret mutations, got %d", len(values))
	}
	for field, value := range values {
		t.Run(field, func(t *testing.T) {
			if _, found := findCredentialShapedValue(value, "$."+field); !found {
				t.Fatalf("credential-shaped value in %s was not rejected", field)
			}
		})
	}
}

func TestCredentialScannerRejectsRepresentativeAuthenticationSecrets(t *testing.T) {
	mutations := map[string]string{
		"none":                          "sk_" + "live_51ReviewFixtureAbCdEf123456789",
		"api-key":                       "AIzaReviewFixtureKey1234567890",
		"bearer-token":                  "Bearer reviewfixturetoken1234567890",
		"oauth-authorization-code-pkce": "authorization_code=ReviewCodeAbCd1234567890",
		"oauth-device-flow":             "device_code=ReviewDeviceCode1234567890",
		"oauth-client-credentials":      "client_secret=ReviewClientSecret1234567890",
		"service-account":               "-----BEGIN PRIVATE KEY-----",
		"workload-identity":             "eyJabcdefgh.ijklmnop.qrstuvwx",
		"brokered":                      "xoxb-ReviewBrokeredToken1234",
		"custom":                        "password=ReviewPassword1234567890",
	}
	for method, value := range mutations {
		t.Run(method, func(t *testing.T) {
			content := "schema_version: \"2.0\"\n" + unversionedV2ToolManifest
			content = strings.Replace(content, "methods: [api-key]", "methods: ["+method+"]", 1)
			content = strings.Replace(content, "validation: Validate the configured identity.", "validation: "+strconv.Quote(value), 1)
			if method == "none" {
				content = strings.Replace(content, "status: required", "status: none", 1)
				content = strings.Replace(content, "credential_bindings: [PROVIDER_API_KEY]", "credential_bindings: []", 1)
				content = strings.Replace(content, "scopes: [read]", "scopes: []", 1)
			}
			path := filepath.Join(t.TempDir(), "tool.yaml")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			_, err := ParseManifest(path)
			if err == nil || !strings.Contains(err.Error(), "$.authentication.validation") {
				t.Fatalf("credential mutation for %s was not rejected: %v", method, err)
			}
			if strings.Contains(err.Error(), value) {
				t.Fatal("credential-shaped value was echoed in the validation error")
			}
		})
	}
}

func TestCredentialScannerAllowsBenignAuthenticationGuidance(t *testing.T) {
	for _, value := range []string{
		"Bearer authentication is supported.",
		"Store the token in the runtime credential store.",
		"Use the PROVIDER_API_KEY binding.",
		"api_key=PROVIDER_API_KEY",
		"token=not-applicable",
		"credential=NOT_CONFIGURED",
		"secret=runtime-managed.",
	} {
		if foundPath, found := findCredentialShapedValue(value, "$.guidance"); found {
			t.Fatalf("benign guidance was rejected at %s: %q", foundPath, value)
		}
	}
}

func TestParseManifestAllowsBenignCredentialSentinels(t *testing.T) {
	for _, sentinel := range []string{"not-applicable", "not-configured", "runtime-managed"} {
		t.Run(sentinel, func(t *testing.T) {
			content := "schema_version: \"2.0\"\n" + unversionedV2ToolManifest
			content = strings.Replace(
				content,
				"validation: Validate the configured identity.",
				"validation: token="+sentinel,
				1,
			)
			path := filepath.Join(t.TempDir(), "tool.yaml")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}
			manifest, err := ParseManifest(path)
			if err != nil {
				t.Fatalf("parse benign sentinel: %v", err)
			}
			if manifest.Authentication.Validation != "token="+sentinel {
				t.Fatalf("benign guidance was not preserved: %q", manifest.Authentication.Validation)
			}
		})
	}
}

func TestCredentialScannerChecksAssignmentsAfterBenignSentinel(t *testing.T) {
	value := "token=not-applicable; client_secret=ReviewClientSecret1234567890"
	if foundPath, found := findCredentialShapedValue(value, "$.authentication.validation"); !found {
		t.Fatalf("later credential assignment was masked by sentinel at %s", foundPath)
	}
}

func TestParseManifestRejectsCredentialShapedAllowedValue(t *testing.T) {
	fixturePath := filepath.Join(
		"..", "..", "shared", "schemas", "fixtures", "manifest-v2",
		"tool-secret-in-allowed-field.semantic-invalid.yaml",
	)
	_, err := ParseManifest(fixturePath)
	if err == nil || !strings.Contains(err.Error(), "$.authentication.credential_storage") {
		t.Fatalf("expected credential_storage rejection, got %v", err)
	}
	if strings.Contains(err.Error(), "reviewfixturetoken") {
		t.Fatal("secret-shaped value was echoed in the validation error")
	}
}

func TestParseManifestRejectsUnversionedV2Contract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(unversionedV2ToolManifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := ParseManifest(path)
	if err == nil || !strings.Contains(err.Error(), "declares v2 contract field $.execution without schema_version") {
		t.Fatalf("expected unversioned v2 rejection, got %v", err)
	}
}

func TestFirstV2ContractField(t *testing.T) {
	tests := map[string]map[string]any{
		"execution":             {"execution": map[string]any{}},
		"artifact":              {"artifact": map[string]any{}},
		"authentication":        {"authentication": map[string]any{}},
		"verification evidence": {"verification": map[string]any{"evidence": []any{}}},
		"verification timestamp": {"verification": map[string]any{
			"last_verified_at": "2026-09-13T00:00:00Z",
		}},
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, found := firstV2ContractField(raw); !found {
				t.Fatal("v2 contract field was not detected")
			}
		})
	}
	if field, found := firstV2ContractField(map[string]any{
		"verification": map[string]any{"security_reviewed": true},
	}); found {
		t.Fatalf("legacy verification field was classified as v2: %s", field)
	}
}

const unversionedV2ToolManifest = `id: adtech/unversioned-v2-tool
name: Unversioned V2 Tool
description: Complete v2-shaped tool manifest without a schema version.
version: 1.0.0
released_at: "2026-09-13T00:00:00Z"
category: tools-mcp/adtech
tags: [contract-test]
license: MIT
author:
  name: Tests
runtimes: [codex]
entrypoints:
  spec: TOOL.md
usability:
  availability: setup-required
  execution: local-tool
execution:
  kind: cli
  command: [bin/tool]
  smoke_test: [bin/tool, --smoke]
  supported_platforms: [linux]
  supported_runtimes: [native]
artifact:
  self_contained: true
  dependency_lock: null
  checksums: checksums.txt
  sbom: sbom.cdx.json
authentication:
  status: required
  methods: [api-key]
  credential_bindings: [PROVIDER_API_KEY]
  scopes: [read]
  credential_storage: Runtime credential store.
  validation: Validate the configured identity.
  revocation: Revoke the provider key.
verification:
  evidence: [evidence://tests/unversioned-v2]
  last_verified_at: "2026-09-13T00:00:00Z"
deprecated: false
`
