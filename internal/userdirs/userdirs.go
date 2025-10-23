package userdirs

import (
	"os"
	"path/filepath"
)

func joinHomeDir(elem ...string) (string, error) {
	localDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	chunks := append([]string{localDir}, elem...)
	return filepath.Join(chunks...), nil
}

func RuntimeDir() (string, error) {
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir != "" {
		return filepath.Join(xdgRuntimeDir, ".remoteproc-runtime"), nil
	}
	return joinHomeDir(".remoteproc-runtime") // I'M NOT SURE ABOUT THIS BEING SENSIBLE DEFAULT
}
