package userdirs

import (
	"os"
	"os/user"
	"path/filepath"
)

func joinHomeDir(elem ...string) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	chunks := append([]string{user.HomeDir}, elem...)
	return filepath.Join(chunks...), nil
}

func RuntimeDir() (string, error) {
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir != "" {
		return filepath.Join(xdgRuntimeDir, "remoteproc-runtime"), nil
	} else {
		return joinHomeDir(".local", "run", "remoteproc-runtime")
	}
}
