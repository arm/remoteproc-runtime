package shared

import (
	"fmt"
	"os"
	"path/filepath"
)

func findRepoRootDir() (string, error) {
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

func mustRepoRootJoin(segments ...string) string {
	repoRoot, err := findRepoRootDir()
	if err != nil {
		panic("can't determine repo root")
	}
	allSegments := append([]string{repoRoot}, segments...)
	return filepath.Join(allSegments...)
}
