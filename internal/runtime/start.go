package runtime

import (
	"fmt"

	"github.com/arm/remoteproc-runtime/internal/oci"
	"github.com/arm/remoteproc-runtime/internal/proxy"
	"github.com/arm/remoteproc-runtime/internal/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Start(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	firmwarePath := state.Annotations[oci.StateFirmwarePath]
	storedFirmwareName, err := remoteproc.StoreFirmware(firmwarePath)
	if err != nil {
		return fmt.Errorf("failed to store firmware file %s: %w", firmwarePath, err)
	}
	state.Annotations[oci.OptionalStateStoredFirmwareName] = storedFirmwareName
	needCleanup := true
	defer func() {
		if needCleanup {
			_ = remoteproc.RemoveFirmware(storedFirmwareName)
		}
	}()

	if err := remoteproc.SetFirmware(
		state.Annotations[oci.StateDriverPath],
		storedFirmwareName,
	); err != nil {
		return fmt.Errorf("failed to set firmware: %w", err)
	}

	if err := proxy.StartFirmware(state.Pid); err != nil {
		return fmt.Errorf("failed to start firmware: %w", err)
	}

	state.Status = specs.StateRunning
	if err := oci.WriteState(state); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	needCleanup = false
	return nil
}
