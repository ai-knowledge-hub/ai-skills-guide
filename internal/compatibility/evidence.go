package compatibility

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	manifestschemas "github.com/ai-knowledge-hub/ai-skills-guide/shared/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var schemaOnce sync.Once
var evidenceSchema *jsonschema.Schema
var schemaError error

// ParseEvidence validates representation, not producer trust. The caller must
// check raw bytes against independent producer verification/approval authority.
func ParseEvidence(payload []byte) (Evidence, error) {
	var e Evidence
	if len(payload) > 1024*1024 {
		return e, fmt.Errorf("compatibility evidence exceeds 1 MiB")
	}
	schemaOnce.Do(func() {
		data, err := manifestschemas.ManifestFiles.ReadFile("compatibility-evidence.schema.json")
		if err != nil {
			schemaError = err
			return
		}
		var document any
		if err = json.Unmarshal(data, &document); err != nil {
			schemaError = err
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		const url = "https://skills.ai-knowledge-hub.org/schemas/compatibility-evidence.schema.json"
		if err = compiler.AddResource(url, document); err != nil {
			schemaError = err
			return
		}
		evidenceSchema, schemaError = compiler.Compile(url)
	})
	if schemaError != nil {
		return e, schemaError
	}
	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		return e, fmt.Errorf("invalid compatibility evidence JSON")
	}
	if err := evidenceSchema.Validate(document); err != nil {
		return e, fmt.Errorf("compatibility evidence violates schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&e); err != nil {
		return e, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return e, fmt.Errorf("trailing evidence content")
	}
	if _, err := canonicalSubject(e.Subject); err != nil {
		return e, err
	}
	return e, nil
}
