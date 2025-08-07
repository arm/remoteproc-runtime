package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/sysfs/remoteproc"
)

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
