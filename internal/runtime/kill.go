package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Kill(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	isContainerCurrentlyRunningOnRemoteProcessor := state.Status == specs.StateRunning
	if isContainerCurrentlyRunningOnRemoteProcessor {
		// Don't want to kill somebody else's remote proc execution
		if err := remoteproc.Stop(state.Annotations[oci.StateResolvedPath]); err != nil {
			return fmt.Errorf("failed to stop firmware: %w", err)
		}
	}
	state.Status = specs.StateStopped
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}
