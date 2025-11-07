package userdirs

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

func getHomeDirFromUserDB() (string, error) {
	// Gets home directory from system user database instead of environment variables.
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.HomeDir, nil
}

func RuntimeDir() (string, error) {
	/*
		When running with podman, environment variables are sanitized.
		The only environement variable guaranteed after podman sanitization is $XDG_RUNTIME_DIR and $PATH.
		$XDG_RUNTIME_DIR will be set to an empty string in case of podman root run,
		resolving the directory to an area in home directory.
	*/
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir != "" {
		return filepath.Join(xdgRuntimeDir, "remoteproc-runtime"), nil
	} else {
		userHomeDir, err := getUserHomeDirFromSystem()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		return filepath.Join(userHomeDir, ".local", "run", "remoteproc-runtime"), nil
	}
}
