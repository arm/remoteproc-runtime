package shared

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func BuildBinary(binToBuild string, binOutDir string, rootPathPrefix string) (string, error) {
	bin := filepath.Join(binOutDir, binToBuild)
	rootPathLDFlag := fmt.Sprintf("-X github.com/Arm-Debug/remoteproc-runtime/internal/rootpath.prefix=%s", rootPathPrefix)
	repoRootDir, err := findRepoRootDir()
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
	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary %s: %s\n%s", bin, err, out)
	}
	return bin, nil
}
