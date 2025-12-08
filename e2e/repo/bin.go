package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/stretchr/testify/assert"
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
	const repoURL = "https://github.com/arm/remoteproc-simulator.git"
	const repoDirName = "remoteproc-simulator"
	const tag = "v0.0.8"

	tempDir, err := os.MkdirTemp("", "remoteproc-simulator-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	repoDir := filepath.Join(tempDir, repoDirName)

	err = gitClone(repoURL, repoDir, env)
	if err != nil {
		return "", err
	}

	err = gitCheckoutToTag(repoDir, tag, env)
	if err != nil {
		return "", err
	}

	assert.DirExists(nil, repoDir, "remoteproc-simulator repo should be cloned")

	binOut := filepath.Join(binOutDir, "remoteproc-simulator")
	build := exec.Command(
		"go", "build",
		"-o", binOut,
		"./cmd/remoteproc-simulator",
	)
	build.Dir = repoDir
	build.Env = os.Environ()
	for k, v := range env {
		build.Env = append(build.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build remoteproc simulator: %s\n%s", err, out)
	}
	return binOut, nil
}

func gitClone(repoURL, repoDir string, env map[string]string) error {
	clone := exec.Command("git", "clone", "--quiet", repoURL, repoDir)
	clone.Env = os.Environ()
	for k, v := range env {
		clone.Env = append(clone.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if out, err := clone.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone remoteproc simulator repository: %w\n%s", err, out)
	}
	return nil
}

func gitCheckoutToTag(repoDir, tag string, env map[string]string) error {
	checkout := exec.Command("git", "checkout", "--quiet", tag)
	checkout.Dir = repoDir
	checkout.Env = os.Environ()
	for k, v := range env {
		checkout.Env = append(checkout.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if out, err := checkout.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout tag %s: %w\n%s", tag, err, out)
	}
	return nil
}
