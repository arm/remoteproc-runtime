package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/devicetree"
	"github.com/Arm-Debug/remoteproc-shim/internal/sysfs/remoteproc"
	taskAPI "github.com/containerd/containerd/api/runtime/task/v2"
	"github.com/containerd/containerd/api/types"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// TODO: this needs to eventually live in `runtime` namespace, but currently is coupled to `RootFS` from CreateTaskRequest. Looks like `rootfs` specified in `config.json` is empty, so the request is the only way to get our hands on the firmware.
func CreateContainer(req *taskAPI.CreateTaskRequest) error {
	params, err := newContainerParams(req)
	if err != nil {
		return err
	}
	return createContainer(params)
}

type target struct {
	Board string
	MCU   string
}

type containerParams struct {
	ID                  string
	BundlePath          string
	FirmwareLookupPaths []string
	FirmwareName        string
	target
}

func newContainerParams(req *taskAPI.CreateTaskRequest) (containerParams, error) {
	spec, err := oci.ReadSpec(filepath.Join(req.Bundle, "config.json"))
	if err != nil {
		return containerParams{}, fmt.Errorf("can't read spec: %w", err)
	}

	firmwareLookupPaths := listFirmwareLookupPaths(req.Bundle, spec, req.Rootfs)

	firmwareName, err := extractFirmwareName(spec)
	if err != nil {
		return containerParams{}, fmt.Errorf("can't extract firmware name: %w", err)
	}

	t, err := extractTarget(spec)
	if err != nil {
		return containerParams{}, fmt.Errorf("can't extract target data: %w", err)
	}

	return containerParams{
		ID:                  req.ID,
		BundlePath:          req.Bundle,
		FirmwareLookupPaths: firmwareLookupPaths,
		FirmwareName:        firmwareName,
		target:              t,
	}, nil
}

func listFirmwareLookupPaths(bundlePath string, spec *specs.Spec, rootFS []*types.Mount) []string {
	paths := []string{}
	if spec.Root != nil && spec.Root.Path != "" {
		root := spec.Root.Path
		if !filepath.IsAbs(root) {
			root = filepath.Join(bundlePath, root)
		}
		paths = append(paths, root)
	}

	for _, mount := range rootFS {
		const prefix = "lowerdir="
		for _, option := range mount.Options {
			if path, found := strings.CutPrefix(option, prefix); found {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

func extractFirmwareName(spec *specs.Spec) (string, error) {
	if len(spec.Process.Args) != 1 {
		return "", fmt.Errorf("expected exactly one process argument")
	}
	return spec.Process.Args[0], nil
}

func extractTarget(spec *specs.Spec) (target, error) {
	env := parseEnvVars(spec.Process.Env)
	mcu, ok := env["MCU"]
	if !ok {
		return target{}, fmt.Errorf("MCU env variable not set")
	}

	board, ok := env["BOARD"]
	if !ok {
		return target{}, fmt.Errorf("BOARD env variable not set")
	}

	return target{
		MCU:   mcu,
		Board: board,
	}, nil
}

func parseEnvVars(envVars []string) map[string]string {
	result := map[string]string{}
	for _, envVar := range envVars {
		chunks := strings.SplitN(envVar, "=", 2)
		if len(chunks) != 2 {
			continue
		}
		key := chunks[0]
		value := chunks[1]
		if key != "" {
			result[key] = value
		}
	}
	return result
}

const pid uint32 = 1

func createContainer(params containerParams) error {
	if err := validateBoardMatchesModel(params.Board); err != nil {
		return err
	}

	devicePath, err := remoteproc.FindDevicePath(params.MCU)
	if err != nil {
		return fmt.Errorf("can't determine remoteproc mcu path: %w", err)
	}

	firmwarePath, err := findFirmware(params.FirmwareLookupPaths, params.FirmwareName)
	if err != nil {
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

	state := oci.NewState(params.ID, int(pid), params.BundlePath)
	annotations := oci.RemoteprocAnnotations{
		MCU:          params.MCU,
		DevicePath:   devicePath,
		FirmwareName: storedFirmwareName,
	}
	annotations.Apply(state)

	if err := oci.WriteState(state); err != nil {
		return err
	}

	needCleanup = false
	return nil
}

func findFirmware(lookupPaths []string, firmwareFileName string) (string, error) {
	for _, path := range lookupPaths {
		fullPath := filepath.Join(path, firmwareFileName)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("firmware %s not found in any of the provided paths %v", firmwareFileName, lookupPaths)
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
