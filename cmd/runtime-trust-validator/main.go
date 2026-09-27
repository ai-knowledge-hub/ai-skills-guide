package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/agents"
)

func main() {
	keys := map[string]*string{
		"CODEX_MODEL_ATTESTATION_PUBLIC_KEY":   flag.String("codex", "", "Codex runtime model-attestation public key"),
		"CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY":  flag.String("claude", "", "Claude runtime model-attestation public key"),
		"GENERIC_MODEL_ATTESTATION_PUBLIC_KEY": flag.String("generic", "", "generic runtime model-attestation public key"),
	}
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "runtime trust validator does not accept positional arguments")
		os.Exit(2)
	}
	for name, value := range keys {
		if *value == "" {
			fmt.Fprintf(os.Stderr, "missing governed %s\n", name)
			os.Exit(1)
		}
		if _, err := agents.ValidateModelAttestationPublicKey(*value); err != nil {
			fmt.Fprintf(os.Stderr, "invalid governed %s: %v\n", name, err)
			os.Exit(1)
		}
	}
}
