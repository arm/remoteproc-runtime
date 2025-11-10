package remoteproc

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arm/remoteproc-runtime/internal/rootpath"
)

const (
	rprocStateFileName        = "state"
	rprocInstanceNameFileName = "name"
	rprocFirmwareFileName     = "firmware"
)

var (
	rprocClassPath      = rootpath.Join("sys", "class", "remoteproc")
	firmwareParamPath   = rootpath.Join("sys", "module", "firmware_class", "parameters", "path")
	defaultFirmwarePath = rootpath.Join("lib", "firmware")
)

func GetCustomFirmwarePath(customPathFile string) (string, error) {
	customPath, err := os.ReadFile(customPathFile)
	if err == nil {
		path := strings.TrimSpace(string(customPath))
		if path == "" {
			return "", fmt.Errorf("custom firmware path is empty in %s", customPathFile)
		}
		return path, nil
	}
	return "", fmt.Errorf("failed to read custom firmware path %s: %w", customPathFile, err)
}

func GetSystemFirmwarePath() (string, error) {
	customPath, _ := GetCustomFirmwarePath(firmwareParamPath)
	if customPath == "" {
		return defaultFirmwarePath, fmt.Errorf("current user is root")
	}

	return customPath, nil
}

func FindDevicePath(name string) (string, error) {
	files, err := os.ReadDir(rprocClassPath)
	if err != nil {
		return "", fmt.Errorf("failed to read remoteproc directory %s: %w", rprocClassPath, err)
	}

	availableNames := []string{}
	for _, file := range files {
		instancePath := filepath.Join(rprocClassPath, file.Name())
		instanceName, err := readFile(filepath.Join(instancePath, rprocInstanceNameFileName))
		if err != nil {
			continue
		}
		if instanceName == name {
			return instancePath, nil
		}
		availableNames = append(availableNames, instanceName)
	}

	return "", fmt.Errorf("remote processor %s does not exist, available remote processors: %s", name, strings.Join(availableNames, ", "))
}

type State string

const (
	StateOffline   State = "offline"
	StateSuspended State = "suspended"
	StateRunning   State = "running"
	StateCrashed   State = "crashed"
	StateInvalid   State = "invalid"
)

func newState(value string) (State, error) {
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

func GetState(devicePath string) (State, error) {
	stateFilePath := buildStateFilePath(devicePath)
	rawState, err := readFile(stateFilePath)
	if err != nil {
		return "", err
	}
	state, err := newState(string(rawState))
	if err != nil {
		return "", fmt.Errorf("can't parse state from %s: %w", stateFilePath, err)
	}
	return state, nil
}

// StoreFirmware copies a firmware file to kernel's firmware directory from sourcePath with a unique suffix
// to prevent overwriting existing files. Returns the stored firmware file path.
func StoreFirmware(sourcePath string, destDir string) (string, error) {
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
	destPath := filepath.Join(destDir, targetFileName)
	err = os.MkdirAll(filepath.Dir(destPath), 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create firmware storage directory %s: %w", filepath.Dir(destPath), err)
	}
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write firmware file %s: %w", destPath, err)
	}
	return destPath, nil
}

func SetFirmware(devicePath string, firmwareFilePath string) error {
	state, err := GetState(devicePath)
	if err != nil {
		return fmt.Errorf("pre-flight state check failed: %w", err)
	}
	if state == StateRunning {
		return fmt.Errorf("remote processor is already running")
	}

	firmwareFileName := filepath.Base(firmwareFilePath)
	if err := os.WriteFile(buildFirmwareFilePath(devicePath), []byte(firmwareFileName), 0o644); err != nil {
		return fmt.Errorf("failed to set firmware %s: %w", firmwareFileName, err)
	}
	return nil
}

func Start(devicePath string) error {
	state, err := GetState(devicePath)
	if err != nil {
		return fmt.Errorf("pre-flight state check failed: %w", err)
	}
	if state == StateRunning {
		return fmt.Errorf("remote processor is already running")
	}
	if err := os.WriteFile(buildStateFilePath(devicePath), []byte("start"), 0o644); err != nil {
		return fmt.Errorf("failed to start remote processor: %w", err)
	}
	return nil
}

func Stop(devicePath string) error {
	if err := os.WriteFile(buildStateFilePath(devicePath), []byte("stop"), 0o644); err != nil {
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

func generateUniqueSuffix() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return fmt.Sprintf("_%s_%x", timestamp, randomBytes), nil
}
