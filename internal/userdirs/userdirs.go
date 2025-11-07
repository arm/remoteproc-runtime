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
	// Podman sanitizes environment variables, leaving only $XDG_RUNTIME_DIR and $PATH.
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir != "" {
		return filepath.Join(xdgRuntimeDir, "remoteproc-runtime"), nil
	} else {
		// When podman runs as root, $XDG_RUNTIME_DIR and $HOME are both unset
		// Therefore, HomeDir must be read from UserDB
		userHomeDir, err := getHomeDirFromUserDB()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		return filepath.Join(userHomeDir, ".local", "run", "remoteproc-runtime"), nil
	}
}
