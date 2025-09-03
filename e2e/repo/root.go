package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

func findRootDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	dir := currentDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root (go.mod not found)")
		}
		dir = parent
	}
}

func MustFindRootDir() string {
	repoRoot, err := findRootDir()
	if err != nil {
		panic(err)
	}
	return repoRoot
}
