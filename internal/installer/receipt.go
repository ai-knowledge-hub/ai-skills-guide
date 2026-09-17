package installer

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

const installReceiptName = ".skills-hub-install.json"

type InstallReceipt struct {
	SchemaVersion         string                 `json:"schema_version"`
	Source                string                 `json:"source"`
	Module                string                 `json:"module"`
	ID                    string                 `json:"id"`
	Version               string                 `json:"version"`
	Runtime               string                 `json:"runtime"`
	ArtifactSHA256        string                 `json:"artifact_sha256,omitempty"`
	RuntimeContractSHA256 string                 `json:"runtime_contract_sha256"`
	TreeSHA256            string                 `json:"tree_sha256"`
	ParentArtifactSHA256  string                 `json:"parent_artifact_sha256,omitempty"`
	Closure               []InstallReceiptMember `json:"closure,omitempty"`
	InstalledAt           time.Time              `json:"installed_at"`
}

type InstallReceiptMember struct {
	Module                string `json:"module"`
	ID                    string `json:"id"`
	Version               string `json:"version"`
	RuntimeContractSHA256 string `json:"runtime_contract_sha256"`
	TreeSHA256            string `json:"tree_sha256"`
}

func WriteInstallReceipt(packageDir string, receipt InstallReceipt) error {
	if strings.TrimSpace(packageDir) == "" || receipt.ID == "" || receipt.Version == "" || receipt.Module == "" || receipt.Runtime == "" {
		return errors.New("install receipt identity is incomplete")
	}
	if err := validateReceiptDigests(receipt); err != nil {
		return err
	}
	digest, err := packageTreeSHA256(packageDir)
	if err != nil {
		return fmt.Errorf("compute installed package digest: %w", err)
	}
	receipt.SchemaVersion = "skills-hub.install-receipt/v1"
	receipt.TreeSHA256 = digest
	if receipt.InstalledAt.IsZero() {
		receipt.InstalledAt = time.Now().UTC()
	}
	payload, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(packageDir, installReceiptName)
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		return fmt.Errorf("write install receipt: %w", err)
	}
	return nil
}

func VerifyInstallReceipt(packageDir, module, id, version, runtimeName, artifactSHA256, runtimeContractSHA256 string) (InstallReceipt, error) {
	payload, err := os.ReadFile(filepath.Join(packageDir, installReceiptName))
	if err != nil {
		return InstallReceipt{}, fmt.Errorf("read install receipt: %w", err)
	}
	var receipt InstallReceipt
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return InstallReceipt{}, errors.New("install receipt is malformed")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return InstallReceipt{}, errors.New("install receipt contains trailing data")
	}
	if receipt.SchemaVersion != "skills-hub.install-receipt/v1" || receipt.Module != module || receipt.ID != id || receipt.Version != version || receipt.Runtime != runtimeName {
		return InstallReceipt{}, errors.New("install receipt does not match the selected package")
	}
	if err := validateReceiptDigests(receipt); err != nil {
		return InstallReceipt{}, err
	}
	if len(receipt.TreeSHA256) != sha256.Size*2 || receipt.InstalledAt.IsZero() {
		return InstallReceipt{}, errors.New("install receipt integrity metadata is invalid")
	}
	for _, member := range receipt.Closure {
		if member.Module == "" || member.ID == "" || member.Version == "" || !validSHA256(member.RuntimeContractSHA256) || !validSHA256(member.TreeSHA256) {
			return InstallReceipt{}, errors.New("install receipt closure metadata is invalid")
		}
	}
	if _, err := hex.DecodeString(receipt.TreeSHA256); err != nil {
		return InstallReceipt{}, errors.New("install receipt integrity metadata is invalid")
	}
	if artifactSHA256 != "" && !strings.EqualFold(receipt.ArtifactSHA256, artifactSHA256) {
		return InstallReceipt{}, errors.New("install receipt does not match the selected artifact")
	}
	if runtimeContractSHA256 != "" && !strings.EqualFold(receipt.RuntimeContractSHA256, runtimeContractSHA256) {
		return InstallReceipt{}, errors.New("install receipt does not match the selected runtime contract")
	}
	digest, err := packageTreeSHA256(packageDir)
	if err != nil {
		return InstallReceipt{}, fmt.Errorf("verify installed package digest: %w", err)
	}
	if !strings.EqualFold(digest, receipt.TreeSHA256) {
		return InstallReceipt{}, errors.New("installed package contents have changed since installation")
	}
	return receipt, nil
}

func validateReceiptDigests(receipt InstallReceipt) error {
	if receipt.Source != "local" && receipt.Source != "remote" && receipt.Source != "bundled" {
		return errors.New("install receipt source is invalid")
	}
	if len(receipt.RuntimeContractSHA256) != sha256.Size*2 {
		return errors.New("install receipt runtime contract digest is invalid")
	}
	if _, err := hex.DecodeString(receipt.RuntimeContractSHA256); err != nil {
		return errors.New("install receipt runtime contract digest is invalid")
	}
	if receipt.Source == "remote" {
		if len(receipt.ArtifactSHA256) != sha256.Size*2 {
			return errors.New("remote install receipt artifact digest is invalid")
		}
		if _, err := hex.DecodeString(receipt.ArtifactSHA256); err != nil {
			return errors.New("remote install receipt artifact digest is invalid")
		}
	}
	if receipt.Source == "bundled" && !validSHA256(receipt.ParentArtifactSHA256) {
		return errors.New("bundled install receipt parent artifact digest is invalid")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// VerifyInstallClosure verifies every separately activated dependency recorded
// by a plugin receipt. The plugin tree alone is not authoritative because
// runtimes discover these copies from sibling module directories.
func VerifyInstallClosure(packageDir, pluginTargetRoot string, receipt InstallReceipt, includes *registry.IncludeSet) error {
	if receipt.Module != "plugins" {
		return nil
	}
	expected := make(map[string]struct{})
	if includes != nil {
		for _, set := range []struct {
			module string
			ids    []string
		}{{"skills", includes.Skills}, {"agents", includes.Agents}, {"tools", includes.Tools}} {
			for _, id := range set.ids {
				expected[set.module+"\x00"+id] = struct{}{}
			}
		}
	}
	if len(receipt.Closure) != len(expected) {
		return errors.New("install receipt does not describe the complete plugin closure")
	}
	seen := make(map[string]struct{}, len(receipt.Closure))
	for _, member := range receipt.Closure {
		key := member.Module + "\x00" + member.ID
		if _, ok := expected[key]; !ok {
			return errors.New("install receipt contains an unexpected plugin dependency")
		}
		if _, duplicate := seen[key]; duplicate {
			return errors.New("install receipt contains a duplicate plugin dependency")
		}
		seen[key] = struct{}{}
		target, err := ResolvePluginDependencyTarget(receipt.Runtime, member.Module, pluginTargetRoot)
		if err != nil {
			return err
		}
		memberDir := filepath.Join(target.TargetPath, filepath.FromSlash(member.ID))
		installed, err := VerifyInstallReceipt(memberDir, member.Module, member.ID, member.Version, receipt.Runtime, "", member.RuntimeContractSHA256)
		if err != nil {
			return fmt.Errorf("verify installed plugin dependency %s %s: %w", member.Module, member.ID, err)
		}
		sourceBound := installed.Source == "local"
		if receipt.Source == "remote" {
			sourceBound = installed.Source == "bundled" && strings.EqualFold(installed.ParentArtifactSHA256, receipt.ArtifactSHA256)
		}
		if !sourceBound || !strings.EqualFold(installed.TreeSHA256, member.TreeSHA256) {
			return fmt.Errorf("installed plugin dependency %s %s is not bound to the plugin release", member.Module, member.ID)
		}
	}
	return nil
}

func RuntimeContractSHA256(entry registry.SkillEntry, version string) (string, error) {
	contract := struct {
		ID             string                           `json:"id"`
		Version        string                           `json:"version"`
		Runtimes       []string                         `json:"runtimes"`
		Execution      *registry.ExecutionMetadata      `json:"execution"`
		Artifact       *registry.ArtifactMetadata       `json:"artifact"`
		Authentication *registry.AuthenticationMetadata `json:"authentication"`
	}{
		ID: entry.ID, Version: version, Runtimes: entry.Runtimes, Execution: entry.Execution,
		Artifact: entry.Artifact, Authentication: entry.Authentication,
	}
	payload, err := json.Marshal(contract)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func packageTreeSHA256(root string) (string, error) {
	type item struct {
		path string
		mode fs.FileMode
	}
	var items []item
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." || filepath.ToSlash(relative) == installReceiptName {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !entry.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported installed package entry %s", filepath.ToSlash(relative))
		}
		items = append(items, item{path: relative, mode: info.Mode()})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(items, func(i, j int) bool { return filepath.ToSlash(items[i].path) < filepath.ToSlash(items[j].path) })
	hash := sha256.New()
	writer := bufio.NewWriter(hash)
	for _, entry := range items {
		relative := filepath.ToSlash(entry.path)
		kind := "file"
		canonicalMode := os.FileMode(0o644)
		if entry.mode.IsDir() {
			kind = "dir"
			canonicalMode = 0o755
		} else if entry.mode&0o111 != 0 {
			canonicalMode = 0o755
		}
		_, _ = fmt.Fprintf(writer, "%s\x00%s\x00%o\x00", kind, relative, canonicalMode)
		if kind == "file" {
			file, err := os.Open(filepath.Join(root, entry.path))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(writer, file); err != nil {
				file.Close()
				return "", err
			}
			if err := file.Close(); err != nil {
				return "", err
			}
		}
		_, _ = writer.WriteString("\x00")
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
