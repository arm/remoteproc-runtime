package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/proxy"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Kill(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if state.Pid > 0 {
		proxyProcess, err := proxy.FindProcess(state.Pid)
		if err == nil {
			if err := proxyProcess.StopFirmware(); err != nil {
				return fmt.Errorf("failed to stop firmware: %w", err)
			}
		}
	}

	state.Status = specs.StateStopped
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}
