package remoteproc

import (
	"fmt"
	"os"
	"path/filepath"
)

const remoteprocDir = "/sys/class/remoteproc"

func CheckMCUExists(mcu string) error {
	const remoteprocDir = "/sys/class/remoteproc"
	files, err := os.ReadDir(remoteprocDir)
	if err != nil {
		return fmt.Errorf("failed to read remoteproc directory %s: %w", remoteprocDir, err)
	}

	for _, file := range files {
		if file.Type().IsDir() {
			nameFilePath := filepath.Join(remoteprocDir, file.Name(), "name")
			if nameFileContents, err := os.ReadFile(nameFilePath); err == nil {
				if string(nameFileContents) == mcu {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("%s mcu not found in %s/*/name", mcu, remoteprocDir)
}
