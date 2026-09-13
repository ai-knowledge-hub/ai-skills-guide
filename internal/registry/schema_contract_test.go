package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
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
