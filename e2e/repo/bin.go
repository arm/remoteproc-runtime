package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func GetRemoteprocSimulator(binOutDir string) (string, error) {
	const repoDirName = "remoteproc-simulator"
	const version = "0.0.8"

	artifactURL := fmt.Sprintf(
		"https://github.com/arm/remoteproc-simulator/releases/download/v%s/remoteproc-simulator_%s_linux_%s.tar.gz",
		version,
		version,
		runtime.GOARCH,
	)

	downloader := exec.Command("curl", "-L", "-o", filepath.Join(binOutDir, "simulator.tar.gz"), artifactURL)
	if out, err := downloader.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to download remoteproc-simulator: %s\n%s", err, out)
	}

	extractor := exec.Command("tar", "-xzf", filepath.Join(binOutDir, "simulator.tar.gz"), "-C", binOutDir)
	if out, err := extractor.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to extract remoteproc-simulator: %s\n%s", err, out)
	}

	simulatorPath := filepath.Join(binOutDir, repoDirName)
	return simulatorPath, nil
}
