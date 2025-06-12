package remoteproc

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const rprocFirmwareStorePath = "/lib/firmware"

type State string

const (
	StateOffline   State = "offline"
	StateSuspended State = "suspended"
	StateRunning   State = "running"
	StateCrashed   State = "crashed"
	StateInvalid   State = "invalid"
)

func NewState(value string) (State, error) {
	switch State(value) {
	case StateOffline:
		return StateOffline, nil
	case StateSuspended:
		return StateSuspended, nil
	case StateRunning:
		return StateRunning, nil
	case StateCrashed:
		return StateCrashed, nil
	case StateInvalid:
		return StateInvalid, nil
	default:
		return "", fmt.Errorf("unknown state %s", value)
	}
}

// StoreFirmware copies a firmware file to /lib/firmrware with a unique suffix
// to prevent overwriting existing files. Returns the new file name.
func StoreFirmware(sourcePath string) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	fileName := filepath.Base(sourcePath)
	ext := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, ext)

	suffix, err := generateUniqueSuffix()
	if err != nil {
		return "", fmt.Errorf("can't generate random suffix: %w", err)
	}

	targetFileName := fmt.Sprintf("%s%s%s", nameWithoutExt, suffix, ext)

	destPath := filepath.Join(rprocFirmwareStorePath, targetFileName)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write firmware file %s: %w", destPath, err)
	}
	return targetFileName, nil
}

func RemoveFirmware(firmwareFileName string) error {
	return os.Remove(filepath.Join(rprocFirmwareStorePath, firmwareFileName))
}

func generateUniqueSuffix() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return fmt.Sprintf("_%s_%x", timestamp, randomBytes), nil
}
