package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Start(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if err := remoteproc.SetFirmwareAndStart(
		state.Annotations[oci.StateMCUResolvedPath],
		state.Annotations[oci.StateFirmwareName],
	); err != nil {
		return fmt.Errorf("failed to run firmware: %w", err)
	}

	state.Status = specs.StateRunning
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

func Kill(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if err := remoteproc.Stop(state.Annotations[oci.StateMCUResolvedPath]); err != nil {
		return fmt.Errorf("failed to stop firmware: %w", err)
	}
	state.Status = specs.StateStopped
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}

func Delete(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	_ = remoteproc.RemoveFirmware(state.Annotations[oci.StateFirmwareName])

	if err := oci.RemoveState(containerID); err != nil {
		return fmt.Errorf("failed to remove state: %w", err)
	}
	return nil
}

func State(containerID string) (*specs.State, error) {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	return state, nil
}
