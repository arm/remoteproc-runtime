package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Start(containerID string) (uint32, error) {
	// TODO: actual echo start > /sys/class/remoteproc/...
	ociState, err := oci.ReadState(containerID)
	if err != nil {
		return 0, fmt.Errorf("failed to read state: %w", err)
	}
	ociState.Status = specs.StateRunning
	if err := oci.WriteState(ociState); err != nil {
		return 0, fmt.Errorf("failed to write state: %w", err)
	}

	return uint32(ociState.Pid), nil
}

func Kill(containerID string) error {
	// TODO: actual echo stop > /sys/class/remoteproc/...
	ociState, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	ociState.Status = specs.StateStopped
	if err := oci.WriteState(ociState); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}

func Delete(containerID string) error {
	// TODO: Does rproc need additional cleanup?
	//       - restore path where firmware is loaded from
	//       - delete firmware file if moved outside of rootfs
	if err := oci.RemoveState(containerID); err != nil {
		return fmt.Errorf("failed to remove state: %w", err)
	}
	return nil
}

func State(containerID string) (*specs.State, error) {
	ociState, err := oci.ReadState(containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	return ociState, nil
}
