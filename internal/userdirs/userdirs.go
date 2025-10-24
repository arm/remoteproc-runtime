package userdirs

import (
	"fmt"
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
		return filepath.Join(xdgRuntimeDir, ".remoteproc-runtime"), nil
	} else {
		fmt.Println("XDG_RUNTIME_DIR is not set, falling back to home directory")
	}
	return joinHomeDir(".remoteproc-runtime") // I'M NOT SURE ABOUT THIS BEING SENSIBLE DEFAULT
}
