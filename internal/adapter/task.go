package adapter

import (
	"context"
	"fmt"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
)

func WaitForExit(ctx context.Context, containerID string) (uint32, error) {
	ociState, err := oci.ReadState(containerID)
	if err != nil {
		return 0, err
	}

	annotations, err := oci.NewRemoteprocAnnotations(ociState)
	if err != nil {
		return 0, err
	}

	// TODO: actual polling of /sys/class/remoteproc/.../state
	remoteprocState, err := remoteproc.GetState(annotations.ResolvedPath)
	if err != nil {
		return 0, err
	}

	switch remoteprocState {
	case remoteproc.StateOffline:
		return 0, nil // Clean exit
	case remoteproc.StateCrashed:
		return 1, nil // Error exit
	case remoteproc.StateInvalid:
		return 2, nil
	default:
		return 0, fmt.Errorf("looks like remote processor is still %s", remoteprocState)
	}
}
