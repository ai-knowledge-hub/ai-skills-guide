package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCatalogueDeclarationsMatchRetainedRelease(t *testing.T) {
	// Immutable published records are an independent oracle for existing
	// releases, including instruction-only skills outside the named anchors.
	root := filepath.Join("..", "..")
	for _, module := range []struct {
		directory string
		build     func(string) (Index, error)
	}{
		{"skills", BuildSkillsIndex}, {"agents", BuildAgentsIndex},
		{"tools-mcp", BuildToolsIndex}, {"plugins", BuildPluginsIndex},
	} {
		t.Run(module.directory, func(t *testing.T) {
			index, err := module.build(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range index.Skills {
				file := filepath.Join(root, "releases", module.directory, filepath.FromSlash(entry.ID), entry.Latest, "registry-entry.json")
				payload, err := os.ReadFile(file)
				if os.IsNotExist(err) {
					continue
				} // New unpublished versions have no historical oracle yet.
				if err != nil {
					t.Fatal(err)
				}
				var retained SkillEntry
				if err := json.Unmarshal(payload, &retained); err != nil {
					t.Fatal(err)
				}
				// Release selection and URLs are attached by the builder; all
				// manifest-derived declarations must match the retained record.
				entry.Latest, entry.Versions = "", nil
				if !reflect.DeepEqual(entry, retained) {
					t.Errorf("%s declarations differ from immutable release %s", entry.ID, file)
				}
			}
		})
	}
}

func TestCatalogueAdditionPreservesExistingEntries(t *testing.T) {
	for _, module := range []struct {
		kind  string
		build func(string) (Index, error)
	}{
		{"skill", BuildSkillsIndex}, {"agent", BuildAgentsIndex},
		{"tool", BuildToolsIndex}, {"plugin", BuildPluginsIndex},
	} {
		t.Run(module.kind, func(t *testing.T) {
			root := t.TempDir()
			writeTestModule(t, root, module.kind, "shared/existing", "codex", "")
			before, err := module.build(root)
			if err != nil {
				t.Fatal(err)
			}
			writeTestModule(t, root, module.kind, "shared/additional", "codex", "")
			after, err := module.build(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := requireCatalogueIdentities(after, []string{"shared/existing", "shared/additional"}); err != nil {
				t.Fatal(err)
			}
			for _, entry := range after.Skills {
				if entry.ID == "shared/existing" && !reflect.DeepEqual(entry, before.Skills[0]) {
					t.Fatal("unrelated addition changed existing release identity or declarations")
				}
			}
		})
	}
}

func TestCatalogueWorkflowRequiresNamedDependencyDespiteCoordinatedDeletion(t *testing.T) {
	// This fixture's independently stated workflow needs shared/report. A
	// self-consistent agent with that dependency removed is still incomplete.
	for _, mutation := range []string{"none", "unrelated addition", "remove dependency", "same-count substitution", "remove dependency and edge"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			skill := writeTestModule(t, root, "skill", "shared/report", "codex", "")
			writeTestModule(t, root, "agent", "shared/reporting-agent", "codex", "dependencies:\n  skills: [shared/report]\n")
			if mutation == "unrelated addition" {
				writeTestModule(t, root, "skill", "shared/unrelated", "codex", "")
			}
			if mutation != "none" && mutation != "unrelated addition" {
				if err := os.RemoveAll(filepath.Dir(skill)); err != nil {
					t.Fatal(err)
				}
			}
			if mutation == "same-count substitution" {
				writeTestModule(t, root, "skill", "shared/other-report", "codex", "")
			}
			if mutation == "remove dependency and edge" {
				writeTestModule(t, root, "agent", "shared/reporting-agent", "codex", "")
			}
			_, admissionErr := ValidateRepository(root)
			index, err := BuildSkillsIndex(root)
			if err != nil {
				t.Fatal(err)
			}
			membershipErr := requireCatalogueIdentities(index, []string{"shared/report"})
			switch mutation {
			case "none", "unrelated addition":
				if admissionErr != nil || membershipErr != nil {
					t.Fatalf("valid growth rejected: %v; %v", admissionErr, membershipErr)
				}
			case "remove dependency and edge":
				if admissionErr != nil {
					t.Fatalf("coordinated deletion should be structurally self-consistent: %v", admissionErr)
				}
				if membershipErr == nil {
					t.Fatal("independent workflow requirement lost with both node and edge")
				}
			default:
				if admissionErr == nil || membershipErr == nil {
					t.Fatalf("broken closure escaped a boundary: %v; %v", admissionErr, membershipErr)
				}
			}
		})
	}
}
