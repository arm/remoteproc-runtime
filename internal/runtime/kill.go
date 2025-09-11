package runtime

import (
	"fmt"
	"syscall"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/proxy"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Kill(containerID string, signal syscall.Signal) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if state.Pid > 0 {
		if err := proxy.SendSignal(state.Pid, signal); err != nil {
			return fmt.Errorf("failed to send signal: %w", err)
		}
	}

	state.Status = specs.StateStopped
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}
