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

type MCUAnnotations struct {
	Requested    string
	ResolvedPath string
}

func (a MCUAnnotations) Apply(state *specs.State) {
	if state.Annotations == nil {
		state.Annotations = map[string]string{}
	}
	state.Annotations["remoteproc.requested"] = a.Requested
	state.Annotations["remoteproc.resolved"] = a.ResolvedPath
}

func WriteStateFile(state *specs.State) error {
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

func RemoveStateFile(containerID string) error {
	return nil
}
