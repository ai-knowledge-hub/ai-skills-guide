package registry

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

// The advertising lab's documented product boundary requires its OpenAI Ads
// read client and excludes live writes. A self-consistent rewrite of both
// includes and provider_dependencies must not silently redefine that boundary.
func advertisingLabContract(entry SkillEntry) error {
	const client = "adtech/openai-ads-api-client"
	if entry.Includes == nil || !slices.Contains(entry.Includes.Tools, client) {
		return fmt.Errorf("advertising lab requires included client %s", client)
	}
	found := false
	for _, dependency := range entry.ProviderDependencies {
		if dependency.Tool == client && dependency.Requirement == "required" && dependency.Access == "read-only" {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("advertising lab requires read-only provider authority from %s", client)
	}
	for _, capability := range entry.CapabilityReadiness {
		if capability.ID == "live-write" {
			if capability.Access != "read-write" || capability.Availability != "not-verified" {
				return fmt.Errorf("advertising lab cannot advertise implemented live writes")
			}
			return nil
		}
	}
	return fmt.Errorf("advertising lab must disclose unavailable live writes")
}

func TestAdvertisingLabPreservesRequiredProviderAndEffectBoundary(t *testing.T) {
	index, err := BuildPluginsIndex(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var baseline SkillEntry
	for _, entry := range index.Skills {
		if entry.ID == "marketing/chatgpt-advertising-experiment-plugin" {
			baseline = entry
		}
	}
	if err := advertisingLabContract(baseline); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*SkillEntry){
		"remove included client":    func(e *SkillEntry) { e.Includes.Tools = []string{"adtech/conversion-event-reconciler"} },
		"remove provider authority": func(e *SkillEntry) { e.ProviderDependencies = nil },
		"remove both sides and disable authentication": func(e *SkillEntry) {
			e.Includes.Tools = []string{"adtech/conversion-event-reconciler"}
			e.ProviderDependencies = nil
			e.Authentication.Status = "none"
			e.Authentication.Methods = []string{"none"}
		},
		"substitute valid provider on both sides": func(e *SkillEntry) {
			e.Includes.Tools = []string{"analytics/ga4-mcp-connector", "adtech/conversion-event-reconciler"}
			e.ProviderDependencies[0].Tool = "analytics/ga4-mcp-connector"
		},
		"expand provider authority": func(e *SkillEntry) { e.ProviderDependencies[0].Access = "read-write" },
		"promote live writes": func(e *SkillEntry) {
			for i := range e.CapabilityReadiness {
				if e.CapabilityReadiness[i].ID == "live-write" {
					e.CapabilityReadiness[i].Availability = "setup-required"
				}
			}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			// Deep copies keep coordinated mutations independent of map order.
			data, err := json.Marshal(baseline)
			if err != nil {
				t.Fatal(err)
			}
			var candidate SkillEntry
			if err := json.Unmarshal(data, &candidate); err != nil {
				t.Fatal(err)
			}
			mutate(&candidate)
			if err := advertisingLabContract(candidate); err == nil {
				t.Fatal("harmful relationship mutation escaped the independent product boundary")
			}
		})
	}
}
