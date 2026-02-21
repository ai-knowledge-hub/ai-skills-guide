package installer

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
