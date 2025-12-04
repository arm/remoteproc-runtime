package e2e

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
)

var testCounter atomic.Uint32

func getTestNumber() uint {
	return uint(testCounter.Add(1))
}

// ensureDir ensures the provided directory path exists, creating missing folders as needed.
func ensureDir(path string) (string, error) {
	cleaned := filepath.Clean(path)

	if cleaned == "." || cleaned == string(os.PathSeparator) || cleaned == "" {
		return "", nil
	}

	var firstMissing string
	dirToCheck := cleaned

	for {
		info, err := os.Stat(dirToCheck)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s exists but is not a directory", dirToCheck)
			}
			break
		}

		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}

		firstMissing = dirToCheck

		parent := filepath.Dir(dirToCheck)
		if parent == dirToCheck {
			break
		}
		dirToCheck = parent
	}

	if firstMissing == "" {
		return "", nil
	}

	if err := os.MkdirAll(cleaned, 0o755); err != nil {
		return "", err
	}

	return firstMissing, nil
}
