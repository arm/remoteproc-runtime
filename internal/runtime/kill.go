package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

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
