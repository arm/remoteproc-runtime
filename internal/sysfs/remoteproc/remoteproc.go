//go:build !fake_sysfs

package remoteproc

import (
	"fmt"
	"os"
	"path/filepath"
)

const remoteprocDir = "/sys/class/remoteproc"

func ListMCUs() ([]string, error) {
	files, err := os.ReadDir(remoteprocDir)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read remoteproc directory %s: %w", remoteprocDir, err)
	}

	mcus := []string{}
	for _, file := range files {
		if file.Type().IsDir() {
			nameFilePath := filepath.Join(remoteprocDir, file.Name(), "name")
			if nameFileContents, err := os.ReadFile(nameFilePath); err == nil {
				mcus = append(mcus, string(nameFileContents))
			}
		}
	}

	return mcus, nil
}
