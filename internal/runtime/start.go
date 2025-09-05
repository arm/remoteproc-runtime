package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/proxy"
	"github.com/Arm-Debug/remoteproc-runtime/internal/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Start(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if err := remoteproc.SetFirmware(
		state.Annotations[oci.StateResolvedPath],
		state.Annotations[oci.StateFirmware],
	); err != nil {
		return fmt.Errorf("failed to set firmware: %w", err)
	}

	proxyProcess, err := proxy.FindProcess(state.Pid)
	if err != nil {
		return fmt.Errorf("failed to get proxy process: %w", err)
	}

	if err := proxyProcess.StartFirmware(); err != nil {
		return fmt.Errorf("failed to signal proxy process: %w", err)
	}

	state.Status = specs.StateRunning
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
