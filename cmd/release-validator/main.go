package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/ai-knowledge-hub/ai-skills-guide/internal/installer"
	"github.com/ai-knowledge-hub/ai-skills-guide/internal/registry"
)

type moduleDefinition struct {
	directory string
	manifest  string
}

var modules = []moduleDefinition{
	{directory: "skills", manifest: "skill.yaml"},
	{directory: "agents", manifest: "agent.yaml"},
	{directory: "tools-mcp", manifest: "tool.yaml"},
	{directory: "plugins", manifest: "plugin.yaml"},
}

func main() {
	root := flag.String("root", "releases", "immutable release store")
	jsonOutput := flag.Bool("json", false, "write current catalog projections as JSON")
	flag.Parse()
	result, err := validateReleaseStore(*root, time.Now().UTC())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	fmt.Printf("Validated %d retained release(s).\n", result.Count)
}

type validationResult struct {
	Count          int                            `json:"count"`
	CurrentEntries map[string]registry.SkillEntry `json:"current_entries"`
}

func validateReleaseStore(root string, now time.Time) (validationResult, error) {
	result := validationResult{CurrentEntries: make(map[string]registry.SkillEntry)}
	for _, module := range modules {
		moduleRoot := filepath.Join(root, module.directory)
		if _, err := os.Stat(moduleRoot); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return validationResult{}, err
		}
		err := filepath.WalkDir(moduleRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || entry.Name() != module.manifest {
				return nil
			}
			relative, err := filepath.Rel(moduleRoot, path)
			if err != nil {
				return err
			}
			parts := strings.Split(filepath.ToSlash(relative), "/")
			if len(parts) != 4 {
				return fmt.Errorf("invalid release manifest path %s", path)
			}
			id := parts[0] + "/" + parts[1]
			version := parts[2]
			releaseDir := filepath.Dir(path)
			manifest, err := installer.ValidateRetainedRelease(
				filepath.Join(releaseDir, "package.tar.gz"), path,
				module.directory, id, version,
			)
			if err != nil {
				return fmt.Errorf("%s: %w", releaseDir, err)
			}
			projectionPath := filepath.Join(releaseDir, "registry-entry.json")
			projectionInfo, err := os.Lstat(projectionPath)
			if err != nil || !projectionInfo.Mode().IsRegular() {
				return fmt.Errorf("release projection must be a regular file: %s", projectionPath)
			}
			projectionFile, err := os.Open(projectionPath)
			if err != nil {
				return err
			}
			decoder := json.NewDecoder(projectionFile)
			decoder.DisallowUnknownFields()
			var stored registry.SkillEntry
			decodeErr := decoder.Decode(&stored)
			if decodeErr == nil {
				var trailing any
				if err := decoder.Decode(&trailing); err != io.EOF {
					decodeErr = fmt.Errorf("unexpected trailing JSON content")
				}
			}
			closeErr := projectionFile.Close()
			if decodeErr != nil || closeErr != nil {
				return fmt.Errorf("decode release projection %s: %v", projectionPath, decodeErr)
			}
			if stored.Latest != "" || len(stored.Versions) != 0 {
				return fmt.Errorf("release projection %s must not contain release catalog fields", projectionPath)
			}
			expected := registry.ProjectManifest(manifest)
			if !reflect.DeepEqual(stored, expected) {
				return fmt.Errorf("release projection %s does not match its admitted manifest", projectionPath)
			}
			result.Count++
			result.CurrentEntries[module.directory+":"+id+"@"+version] = registry.ProjectManifestForCurrentCatalog(manifest, now)
			return nil
		})
		if err != nil {
			return validationResult{}, err
		}
	}
	return result, nil
}
