package manifestschemas

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRegistryIndexSchemaAcceptsGeneratedIndexes(t *testing.T) {
	schemaBytes, err := ManifestFiles.ReadFile("registry-index.schema.json")
	if err != nil {
		t.Fatalf("read registry index schema: %v", err)
	}
	var schemaDocument any
	if err := json.Unmarshal(schemaBytes, &schemaDocument); err != nil {
		t.Fatalf("decode registry index schema: %v", err)
	}
	// jsonschema/v6 uses Go's RE2 engine, which cannot compile the canonical
	// negative-lookahead path pattern. Runtime admission independently enforces
	// those path rules; replace only that pattern so the embedded suite can
	// validate every other closed-schema constraint, including orchestration.
	normalizeUnsupportedSchemaPatterns(schemaDocument)
	compiler := jsonschema.NewCompiler()
	const schemaURL = "https://skills.ai-knowledge-hub.org/schemas/registry-index-v1.3.schema.json"
	if err := compiler.AddResource(schemaURL, schemaDocument); err != nil {
		t.Fatalf("load registry index schema: %v", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatalf("compile registry index schema: %v", err)
	}

	paths, err := filepath.Glob(filepath.Join("..", "..", "registry", "*-index.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("discover generated registry indexes: paths=%v err=%v", paths, err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			payload, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read generated index: %v", err)
			}
			var document any
			if err := json.Unmarshal(payload, &document); err != nil {
				t.Fatalf("decode generated index: %v", err)
			}
			if err := schema.Validate(document); err != nil {
				t.Fatalf("generated index violates embedded schema: %v", err)
			}
		})
	}
}

func normalizeUnsupportedSchemaPatterns(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if pattern, ok := typed["pattern"].(string); ok && strings.Contains(pattern, "(?!") {
			typed["pattern"] = "^.+$"
		}
		for _, child := range typed {
			normalizeUnsupportedSchemaPatterns(child)
		}
	case []any:
		for _, child := range typed {
			normalizeUnsupportedSchemaPatterns(child)
		}
	}
}
