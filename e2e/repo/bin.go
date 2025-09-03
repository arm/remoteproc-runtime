package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func BuildBinary(binToBuild string, binOutDir string, rootPathPrefix string, env map[string]string) (string, error) {
	bin := filepath.Join(binOutDir, binToBuild)
	rootPathLDFlag := fmt.Sprintf("-X github.com/Arm-Debug/remoteproc-runtime/internal/rootpath.prefix=%s", rootPathPrefix)
	repoRootDir, err := findRootDir()
	if err != nil {
		return "", err
	}
	toBuild := filepath.Join(repoRootDir, "cmd", binToBuild)
	build := exec.Command(
		"go", "build",
		"-ldflags", rootPathLDFlag,
		"-o", bin,
		toBuild,
	)

	build.Env = os.Environ()
	for k, v := range env {
		build.Env = append(build.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary %s: %s\n%s", bin, err, out)
	}
	return bin, nil
}
