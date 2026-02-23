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
	out := fs.String("out", "registry/index.json", "output path relative to root")
	_ = fs.Parse(os.Args[1:])

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fatal(err)
	}

	index, err := registry.BuildIndex(absRoot)
	if err != nil {
		fatal(err)
	}

	outputPath := filepath.Join(absRoot, filepath.FromSlash(*out))
	if err := registry.WriteIndex(outputPath, index); err != nil {
		fatal(err)
	}

	rel, err := filepath.Rel(absRoot, outputPath)
	if err != nil {
		rel = outputPath
	}
	fmt.Printf("Wrote %s with %d skill(s).\n", filepath.ToSlash(rel), len(index.Skills))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
