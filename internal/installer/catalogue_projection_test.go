package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func TestWebFallbackProjectionMatchesRuntimePolicy(t *testing.T) {
	type fixture struct {
		Name  string              `json:"name"`
		Entry registry.SkillEntry `json:"entry"`
		Want  registry.SkillEntry `json:"want"`
	}
	now := time.Now().UTC()
	var fixtures []fixture
	// Current opaque evidence must be just as conservative as old, future or
	// missing observations. Literal expectations come from v3, not date math.
	for _, date := range []string{now.Format(time.RFC3339), "2000-01-01T00:00:00Z", "2099-01-01T00:00:00Z", ""} {
		for _, availability := range []string{"usable-now", "setup-required", "not-verified", "template-only", "documentation-only"} {
			for _, deprecated := range []bool{false, true} {
				manifest := registry.Manifest{
					SchemaVersion: "2.1", ID: "shared/projection-fixture", Version: "1.0.0", Deprecated: deprecated,
					Usability: registry.UsabilityMetadata{Availability: availability, Execution: "bundle", Source: "declared",
						ExecutableHelpers: []registry.ExecutableHelperMetadata{
							{Entrypoint: "bin/ready", Availability: "usable-now", Execution: "local-tool"},
							{Entrypoint: "bin/setup", Availability: "setup-required", Execution: "local-tool"},
							{Entrypoint: "bin/unknown", Availability: "not-verified", Execution: "local-tool"},
						},
					},
					Execution: registry.ExecutionMetadata{Kind: "bundle"}, Authentication: registry.AuthenticationMetadata{Status: "required"},
					Verification: registry.VerificationMetadata{Evidence: []string{"evidence://tests/opaque"}, LastVerifiedAt: date},
					CapabilityReadiness: []registry.CapabilityReadinessMetadata{
						{ID: "mock", Availability: "usable-now", Access: "offline"},
						{ID: "live-read", Availability: "setup-required", Access: "read-only"},
						{ID: "live-write", Availability: "not-verified", Access: "read-write"},
						{ID: "template", Availability: "template-only", Access: "offline"},
					},
				}
				entry := registry.ProjectManifest(manifest)
				want := registry.ProjectManifestForCurrentCatalog(manifest, now)
				expected := availability
				if availability == "usable-now" || (deprecated && availability == "setup-required") {
					expected = "not-verified"
				}
				if want.Usability.Availability != expected {
					t.Fatalf("Go oracle violates v3 aggregate rule: %#v", want)
				}
				for i, expected := range []string{"not-verified", "not-verified", "not-verified", "template-only"} {
					if want.CapabilityReadiness[i].Availability != expected {
						t.Fatalf("Go oracle violates v3 capability rule: %#v", want)
					}
				}
				for _, helper := range want.Usability.ExecutableHelpers {
					if helper.Availability != "not-verified" {
						t.Fatalf("Go oracle violates independent v3 helper rule: %#v", helper)
					}
				}
				fixtures = append(fixtures, fixture{Name: fmt.Sprintf("%s/%s/deprecated=%v", date, availability, deprecated), Entry: entry, Want: want})
			}
		}
	}
	payload, err := json.Marshal(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := filepath.Abs(filepath.Join("..", "..", "apps", "web", "scripts", "prepare-public-assets.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	// The marker keeps the publisher's main guard off while testing its export.
	script := `import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
const { projectCurrentCatalogEntry } = await import(pathToFileURL(process.argv[2]).href);
for (const { name, entry, want } of JSON.parse(readFileSync(0,"utf8"))) {
 const original=structuredClone(entry);
 const result=projectCurrentCatalogEntry(entry);
 assert.deepStrictEqual(result,want,name);
 assert.deepStrictEqual(entry,original,name+" mutated retained declarations");
 assert.deepStrictEqual(projectCurrentCatalogEntry(result),result,name+" is not idempotent");
}`
	command := exec.Command("node", "--input-type=module", "-e", script, "projection-test", publisher)
	command.Stdin = bytes.NewReader(payload)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fallback projection diverged: %v\n%s", err, output)
	}
}
