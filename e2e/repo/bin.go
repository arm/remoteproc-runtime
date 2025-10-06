package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type RuntimeBin string

func BuildRuntimeBin(binOutDir string, rootPathPrefix string, env map[string]string) (RuntimeBin, error) {
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
	return RuntimeBin(binOut), nil
}

type ShimBin string

func BuildShimBin(binOutDir string, env map[string]string) (ShimBin, error) {
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
	return ShimBin(binOut), nil
}

type Bins struct {
	Shim    ShimBin
	Runtime RuntimeBin
}

func BuildBothBins(binOutDir string, rootPathPrefix string, env map[string]string) (Bins, error) {
	runtime, err := BuildRuntimeBin(binOutDir, rootPathPrefix, env)
	if err != nil {
		return Bins{}, err
	}

	shim, err := BuildShimBin(binOutDir, env)
	if err != nil {
		return Bins{}, err
	}

	return Bins{
		Runtime: runtime,
		Shim:    shim,
	}, nil
}
