package main

import (
	_ "embed"
	"fmt"
	"os"
)

// The guide belongs to the CLI binary, not the receipt-verified package tree.
//
//go:embed setup-guide.md
var installerSetupGuide string

func runHelp(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	if len(args) == 1 && args[0] == "setup" {
		_, err := fmt.Fprint(os.Stdout, installerSetupGuide)
		return err
	}
	return fmt.Errorf("unknown help topic; use skills-hub help or skills-hub help setup")
}
