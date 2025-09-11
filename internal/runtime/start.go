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

	if err := proxy.StartFirmware(state.Pid); err != nil {
		return fmt.Errorf("failed to start firmware: %w", err)
	}

	state.Status = specs.StateRunning
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
