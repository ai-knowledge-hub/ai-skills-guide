package main

import (
	"fmt"
	"os"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	count, err := registry.ValidateRepository(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] semantic manifest validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Semantic manifest validation completed for %d manifests.\n", count)
}
