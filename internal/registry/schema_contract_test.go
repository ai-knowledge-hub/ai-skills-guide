package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestManifestSchemasEmbedCanonicalV2Definitions(t *testing.T) {
	root := filepath.Join("..", "..")
	canonical := readSchema(t, filepath.Join(root, "shared", "schemas", "manifest-contract-v2.schema.json"))
	canonicalDefs := schemaDefinitions(t, canonical)

	for _, name := range []string{
		"skill.schema.json",
		"agent.schema.json",
		"tool.schema.json",
		"plugin.schema.json",
		"registry-index.schema.json",
	} {
		t.Run(name, func(t *testing.T) {
			consumer := readSchema(t, filepath.Join(root, "shared", "schemas", name))
			consumerDefs := schemaDefinitions(t, consumer)
			for definition, want := range canonicalDefs {
				got, ok := consumerDefs[definition]
				if !ok {
					t.Fatalf("missing canonical definition %q", definition)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("definition %q has drifted from manifest-contract-v2.schema.json", definition)
				}
			}
		})
	}
}

func TestRegistrySchemaSharesAgentOrchestrationContract(t *testing.T) {
	root := filepath.Join("..", "..")
	agentDefs := schemaDefinitions(t, readSchema(t, filepath.Join(root, "shared", "schemas", "agent.schema.json")))
	registryDefs := schemaDefinitions(t, readSchema(t, filepath.Join(root, "shared", "schemas", "registry-index.schema.json")))
	for _, definition := range []string{"orchestration", "orchestration_profile", "orchestration_binding"} {
		if !reflect.DeepEqual(agentDefs[definition], registryDefs[definition]) {
			t.Fatalf("registry definition %q has drifted from agent.schema.json", definition)
		}
	}
}

func TestManifestSchemasRejectExecutableHelpersForV20(t *testing.T) {
	// Manifest contract v2: helpers are a 2.1+ extension. Exercise the real
	// validator rather than requiring a particular allOf/if/then arrangement.
	for _, module := range []string{"skill", "agent", "tool", "plugin"} {
		for _, version := range []string{"2.0", "2.1", "2.2"} {
			t.Run(module+"/"+version, func(t *testing.T) {
				data, err := os.ReadFile(filepath.Join("..", "..", "shared", "schemas", "fixtures", "manifest-v2", module+".valid.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				var document map[string]any
				if err := yaml.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				document["schema_version"] = version
				usability := document["usability"].(map[string]any)
				// not-verified is also a 2.1 extension; keep the control
				// classification valid on both sides of the helper boundary.
				if usability["availability"] == "not-verified" {
					usability["availability"] = "setup-required"
					usability["requires_setup"] = []string{"Configure the test runtime."}
				}
				manifestPath := filepath.Join(t.TempDir(), module+".yaml")
				writeDocument := func() {
					payload, err := json.Marshal(document)
					if err != nil {
						t.Fatal(err)
					}
					writeTestFile(t, manifestPath, string(payload), 0o644)
				}
				// A valid control prevents malformed fixture data from making a
				// rejection look like evidence for the version boundary.
				writeDocument()
				if err := validateManifestSchema(manifestPath); err != nil {
					t.Fatalf("control: %v", err)
				}
				document["usability"].(map[string]any)["executable_helpers"] = []any{map[string]any{
					"entrypoint": "scripts/check.py", "availability": "not-verified",
					"execution": "local-tool", "limitations": []string{"No executable evidence recorded."},
				}}
				writeDocument()
				if err := validateManifestSchema(manifestPath); (err == nil) != (version != "2.0") {
					t.Fatalf("helpers admitted=%v, schema %s: %v", err == nil, version, err)
				}
			})
		}
	}
}

func readSchema(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode schema %s: %v", path, err)
	}
	return schema
}

func schemaDefinitions(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	definitions, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("schema is missing $defs")
	}
	return definitions
}
