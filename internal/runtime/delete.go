package runtime

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/arm/remoteproc-runtime/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Delete(logger *slog.Logger, containerID string, force bool) error {
	if force {
		forceDelete(logger, containerID)
		return nil
	} else {
		return delete(containerID)
	}
}

func delete(containerID string) error {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if state.Status == specs.StateRunning {
		return fmt.Errorf("cannot delete running container %s", containerID)
	}

	firmwarePath, ok := state.Annotations[oci.OptionalStateStoredFirmwarePath]
	if ok {
		if err := os.Remove(firmwarePath); err != nil {
			return fmt.Errorf("failed to remove firmware: %w", err)
		}
	}

	if err := oci.RemoveState(containerID); err != nil {
		return fmt.Errorf("failed to remove state: %w", err)
	}

	return nil
}

func forceDelete(logger *slog.Logger, containerID string) {
	state, err := oci.ReadState(containerID)
	if err != nil {
		logger.Error("failed to read state", "error", err)
		return
	}

	if state.Status == specs.StateRunning {
		if err := Kill(containerID, syscall.SIGKILL); err != nil {
			logger.Error("failed to kill container", "error", err)
		}
	}

	firmwarePath, ok := state.Annotations[oci.OptionalStateStoredFirmwarePath]
	if ok {
		if err := os.Remove(firmwarePath); err != nil {
			logger.Error("failed to remove firmware", "error", err)
		}
	}

	if err := oci.RemoveState(containerID); err != nil {
		logger.Error("failed to remove state", "error", err)
	}
}
