package shim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const runtimeBinName = "remoteproc-runtime"

func executeCreate(containerID string, bundlePath string) error {
	cmd := exec.Command(runtimeBinName, "create", "--bundle", bundlePath, containerID)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runtime create failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func executeStart(containerID string) error {
	cmd := exec.Command(runtimeBinName, "start", containerID)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runtime start failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func executeDelete(containerID string) error {
	cmd := exec.Command(runtimeBinName, "delete", containerID)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runtime delete failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func executeKill(containerID string, signal syscall.Signal) error {
	args := []string{"kill", containerID, fmt.Sprintf("%d", signal)}
	cmd := exec.Command(runtimeBinName, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runtime kill failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func executeState(containerID string) (*specs.State, error) {
	cmd := exec.Command(runtimeBinName, "state", containerID)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("runtime state failed: %w, stderr: %s", err, stderr.String())
	}

	var state specs.State
	if err := json.Unmarshal(stdout.Bytes(), &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w, output: %s", err, stdout.String())
	}

	return &state, nil
}

func getPid(containerID string) (int, error) {
	state, err := executeState(containerID)
	if err != nil {
		return 0, err
	}
	return state.Pid, nil
}
