package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/arm/remoteproc-runtime/internal/userdirs"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	stateFileName = "state.json"
)

var (
	stateDirOnce      sync.Once
	cachedStateDir    string
	cachedStateDirErr error
)

func getStateDir() (string, error) {
	stateDirOnce.Do(func() {
		cachedStateDir, cachedStateDirErr = userdirs.RuntimeDir()
		if cachedStateDirErr != nil {
			cachedStateDirErr = fmt.Errorf("failed to get runtime directory: %w", cachedStateDirErr)
			return
		}
	})
	return cachedStateDir, cachedStateDirErr
}

func NewState(containerID string, bundlePath string) *specs.State {
	return &specs.State{
		Version:     specs.Version,
		ID:          containerID,
		Status:      specs.StateCreated,
		Pid:         0,
		Bundle:      bundlePath,
		Annotations: map[string]string{},
	}
}

func WriteState(state *specs.State) error {
	stateDir, err := getStateDir()
	if err != nil {
		return err
	}
	containerStateDir := filepath.Join(stateDir, state.ID)
	if err := os.MkdirAll(containerStateDir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	stateFilePath := filepath.Join(containerStateDir, stateFileName)
	if err := atomicWrite(stateFilePath, stateJSON); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

func atomicWrite(filePath string, content []byte) error {
	tmpFilePath := filePath + ".tmp"
	if err := os.WriteFile(tmpFilePath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary file %s: %w", tmpFilePath, err)
	}
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		_ = os.Remove(tmpFilePath)
		return fmt.Errorf("failed to rename temp file %s to %s", tmpFilePath, filePath)
	}
	return nil
}

func ReadState(containerID string) (*specs.State, error) {
	stateDir, err := getStateDir()
	if err != nil {
		return nil, err
	}
	stateFilePath := filepath.Join(stateDir, containerID, stateFileName)
	f, err := os.Open(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", stateFilePath, err)
	}
	defer func() { _ = f.Close() }()
	var s specs.State
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	if err := validateStateAnnotations(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func RemoveState(containerID string) error {
	stateDir, err := getStateDir()
	if err != nil {
		return err
	}
	containerStateDir := filepath.Join(stateDir, containerID)
	if err := os.RemoveAll(containerStateDir); err != nil {
		return fmt.Errorf("cannot remove container state dir: %w", err)
	}
	return nil
}
