package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func main() {
	fs := flag.NewFlagSet("registry-builder", flag.ExitOnError)
	root := fs.String("root", ".", "repository root")
	module := fs.String("module", "all", "module to build: skills|agents|tools|all")
	out := fs.String("out", "", "output path relative to root (single-module mode only)")
	outDir := fs.String("out-dir", "registry", "output directory relative to root")
	_ = fs.Parse(os.Args[1:])

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fatal(err)
	}

	switch *module {
	case "skills":
		writeSingle(absRoot, *outDir, defaultOut(*out, "skills-index.json"), registry.BuildSkillsIndex)
	case "agents":
		writeSingle(absRoot, *outDir, defaultOut(*out, "agents-index.json"), registry.BuildAgentsIndex)
	case "tools":
		writeSingle(absRoot, *outDir, defaultOut(*out, "tools-index.json"), registry.BuildToolsIndex)
	case "all":
		writeAll(absRoot, *outDir)
	default:
		fatal(fmt.Errorf("invalid --module %q (use skills|agents|tools|all)", *module))
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

type buildFn func(root string) (registry.Index, error)

func writeSingle(absRoot, outDir, outName string, fn buildFn) {
	index, err := fn(absRoot)
	if err != nil {
		fatal(err)
	}
	outputPath := resolveOutputPath(absRoot, outDir, outName)
	if err := registry.WriteIndex(outputPath, index); err != nil {
		fatal(err)
	}
	rel, err := filepath.Rel(absRoot, outputPath)
	if err != nil {
		rel = outputPath
	}
	fmt.Printf("Wrote %s with %d item(s).\n", filepath.ToSlash(rel), len(index.Skills))
}

func writeAll(absRoot, outDir string) {
	writeSingle(absRoot, outDir, "skills-index.json", registry.BuildSkillsIndex)
	writeSingle(absRoot, outDir, "agents-index.json", registry.BuildAgentsIndex)
	writeSingle(absRoot, outDir, "tools-index.json", registry.BuildToolsIndex)
	// Backward compatibility: keep legacy skills index path.
	writeSingle(absRoot, outDir, "index.json", registry.BuildSkillsIndex)
}

func defaultOut(out, fallback string) string {
	if out != "" {
		return filepath.ToSlash(out)
	}
	return fallback
}

func resolveOutputPath(absRoot, outDir, outName string) string {
	clean := filepath.ToSlash(outName)
	if filepath.IsAbs(clean) || clean == filepath.Base(clean) {
		if filepath.IsAbs(clean) {
			return clean
		}
		return filepath.Join(absRoot, filepath.FromSlash(outDir), clean)
	}
	return filepath.Join(absRoot, filepath.FromSlash(clean))
}
