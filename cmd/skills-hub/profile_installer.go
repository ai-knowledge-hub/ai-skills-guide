//go:build !runtime_readiness

package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

const cliBuildProfile = "installer"

func runReadinessCommand(command string, _ []string) error {
	return fmt.Errorf("%s is not available in the installation-only CLI; runtime readiness is deferred", command)
}
func readinessUsage() []string {
	return []string{"", "Installation does not verify authentication, model capabilities, or runtime operation."}
}
func registerCatalogueReadinessFlags(_ *flag.FlagSet) catalogueReadinessFlags {
	return catalogueReadinessFlags{}
}
func deriveCatalogueStatus(_ context.Context, _ catalogueReadinessFlags, _ installer.RuntimeTarget, _ string, _ registry.SkillEntry, _ string) (catalogueStatus, error) {
	return catalogueStatus{}, fmt.Errorf("runtime readiness is not available in the installation-only CLI")
}
func printProviderAuthNextStep(w io.Writer, entry registry.SkillEntry, _, _, _ string) {
	for _, dependency := range entry.ProviderDependencies {
		fmt.Fprintf(w, "install.auth.provider: %s (%s, %s)\n", dependency.Tool, dependency.Requirement, dependency.Access)
	}
	if len(entry.ProviderDependencies) > 0 || (entry.Authentication != nil && entry.Authentication.Status != "none") {
		fmt.Fprintln(w, "install.next_step: review the installed README and declared setup requirements in your adopting runtime; provider authentication has not been verified")
	}
}
