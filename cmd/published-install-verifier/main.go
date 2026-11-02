// published-install-verifier is a read-only companion to the public-delivery
// checker. It never installs, authenticates, executes package code or writes
// readiness evidence.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("published-install-verifier", flag.ContinueOnError)
	path := flags.String("manifest", "", "digest-bound pinned skill.yaml")
	digest := flags.String("manifest-sha256", "", "expected manifest SHA-256")
	id := flags.String("id", "", "expected pinned package ID")
	version := flags.String("version", "", "expected pinned version")
	packageDir := flags.String("package-dir", "", "optional installed package to verify")
	runtime := flags.String("runtime", "", "selected adapter")
	artifact := flags.String("artifact-sha256", "", "expected published archive SHA-256")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *path == "" || *id == "" || *version == "" || !validDigest(*digest) {
		return errors.New("manifest, manifest-sha256, id and version are required")
	}
	payload, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(payload)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), *digest) {
		return errors.New("pinned manifest digest mismatch")
	}
	manifest, err := registry.ValidateHistoricalPackageManifest(*path)
	if err != nil {
		return err
	}
	if manifest.ID != *id || manifest.Version != *version {
		return errors.New("manifest does not match the pinned subject")
	}
	if manifest.Usability.Execution != "instructions" || manifest.Usability.Availability != "documentation-only" || len(manifest.Usability.ExecutableHelpers) != 0 {
		return errors.New("pinned sample must be instruction-only without executable helpers")
	}
	contract, err := installer.RuntimeContractSHA256(registry.ProjectManifest(manifest), manifest.Version)
	if err != nil {
		return err
	}
	result := struct {
		ID                    string   `json:"id"`
		Version               string   `json:"version"`
		Runtimes              []string `json:"runtimes"`
		RuntimeContractSHA256 string   `json:"runtime_contract_sha256"`
		ReceiptVerified       bool     `json:"receipt_verified,omitempty"`
	}{ID: manifest.ID, Version: manifest.Version, Runtimes: manifest.Runtimes, RuntimeContractSHA256: contract}
	if *packageDir != "" {
		if !validDigest(*artifact) || !declaresRuntime(manifest.Runtimes, *runtime) {
			return errors.New("installed verification requires an artifact digest and declared runtime")
		}
		receipt, err := installer.VerifyInstallReceipt(*packageDir, "skills", *id, *version, *runtime, *artifact, contract)
		if err != nil {
			return err
		}
		if receipt.Source != "remote" {
			return errors.New("installed receipt source must be remote")
		}
		result.ReceiptVerified = true
	}
	return json.NewEncoder(output).Encode(result)
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func declaresRuntime(values []string, selected string) bool {
	for _, value := range values {
		if value == selected {
			return true
		}
	}
	return false
}
