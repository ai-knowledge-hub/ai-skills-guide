package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompatibilityDeclarationsAreVersionedAndNotEvidence(t *testing.T) {
	for _, module := range []string{"skill", "agent", "tool", "plugin"} {
		t.Run(module, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "shared", "schemas", "fixtures", "manifest-v2", module+".valid.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			base := strings.Replace(string(data), `schema_version: "2.0"`, `schema_version: "2.2"`, 1)
			base = strings.Replace(base, `schema_version: "2.1"`, `schema_version: "2.2"`, 1)
			declaration := "\ncompatibility:\n  - kind: api\n    name: Example Data API\n    versions: [v1beta]\n  - kind: protocol\n    name: Rolling service\n    capabilities: [bounded-query]\n"
			path := filepath.Join(t.TempDir(), module+".yaml")
			writeTestFile(t, path, base+declaration, 0o644)
			if err := validateManifestSchema(path); err != nil {
				t.Fatal(err)
			}
			manifest, err := ParseManifest(path)
			if err != nil {
				t.Fatal(err)
			}
			entry := ProjectManifest(manifest)
			if len(entry.Compatibility) != 2 || !reflect.DeepEqual(entry.Compatibility, manifest.Compatibility) {
				t.Fatal("declarations lost during projection")
			}
			payload, err := json.Marshal(Index{RegistryVersion: "1.3", Skills: []SkillEntry{entry}})
			if err != nil {
				t.Fatal(err)
			}
			index, err := ParseIndex(payload, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(index.Skills[0].Compatibility, entry.Compatibility) {
				t.Fatal("registry round trip lost compatibility")
			}
			for _, version := range []string{"2.0", "2.1"} {
				writeTestFile(t, path, strings.Replace(base, `schema_version: "2.2"`, `schema_version: "`+version+`"`, 1)+declaration, 0o644)
				if err := validateManifestSchema(path); err == nil {
					t.Fatalf("old schema %s accepted compatibility", version)
				}
			}
			for _, invalid := range []string{
				strings.Replace(declaration, "versions: [v1beta]", "versions: ['>=1.0']", 1),
				strings.Replace(declaration, "kind: api", "kind: invented", 1),
				declaration + "    status: tested-compatible\n",
				"\ncompatibility:\n  - kind: sdk\n    name: Unknown client\n",
			} {
				writeTestFile(t, path, base+invalid, 0o644)
				if err := validateManifestSchema(path); err == nil {
					t.Fatal("unsupported comparison or self-issued result admitted")
				}
			}
		})
	}
}
