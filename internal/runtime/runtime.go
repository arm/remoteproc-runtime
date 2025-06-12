package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Start(containerID string) error {
	ociState, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	annotations, err := oci.NewRemoteprocAnnotations(ociState)
	if err != nil {
		return fmt.Errorf("failed to read state annotations: %w", err)
	}

	if err := remoteproc.SetFirmwareAndStart(annotations.DevicePath, annotations.FirmwareName); err != nil {
		return fmt.Errorf("failed to run firmware: %w", err)
	}

	ociState.Status = specs.StateRunning
	if err := oci.WriteState(ociState); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

func Kill(containerID string) error {
	// TODO: actual echo stop > /sys/class/remoteproc/...
	ociState, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	annotations, err := oci.NewRemoteprocAnnotations(ociState)
	if err != nil {
		return fmt.Errorf("failed to read state annotations: %w", err)
	}
	if err := remoteproc.Stop(annotations.DevicePath); err != nil {
		return fmt.Errorf("failed to stop firmware: %w")
	}
	ociState.Status = specs.StateStopped
	if err := oci.WriteState(ociState); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}

func Delete(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	annotations, err := oci.NewRemoteprocAnnotations(state)
	if err != nil {
		return fmt.Errorf("failed to read state annotations: %w", err)
	}
	_ = remoteproc.RemoveFirmware(annotations.FirmwareName)

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
