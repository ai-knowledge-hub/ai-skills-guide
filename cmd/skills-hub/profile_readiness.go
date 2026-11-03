//go:build runtime_readiness

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/readiness"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

const cliBuildProfile = "readiness-development"

func runReadinessCommand(command string, args []string) error {
	switch command {
	case "auth":
		return runAuth(args)
	case "doctor":
		return runDoctor(args)
	case "smoke":
		return runSmoke(args)
	case "run-agent":
		return runAgent(args)
	}
	return fmt.Errorf("unknown readiness command: %s", command)
}
func readinessUsage() []string {
	return []string{
		"", "Development-only runtime commands (not included in installer releases):",
		"  auth      Configure or inspect runtime-owned authentication bindings",
		"  doctor    Diagnose installation, platform, and authentication readiness",
		"  smoke     Run the declared smoke check and record evidence",
		"  run-agent Run preflight checks for an installed agent package",
	}
}
func registerCatalogueReadinessFlags(fs *flag.FlagSet) catalogueReadinessFlags {
	return catalogueReadinessFlags{
		runtime:  fs.String("runtime", "", "derive local evidence-backed status for this runtime"),
		target:   fs.String("target", "", "installed module directory used for local status"),
		stateDir: fs.String("state-dir", defaultStateDir(), "local readiness evidence directory"),
	}
}
func deriveCatalogueStatus(ctx context.Context, flags catalogueReadinessFlags, target installer.RuntimeTarget, module string, entry registry.SkillEntry, version string) (catalogueStatus, error) {
	derived, err := readiness.DeriveCatalogStatus(ctx, readiness.InspectOptions{StateDir: *flags.stateDir, Module: module, Entry: entry, Version: version, PackageDir: filepath.Join(target.TargetPath, filepath.FromSlash(entry.ID)), TargetRoot: target.TargetPath, Runtime: target.Runtime})
	return catalogueStatus{Availability: derived.Availability, EvidenceScope: derived.EvidenceScope, ExpiresAt: derived.ExpiresAt, Reason: derived.Reason}, err
}

func printProviderAuthNextStep(w io.Writer, entry registry.SkillEntry, version, runtimeName, pluginTargetRoot string) {
	if len(entry.ProviderDependencies) == 0 {
		return
	}
	for _, dependency := range entry.ProviderDependencies {
		fmt.Fprintf(w, "install.auth.provider: %s (%s, %s)\n", dependency.Tool, dependency.Requirement, dependency.Access)
	}
	fmt.Fprintf(
		w,
		"install.auth.next_step: skills-hub auth status %s@%s --module plugins --runtime %s --target %q\n",
		entry.ID,
		version,
		runtimeName,
		pluginTargetRoot,
	)
}
