package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/Arm-Debug/remoteproc-runtime/internal/proxy"
	"github.com/Arm-Debug/remoteproc-runtime/internal/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Create(containerID string, bundlePath string, pidFile string) error {
	spec, err := oci.ReadSpec(bundlePath)
	if err != nil {
		return fmt.Errorf("can't read spec: %w", err)
	}

	name := spec.Annotations[oci.SpecName]
	devicePath, err := remoteproc.FindDevicePath(name)
	if err != nil {
		return fmt.Errorf("can't determine remoteproc path: %w", err)
	}

	firmwareName, err := extractFirmwareName(spec)
	if err != nil {
		return fmt.Errorf("can't extract firmware name: %w", err)
	}
	absRootFS := spec.Root.Path
	if !filepath.IsAbs(absRootFS) {
		absRootFS = filepath.Join(bundlePath, absRootFS)
	}
	firmwarePath := filepath.Join(absRootFS, firmwareName)
	if err := validateFirmwareExists(firmwarePath); err != nil {
		return err
	}
	storedFirmwareName, err := remoteproc.StoreFirmware(firmwarePath)
	if err != nil {
		return fmt.Errorf("failed to store firmware file %s: %w", firmwarePath, err)
	}
	needCleanup := true
	defer func() {
		if needCleanup {
			_ = remoteproc.RemoveFirmware(storedFirmwareName)
		}
	}()

	pid, err := proxy.NewProcess(devicePath)
	if err != nil {
		return fmt.Errorf("failed to start proxy process: %w", err)
	}
	defer func() {
		if needCleanup {
			_ = proxy.StopFirmware(pid)
		}
	}()

	state := oci.NewState(containerID, bundlePath)
	state.Pid = pid
	state.Annotations[oci.StateResolvedPath] = devicePath
	state.Annotations[oci.StateFirmware] = storedFirmwareName
	if err := oci.WriteState(state); err != nil {
		return err
	}

	if pidFile != "" {
		if err := writePidFile(pidFile, pid); err != nil {
			return fmt.Errorf("failed to write PID file: %w", err)
		}
	}

	needCleanup = false
	return nil
}

func extractFirmwareName(spec *specs.Spec) (string, error) {
	if len(spec.Process.Args) != 1 {
		return "", fmt.Errorf("expected exactly one process argument")
	}
	return spec.Process.Args[0], nil
}

func validateFirmwareExists(firmwareFilePath string) error {
	if _, err := os.Stat(firmwareFilePath); err != nil {
		return fmt.Errorf("firmware does not exist: %w", err)
	}
	return nil
}

func writePidFile(pidFile string, pid int) error {
	content := fmt.Sprintf("%d", pid)
	return os.WriteFile(pidFile, []byte(content), 0644)
}
