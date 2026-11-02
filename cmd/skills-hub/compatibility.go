package main

import (
	"fmt"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
	"io"
	"strings"
)

func printCompatibilitySummary(w io.Writer, entry registry.SkillEntry, version string) {
	if version != entry.Latest {
		fmt.Fprintln(w, "compatibility: unknown (declarations describe latest version only)")
		return
	}
	if len(entry.Compatibility) == 0 {
		fmt.Fprintln(w, "compatibility: unknown (not declared)")
		return
	}
	for _, requirement := range entry.Compatibility {
		fmt.Fprintf(w, "declared_compatibility: %s %s; exact_versions=%s; capabilities=%s\n", requirement.Kind, requirement.Name, strings.Join(requirement.Versions, ","), strings.Join(requirement.Capabilities, ","))
	}
}
