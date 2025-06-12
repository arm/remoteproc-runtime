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
	ID           string
	BundlePath   string
	FirmwarePath string
	FirmwareName string
	target
}

func newContainerParams(req *taskAPI.CreateTaskRequest) (containerParams, error) {
	firmwarePath, err := extractFirmwarePathFromRootFS(req.Rootfs)
	if err != nil {
		return containerParams{}, fmt.Errorf("can't extract firmware path: %w", err)
	}

	spec, err := oci.ReadSpec(filepath.Join(req.Bundle, "config.json"))
	if err != nil {
		return containerParams{}, fmt.Errorf("can't read spec: %w", err)
	}

	firmwareName, err := extractFirmwareName(spec)
	if err != nil {
		return containerParams{}, fmt.Errorf("can't extract firmware name: %w", err)
	}

	t, err := extractTarget(spec)
	if err != nil {
		return containerParams{}, fmt.Errorf("can't extract target data: %w", err)
	}

	return containerParams{
		ID:           req.ID,
		BundlePath:   req.Bundle,
		FirmwarePath: firmwarePath,
		FirmwareName: firmwareName,
		target:       t,
	}, nil
}

func extractFirmwarePathFromRootFS(rootFS []*types.Mount) (string, error) {
	if len(rootFS) == 0 {
		return "", fmt.Errorf("rootfs is empty")
	}
	// Firmware must exist in top-most layer
	topMount := rootFS[0]
	const prefix = "lowerdir="
	for _, option := range topMount.Options {
		if path, found := strings.CutPrefix(option, prefix); found {
			return path, nil
		}
	}
	return "", fmt.Errorf("lowerdir option not found in top mount")
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
	if err := validateFirmwareExists(params.FirmwarePath, params.FirmwareName); err != nil {
		return err
	}
	if err := validateBoardMatchesModel(params.Board); err != nil {
		return err
	}

	mcuPath, err := remoteproc.FindMCUDirectory(params.MCU)
	if err != nil {
		return fmt.Errorf("can't determine remoteproc mcu path: %w", err)
	}

	state := oci.NewState(params.ID, int(pid), params.BundlePath)
	annotations := oci.RemoteprocAnnotations{
		RequestedMCU: params.MCU,
		ResolvedPath: mcuPath,
	}
	annotations.Apply(state)
	if err := oci.WriteState(state); err != nil {
		return err
	}

	return nil
}

func validateFirmwareExists(firmwarePath, firmwareName string) error {
	firmwareFilePath := filepath.Join(firmwarePath, firmwareName)
	if _, err := os.Stat(firmwareFilePath); err != nil {
		return fmt.Errorf("firmware file %s not accessible: %w", firmwareFilePath, err)
	}
	return nil
}

func validateBoardMatchesModel(wantBoard string) error {
	sysModel, err := devicetree.GetModel()
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}
	if sysModel != wantBoard {
		return fmt.Errorf(
			"target board %s does not match system model %s", wantBoard, sysModel,
		)
	}
	return nil
}
