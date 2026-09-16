package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	manifestschemas "github.com/ai-knowledge-hub/ai-skills-guide/shared/schemas"
	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

var (
	manifestSchemasOnce sync.Once
	manifestSchemas     map[string]*jsonschema.Schema
	manifestSchemasErr  error
)

func validateManifestSchema(manifestPath string) error {
	schemaName, err := schemaNameForManifest(manifestPath)
	if err != nil {
		return err
	}
	manifestSchemasOnce.Do(compileManifestSchemas)
	if manifestSchemasErr != nil {
		return fmt.Errorf("compile canonical manifest schemas: %w", manifestSchemasErr)
	}

	data, err := readYAMLDocument(manifestPath)
	if err != nil {
		return err
	}
	if err := manifestSchemas[schemaName].Validate(data); err != nil {
		return fmt.Errorf("manifest %s does not satisfy %s: %w", manifestPath, schemaName, err)
	}
	return nil
}

func compileManifestSchemas() {
	compiler := jsonschema.NewCompiler()
	compiler.UseRegexpEngine(compileECMAScriptRegexp)
	compiler.AssertFormat()
	manifestSchemas = make(map[string]*jsonschema.Schema, 4)
	for _, name := range []string{"skill.schema.json", "agent.schema.json", "tool.schema.json", "plugin.schema.json"} {
		data, err := manifestschemas.ManifestFiles.ReadFile(name)
		if err != nil {
			manifestSchemasErr = err
			return
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			manifestSchemasErr = fmt.Errorf("decode %s: %w", name, err)
			return
		}
		resourceURL := "https://skills.ai-knowledge-hub.org/schemas/" + name
		if err := compiler.AddResource(resourceURL, document); err != nil {
			manifestSchemasErr = fmt.Errorf("load %s: %w", name, err)
			return
		}
		schema, err := compiler.Compile(resourceURL)
		if err != nil {
			manifestSchemasErr = fmt.Errorf("compile %s: %w", name, err)
			return
		}
		manifestSchemas[name] = schema
	}
}

type ecmaScriptRegexp regexp2.Regexp

func (regexp *ecmaScriptRegexp) MatchString(value string) bool {
	matched, err := (*regexp2.Regexp)(regexp).MatchString(value)
	return err == nil && matched
}

func (regexp *ecmaScriptRegexp) String() string {
	return (*regexp2.Regexp)(regexp).String()
}

func compileECMAScriptRegexp(pattern string) (jsonschema.Regexp, error) {
	compiled, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*ecmaScriptRegexp)(compiled), nil
}

func readYAMLDocument(path string) (any, error) {
	manifestData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open manifest %s: %w", path, err)
	}
	var raw map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(manifestData))
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode manifest %s: %w", path, err)
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("normalize manifest %s: %w", path, err)
	}
	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode normalized manifest %s: %w", path, err)
	}
	return document, nil
}

func schemaNameForManifest(path string) (string, error) {
	switch filepath.Base(path) {
	case "skill.yaml":
		return "skill.schema.json", nil
	case "agent.yaml":
		return "agent.schema.json", nil
	case "tool.yaml":
		return "tool.schema.json", nil
	case "plugin.yaml":
		return "plugin.schema.json", nil
	default:
		return "", fmt.Errorf("manifest %s has no canonical module schema", path)
	}
}
