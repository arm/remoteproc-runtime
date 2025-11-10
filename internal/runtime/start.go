package runtime

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/arm/remoteproc-runtime/internal/oci"
	"github.com/arm/remoteproc-runtime/internal/proxy"
	"github.com/arm/remoteproc-runtime/internal/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Start(logger *slog.Logger, containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	sourceFirmwarePath := state.Annotations[oci.StateFirmwarePath]
	destFirmwareDir := remoteproc.GetSystemFirmwarePath()
	storedFirmwarePath, err := remoteproc.StoreFirmware(sourceFirmwarePath, destFirmwareDir)
	if err != nil {
		return fmt.Errorf("failed to store firmware file %s to %s: %w", sourceFirmwarePath, destFirmwareDir, err)
	}
	state.Annotations[oci.OptionalStateStoredFirmwarePath] = storedFirmwarePath
	needCleanup := true
	defer func() {
		if needCleanup {
			_ = os.Remove(storedFirmwarePath)
		}
	}()

	if err := remoteproc.SetFirmware(
		state.Annotations[oci.StateDriverPath],
		storedFirmwarePath,
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
