//go:build !fake_sysfs

package remoteproc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	rprocClassPath            = "/sys/class/remoteproc"
	rprocStateFileName        = "state"
	rprocInstanceNameFileName = "name"
	rprocFirmwareFileName     = "firmware"
)

func FindDevicePath(mcu string) (string, error) {
	files, err := os.ReadDir(rprocClassPath)
	if err != nil {
		return "", fmt.Errorf("failed to read remoteproc directory %s: %w", rprocClassPath, err)
	}

	availableMCUs := []string{}
	for _, file := range files {
		instancePath := filepath.Join(rprocClassPath, file.Name())
		instanceName, err := readFile(filepath.Join(instancePath, rprocInstanceNameFileName))
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

func GetState(devicePath string) (State, error) {
	stateFilePath := buildStateFilePath(devicePath)
	rawState, err := readFile(stateFilePath)
	if err != nil {
		return "", err
	}
	state, err := NewState(string(rawState))
	if err != nil {
		return "", fmt.Errorf("can't parse state from %s: %w", stateFilePath, err)
	}
	return state, nil
}

func SetFirmwareAndStart(devicePath string, firmwareFileName string) error {
	state, err := GetState(devicePath)
	if err != nil {
		return fmt.Errorf("pre-flight state check failed: %w", err)
	}
	if state == StateRunning {
		return fmt.Errorf("remote processor is already running")
	}
	if err := os.WriteFile(buildFirmwareFilePath(devicePath), []byte(firmwareFileName), 0644); err != nil {
		return fmt.Errorf("failed to set firmware %s: %w", firmwareFileName, err)
	}
	if err := os.WriteFile(buildStateFilePath(devicePath), []byte("start"), 0644); err != nil {
		return fmt.Errorf("failed to start remote processor: %w", err)
	}
	return nil
}

func Stop(devicePath string) error {
	if err := os.WriteFile(buildStateFilePath(devicePath), []byte("stop"), 0644); err != nil {
		return fmt.Errorf("failed to stop remote processor: %w", err)
	}
	return nil
}

func buildStateFilePath(devicePath string) string {
	return filepath.Join(devicePath, rprocStateFileName)
}

func buildFirmwareFilePath(devicePath string) string {
	return filepath.Join(devicePath, rprocFirmwareFileName)
}

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return strings.TrimSpace(string(content)), nil
}
