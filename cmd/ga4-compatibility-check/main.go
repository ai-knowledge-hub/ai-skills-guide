// Runs the reviewed retained GA4 release against a host-owned local fixture.
// No live provider credentials or readiness promotion are involved.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/compatibility"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/connectorcheck"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("ga4-compatibility-check", flag.ContinueOnError)
	describe := fs.Bool("describe-profile", false, "show reviewed public scope fields; operator must supply public key independently")
	release := fs.String("release-dir", "releases/tools-mcp/analytics/ga4-mcp-connector/0.2.0", "reviewed retained GA4 release")
	node := fs.String("node", "node", "Node.js executable")
	keyPath := fs.String("signing-seed-file", "", "operator-owned base64 Ed25519 32-byte seed file; never passed to Node")
	keyID := fs.String("key-id", "", "independently configured verifier key ID")
	output := fs.String("out", "", "new output directory (must not exist)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *describe {
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected arguments")
		}
		return json.NewEncoder(stdout).Encode(compatibility.ProducerKey{Producer: connectorcheck.GA4Producer, TestID: connectorcheck.GA4Test, TestVersion: connectorcheck.ProbeVersion(), EntryID: "analytics/ga4-mcp-connector", ArtifactSHA256: connectorcheck.GA4ArtifactSHA256, Operation: connectorcheck.GA4Operation, ProviderTarget: connectorcheck.GA4Target})
	}
	if fs.NArg() != 0 || *keyPath == "" || *keyID == "" || *output == "" {
		return fmt.Errorf("--signing-seed-file, --key-id and --out are required")
	}
	if _, err := os.Lstat(*output); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("output must be a new directory")
	}
	keyFile, err := os.Open(*keyPath)
	if err != nil {
		return fmt.Errorf("cannot read producer signing seed")
	}
	encoded, err := io.ReadAll(io.LimitReader(keyFile, 4097))
	keyFile.Close()
	if err != nil || len(encoded) > 4096 {
		return fmt.Errorf("invalid producer signing seed file")
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return fmt.Errorf("invalid producer signing seed")
	}
	private := ed25519.NewKeyFromSeed(seed)
	// Capture the observation time after the test completes, not its fixture clock.
	request, payload, err := connectorcheck.RunGA4(context.Background(), *release, *node, time.Now().UTC())
	if err != nil {
		return err
	}
	var evidence compatibility.Evidence
	if err = json.Unmarshal(payload, &evidence); err != nil {
		return err
	}
	evidence.ObservedAt = time.Now().UTC()
	payload, err = json.Marshal(evidence)
	if err != nil {
		return err
	}
	envelope, err := compatibility.SignEvidence(payload, *keyID, private)
	if err != nil {
		return err
	}
	req, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return err
	}
	return publish(*output, map[string][]byte{"request.json": req, "evidence.json": payload, "signed-evidence.json": envelope})
}
func publish(root string, files map[string][]byte) (err error) {
	if err = os.Mkdir(root, 0o700); err != nil {
		return fmt.Errorf("output must be a new directory: %w", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(root)
		}
	}()
	for name, data := range files {
		if err = os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			return err
		}
	}
	return nil
}
