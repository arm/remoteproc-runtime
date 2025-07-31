package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Create(containerID string, bundlePath string) error {
	spec, err := oci.ReadSpec(filepath.Join(bundlePath, "config.json"))
	if err != nil {
		return fmt.Errorf("can't read spec: %w", err)
	}

	mcu, ok := spec.Annotations[oci.SpecMCU]
	if !ok {
		return fmt.Errorf("%s not set in bundle annotations", oci.SpecMCU)
	}
	devicePath, err := remoteproc.FindDevicePath(mcu)
	if err != nil {
		return fmt.Errorf("can't determine remoteproc mcu path: %w", err)
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

	state := oci.NewState(containerID, bundlePath)
	state.Annotations[oci.StateMCUResolvedPath] = devicePath
	state.Annotations[oci.StateFirmwareName] = storedFirmwareName
	if err := oci.WriteState(state); err != nil {
		return err
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
