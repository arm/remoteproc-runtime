package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func BuildRuntimeBin(binOutDir string, rootPathPrefix string, env map[string]string) (string, error) {
	const binToBuild = "remoteproc-runtime"
	repoRootDir, err := findRootDir()
	if err != nil {
		return "", err
	}
	binOut := filepath.Join(binOutDir, binToBuild)
	toBuild := filepath.Join(repoRootDir, "cmd", binToBuild)
	rootPathLDFlag := fmt.Sprintf("-X github.com/arm/remoteproc-runtime/internal/rootpath.prefix=%s", rootPathPrefix)

	build := exec.Command(
		"go", "build",
		"-ldflags", rootPathLDFlag,
		"-o", binOut,
		toBuild,
	)
	build.Env = os.Environ()
	for k, v := range env {
		build.Env = append(build.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary %s: %s\n%s", binOut, err, out)
	}
	return binOut, nil
}

func BuildShimBin(binOutDir string, env map[string]string) (string, error) {
	const binToBuild = "containerd-shim-remoteproc-v1"
	repoRootDir, err := findRootDir()
	if err != nil {
		return "", err
	}
	binOut := filepath.Join(binOutDir, binToBuild)
	toBuild := filepath.Join(repoRootDir, "cmd", binToBuild)

	build := exec.Command(
		"go", "build",
		"-o", binOut,
		toBuild,
	)
	build.Env = os.Environ()
	for k, v := range env {
		build.Env = append(build.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary %s: %s\n%s", binOut, err, out)
	}
	return binOut, nil
}

func BuildBothBins(binOutDir string, rootPathPrefix string, env map[string]string) ([]string, error) {
	runtime, err := BuildRuntimeBin(binOutDir, rootPathPrefix, env)
	if err != nil {
		return nil, err
	}

	shim, err := BuildShimBin(binOutDir, env)
	if err != nil {
		return nil, err
	}

	return []string{runtime, shim}, nil
}

func BuildRemoteprocSimulator(binOutDir string, env map[string]string) (string, error) {
	const modulePath = "github.com/arm/remoteproc-simulator/cmd/remoteproc-simulator"
	binOut := filepath.Join(binOutDir, "remoteproc-simulator")

	build := exec.Command(
		"go", "build",
		"-o", binOut,
		modulePath,
	)
	build.Env = os.Environ()
	for k, v := range env {
		build.Env = append(build.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build remoteproc simulator: %s\n%s", err, out)
	}
	return binOut, nil
}
