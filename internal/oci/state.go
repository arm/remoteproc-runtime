package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	// TODO: is this safe, or should we have `/run/remoteproc-containers`?
	stateDir      = "/run/remoteproc"
	stateFileName = "state.json"
)

func NewState(containerID string, pid int, bundlePath string) *specs.State {
	return &specs.State{
		Version:     "1.2.0", // TODO: validate if this is the case
		ID:          containerID,
		Status:      specs.StateCreated,
		Pid:         1, // TODO: gpt tells me it's ok, is it?
		Bundle:      bundlePath,
		Annotations: map[string]string{},
	}
}

type RemoteprocAnnotations struct {
	MCU          string
	DevicePath   string
	FirmwareName string
}

const mcuRequestedKey = "remoteproc.mcu-requested"
const mcuResolvedPathKey = "remoteproc.mcu-resolved-path"
const firmwareNameKey = "remoteproc.firmware-name"

func (a RemoteprocAnnotations) Apply(state *specs.State) {
	if state.Annotations == nil {
		state.Annotations = map[string]string{}
	}
	state.Annotations[mcuRequestedKey] = a.MCU
	state.Annotations[mcuResolvedPathKey] = a.DevicePath
	state.Annotations[firmwareNameKey] = a.FirmwareName
}

func NewRemoteprocAnnotations(state *specs.State) (RemoteprocAnnotations, error) {
	mcuRequested, err := readAnnotation(state, mcuRequestedKey)
	if err != nil {
		return RemoteprocAnnotations{}, err
	}
	mcuResolvedPath, err := readAnnotation(state, mcuResolvedPathKey)
	if err != nil {
		return RemoteprocAnnotations{}, err
	}
	firmwareName, err := readAnnotation(state, firmwareNameKey)
	if err != nil {
		return RemoteprocAnnotations{}, err
	}

	return RemoteprocAnnotations{
		MCU:          mcuRequested,
		DevicePath:   mcuResolvedPath,
		FirmwareName: firmwareName,
	}, nil
}

func readAnnotation(state *specs.State, key string) (string, error) {
	if state.Annotations != nil {
		value, ok := state.Annotations[key]
		if ok {
			return value, nil
		}
	}
	return "", fmt.Errorf("state does not contain value for %s annotation", key)
}

func WriteState(state *specs.State) error {
	containerStateDir := filepath.Join(stateDir, state.ID)
	if err := os.MkdirAll(containerStateDir, 0755); err != nil {
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
	if err := os.WriteFile(tmpFilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file %s: %w", tmpFilePath, err)
	}
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		os.Remove(tmpFilePath)
		return fmt.Errorf("failed to rename temp file %s to %s", tmpFilePath, filePath)
	}
	return nil
}

func ReadState(containerID string) (*specs.State, error) {
	stateFilePath := filepath.Join(stateDir, containerID, stateFileName)
	f, err := os.Open(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", stateFilePath, err)
	}
	defer f.Close()
	var s specs.State
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func RemoveState(containerID string) error {
	containerStateDir := filepath.Join(stateDir, containerID)
	if err := os.RemoveAll(containerStateDir); err != nil {
		return fmt.Errorf("cannot remove container state dir: %w", err)
	}
	return nil
}
