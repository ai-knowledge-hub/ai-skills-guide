package manifestschemas

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRuntimePackageSchemaAcceptsCompilerGoldens(t *testing.T) {
	schemaBytes, err := ManifestFiles.ReadFile("runtime-package.schema.json")
	if err != nil {
		t.Fatalf("read runtime package schema: %v", err)
	}
	var schemaDocument any
	if err := json.Unmarshal(schemaBytes, &schemaDocument); err != nil {
		t.Fatalf("decode runtime package schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	const schemaURL = "https://skills.ai-knowledge-hub.org/schemas/runtime-package.schema.json"
	if err := compiler.AddResource(schemaURL, schemaDocument); err != nil {
		t.Fatalf("load runtime package schema: %v", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatalf("compile runtime package schema: %v", err)
	}
	for _, runtimeName := range []string{"codex", "claude", "generic"} {
		t.Run(runtimeName, func(t *testing.T) {
			path := filepath.Join("..", "..", "internal", "installer", "testdata", "runtime", runtimeName+".runtime.json")
			payload, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read compiler golden: %v", err)
			}
			var document any
			if err := json.Unmarshal(payload, &document); err != nil {
				t.Fatalf("decode compiler golden: %v", err)
			}
			if err := schema.Validate(document); err != nil {
				t.Fatalf("compiler golden does not satisfy runtime package schema: %v", err)
			}
		})
	}
}
