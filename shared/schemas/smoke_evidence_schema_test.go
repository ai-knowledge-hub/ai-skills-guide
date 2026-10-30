package manifestschemas

import (
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSmokeEvidenceSchema(t *testing.T) {
	schemaBytes, err := ManifestFiles.ReadFile("smoke-evidence.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schemaDocument any
	if err := json.Unmarshal(schemaBytes, &schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	const schemaURL = "https://skills.ai-knowledge-hub.org/schemas/smoke-evidence.schema.json"
	if err := compiler.AddResource(schemaURL, schemaDocument); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	document := map[string]any{
		"schema_version": "skills-hub.smoke-evidence/v1", "contract_id": "operational-readiness", "contract_version": "1.0.3",
		"evidence_scope": "executable", "test_id": "skills-hub/declared-smoke", "test_version": "1", "producer": "skills-hub",
		"entry_id": "ads/example", "entry_version": "1.0.0", "artifact_sha256": stringOf("a", 64),
		"dependency_lock_sha256": "not-applicable", "install_source": "remote", "module": "tools", "runtime": "generic", "platform": "linux",
		"command": []any{"bin/example", "--smoke"}, "environment_id": stringOf("b", 32), "configuration_class": "no-authentication",
		"execution_runtime_fingerprint": stringOf("c", 64),
		"coverage":                      []any{"install-receipt", "installed-tree", "declared-smoke-test"},
		"started_at":                    "2026-09-17T10:00:00Z", "observed_at": "2026-09-17T10:00:01Z", "expires_at": "2026-12-16T10:00:01Z", "result": "passed", "promotion_eligible": true,
	}
	if err := schema.Validate(document); err != nil {
		t.Fatalf("valid smoke evidence rejected: %v", err)
	}
	delete(document, "environment_id")
	if err := schema.Validate(document); err == nil {
		t.Fatal("smoke evidence without environment identity was accepted")
	}
}

func stringOf(value string, count int) string {
	result := ""
	for range count {
		result += value
	}
	return result
}
