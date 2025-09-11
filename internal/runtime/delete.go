package runtime

import (
	"fmt"
	"syscall"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Delete(containerID string, force bool) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if force && state.Status == specs.StateRunning {
		if err := Kill(containerID, syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to kill container: %w", err)
		}
	} else if state.Status == specs.StateRunning {
		return fmt.Errorf("cannot delete running container %s", containerID)
	}

	_ = remoteproc.RemoveFirmware(state.Annotations[oci.StateFirmware])

	if err := oci.RemoveState(containerID); err != nil {
		return fmt.Errorf("failed to remove state: %w", err)
	}
	return nil
}
