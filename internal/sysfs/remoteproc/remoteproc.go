//go:build !fake_sysfs

package remoteproc

import (
	"fmt"
	"os"
	"path/filepath"
)

const classPath = "/sys/class/remoteproc"

func FindMCUDirectory(mcu string) (string, error) {
	files, err := os.ReadDir(classPath)
	if err != nil {
		return "", fmt.Errorf("failed to read remoteproc directory %s: %w", classPath, err)
	}

	availableMCUs := []string{}
	for _, file := range files {
		instancePath := filepath.Join(classPath, file.Name())
		instanceName, err := readInstanceName(instancePath)
		if err != nil {
			continue
		}
		if instanceName == mcu {
			return instancePath, nil
		}
		availableMCUs = append(availableMCUs, instanceName)
	}

	return "", fmt.Errorf("%s is not in the list of available mcus %v", mcu, availableMCUs)
}

func readInstanceName(instancePath string) (string, error) {
	instanceNamePath := filepath.Join(instancePath, "name")
	nameFileContents, err := os.ReadFile(instanceNamePath)
	if err != nil {
		return "", err
	}
	return string(nameFileContents), nil
}
