// An offline diagnostic for explicitly supplied compatibility data. Its output
// is never consumed as runtime admission or authoritative published evidence.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/compatibility"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type request struct {
	InvalidatedAt time.Time                           `json:"invalidated_at,omitempty"`
	Subject       compatibility.Subject               `json:"subject"`
	Requirements  []registry.CompatibilityRequirement `json:"requirements"`
}

func read(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if len(data) > 1024*1024 {
		return nil, fmt.Errorf("input exceeds 1 MiB")
	}
	return data, err
}
func decode(path string, target any) error {
	data, err := read(path)
	if err != nil {
		return err
	}
	// The fixed-shape request rejects fields that could masquerade as readiness.
	return strictDecode(data, target)
}
func main() {
	if err := run(os.Args[1:], os.Stdout, time.Now().UTC()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
func run(args []string, stdout io.Writer, now time.Time) error {
	fs := flag.NewFlagSet("compatibility-evaluator", flag.ContinueOnError)
	reqPath := fs.String("request", "", "JSON subject and current requirements from selected target")
	trustPath := fs.String("trust-store", "", "operator-owned JSON mapping approved raw evidence SHA256 to producer (separate from records)")
	var paths []string
	fs.Func("evidence", "JSON record path; repeat for multiple observations", func(value string) error { paths = append(paths, value); return nil })
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *reqPath == "" || fs.NArg() != 0 {
		return fmt.Errorf("--request is required; see docs/compatibility-evaluation.md")
	}
	var req request
	if err := decode(*reqPath, &req); err != nil {
		return err
	}
	trusted := map[string]string{}
	if *trustPath != "" {
		if err := decode(*trustPath, &trusted); err != nil {
			return err
		}
	}
	digestPattern := regexp.MustCompile(`^[a-f0-9]{64}$`)
	for sha, producer := range trusted {
		if !digestPattern.MatchString(sha) || strings.TrimSpace(producer) == "" {
			return fmt.Errorf("invalid trust-store digest or producer")
		}
	}
	var payloads [][]byte
	for _, path := range paths {
		data, err := read(path)
		if err != nil {
			return err
		}
		payloads = append(payloads, data)
	}
	assessment, err := compatibility.Evaluate(compatibility.Context{Subject: req.Subject, Requirements: req.Requirements, TrustedEvidence: trusted, Now: now, InvalidatedAt: req.InvalidatedAt}, payloads)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(struct {
		Compatibility        compatibility.Assessment `json:"compatibility"`
		OperationalReadiness string                   `json:"operational_readiness"`
	}{assessment, "not_evaluated"})
}
