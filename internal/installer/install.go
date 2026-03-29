package installer

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func InstallSkill(sourceDir, targetRoot, skillID string, force bool) (string, error) {
	destinationDir := filepath.Join(targetRoot, filepath.FromSlash(skillID))
	if _, err := os.Stat(destinationDir); err == nil {
		if !force {
			return "", fmt.Errorf("destination already exists: %s (use --force to overwrite)", destinationDir)
		}
		if err := os.RemoveAll(destinationDir); err != nil {
			return "", fmt.Errorf("remove existing destination %s: %w", destinationDir, err)
		}
	}

	if err := copyTree(sourceDir, destinationDir); err != nil {
		return "", err
	}
	return destinationDir, nil
}

func PreparePluginRuntimeArtifacts(destinationDir, runtime string) ([]string, error) {
	normalizedRuntime := strings.ToLower(strings.TrimSpace(runtime))
	if normalizedRuntime != "codex" && normalizedRuntime != "claude" {
		return nil, nil
	}

	sourceManifest := filepath.Join(destinationDir, "plugin.json")
	payload, err := os.ReadFile(sourceManifest)
	if err != nil {
		return nil, fmt.Errorf("read plugin manifest %s: %w", sourceManifest, err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, fmt.Errorf("parse plugin manifest %s: %w", sourceManifest, err)
	}
	manifest["runtime"] = normalizedRuntime

	runtimeDir := filepath.Join(destinationDir, "."+normalizedRuntime+"-plugin")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, fmt.Errorf("create runtime manifest dir %s: %w", runtimeDir, err)
	}

	rendered, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render runtime manifest: %w", err)
	}
	runtimeManifestPath := filepath.Join(runtimeDir, "plugin.json")
	if err := os.WriteFile(runtimeManifestPath, append(rendered, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write runtime manifest %s: %w", runtimeManifestPath, err)
	}
	return []string{runtimeManifestPath}, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create destination directory %s: %w", filepath.Dir(dst), err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", dst, err)
	}
	return nil
}
