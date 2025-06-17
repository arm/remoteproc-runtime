package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/devicetree"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Create(containerID string, bundlePath string) error {
	spec, err := oci.ReadSpec(filepath.Join(bundlePath, "config.json"))
	if err != nil {
		return fmt.Errorf("can't read spec: %w", err)
	}

	target, err := extractTarget(spec)
	if err != nil {
		return fmt.Errorf("can't extract target data: %w", err)
	}

	if err := validateBoardMatchesModel(target.Board); err != nil {
		return err
	}

	devicePath, err := remoteproc.FindDevicePath(target.MCU)
	if err != nil {
		return fmt.Errorf("can't determine remoteproc mcu path: %w", err)
	}

	firmwareName, err := extractFirmwareName(spec)
	if err != nil {
		return fmt.Errorf("can't extract firmware name: %w", err)
	}
	absRootFS := spec.Root.Path
	if !filepath.IsAbs(absRootFS) {
		absRootFS = filepath.Join(bundlePath, absRootFS, "foo")
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

func extractTarget(spec *specs.Spec) (target, error) {
	mcu, ok := spec.Annotations[oci.SpecMCU]
	if !ok {
		return target{}, fmt.Errorf("%s not set in bundle annotations", oci.SpecMCU)
	}

	board, ok := spec.Annotations[oci.SpecBoard]
	if !ok {
		return target{}, fmt.Errorf("%s not set in bundle annotations", oci.SpecBoard)
	}

	return target{
		MCU:   mcu,
		Board: board,
	}, nil
}

func validateBoardMatchesModel(wantBoard string) error {
	sysModel, err := devicetree.GetModel()
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}
	if sysModel != wantBoard {
		return fmt.Errorf(
			"target board %q does not match system model %q", wantBoard, sysModel,
		)
	}
	return nil
}

func extractFirmwareName(spec *specs.Spec) (string, error) {
	if len(spec.Process.Args) != 1 {
		return "", fmt.Errorf("expected exactly one process argument")
	}
	return spec.Process.Args[0], nil
}

type target struct {
	Board string
	MCU   string
}

func validateFirmwareExists(firmwareFilePath string) error {
	if _, err := os.Stat(firmwareFilePath); err != nil {
		return fmt.Errorf("firmware does not exist: %w", err)
	}
	return nil
}
