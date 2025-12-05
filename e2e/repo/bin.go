package repo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	err = gitCheckoutToLatestRelease(repoDir, env)
	if err != nil {
		return "", err
	}

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

func gitCheckoutToLatestRelease(repoDir string, env map[string]string) error {
	tagCmd := exec.Command("git", "tag", "--list", "--sort=-version:refname")
	tagCmd.Dir = repoDir
	tagCmd.Env = os.Environ()
	for k, v := range env {
		tagCmd.Env = append(tagCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	tagOutput, err := tagCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe for tag listing: %w", err)
	}

	if err := tagCmd.Start(); err != nil {
		return fmt.Errorf("failed to list tags for remoteproc simulator: %w", err)
	}

	var latestTag string
	scanner := bufio.NewScanner(tagOutput)
	for scanner.Scan() {
		tag := strings.TrimSpace(scanner.Text())
		if tag == "" {
			continue
		}
		if !strings.HasPrefix(tag, "v") {
			continue
		}
		latestTag = tag
		break
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read tags for remoteproc simulator: %w", err)
	}
	if err := tagCmd.Wait(); err != nil {
		return fmt.Errorf("failed to list tags for remoteproc simulator: %w", err)
	}

	if latestTag == "" {
		return fmt.Errorf("no semantic version tag found in remoteproc simulator repository")
	}

	checkout := exec.Command("git", "checkout", "--quiet", latestTag)
	checkout.Dir = repoDir
	checkout.Env = os.Environ()
	for k, v := range env {
		checkout.Env = append(checkout.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if out, err := checkout.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout latest tag %s: %w\n%s", latestTag, err, out)
	}
	return nil
}
