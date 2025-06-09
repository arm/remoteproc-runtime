package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arm-Debug/remoteproc-shim/internal/remoteproc"
	taskAPI "github.com/containerd/containerd/api/runtime/task/v2"
	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/log"
)

func CreateContainer(req *taskAPI.CreateTaskRequest) error {
	params, err := newContainerParams(req)
	if err != nil {
		return err
	}

	log.L.WithField("params", fmt.Sprintf("%#v", params)).Info("remoteproc.CreateContainer")

	container, err := createContainer(params)
	if err != nil {
		return err
	}

	log.L.WithField("container", fmt.Sprintf("%#v", container)).Info("remoteproc.CreateContainer")

	return fmt.Errorf("not implemented yet")
}

type containerParams struct {
	ContainerID  string
	FirmwarePath string
	FirmwareName string
	target
}

type target struct {
	MCU   string
	Board string
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
		ContainerID:  req.ID,
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

type Bundle struct {
}

func extractFirmwareName(spec *oci.Spec) (string, error) {
	if len(spec.Process.Args) != 1 {
		return "", fmt.Errorf("expected exactly one process argument")
	}
	return spec.Process.Args[0], nil
}

func extractTarget(spec *oci.Spec) (target, error) {
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

type container struct {
}

func createContainer(params containerParams) (container, error) {
	if err := validateContainerParams(params); err != nil {
		return container{}, err
	}

	// 1. Validate that board and mcu matches the HOST
	// 1. Prepare and set container labels - we're going to use them later to start the container - DETERMINE what exactly is required to start
	// 2.

	return container{}, nil
}

func validateContainerParams(params containerParams) error {
	firmwareFilePath := filepath.Join(params.FirmwarePath, params.FirmwareName)
	if _, err := os.Stat(firmwareFilePath); err != nil {
		return fmt.Errorf("firmware file %s not accessible: %w", firmwareFilePath, err)
	}

	if err := remoteproc.CheckMCUExists(params.target.MCU); err != nil {
		return fmt.Errorf("mcu check failed: %w", err)
	}

	return nil
}
