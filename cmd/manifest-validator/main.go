package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	targets := []struct {
		directory string
		manifest  string
	}{
		{directory: "skills", manifest: "skill.yaml"},
		{directory: "agents", manifest: "agent.yaml"},
		{directory: "tools-mcp", manifest: "tool.yaml"},
		{directory: "plugins", manifest: "plugin.yaml"},
	}

	count := 0
	for _, target := range targets {
		moduleRoot := filepath.Join(root, target.directory)
		err := filepath.WalkDir(moduleRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || entry.Name() != target.manifest {
				return nil
			}
			if _, err := registry.ParseManifest(path); err != nil {
				return err
			}
			count++
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] semantic manifest validation failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Semantic manifest validation completed for %d manifests.\n", count)
}
