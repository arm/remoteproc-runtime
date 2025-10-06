package limavm

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/repo"
	"github.com/Arm-Debug/remoteproc-runtime/e2e/runner"
)

var (
	prepareLimaVMScript  = filepath.Join(repo.MustFindRootDir(), "e2e", "limavm", "prepare-lima-vm.sh")
	teardownLimaVMScript = filepath.Join(repo.MustFindRootDir(), "e2e", "limavm", "teardown-lima-vm.sh")
)

type LimaVM struct {
	name string
}

var BinBuildEnv = map[string]string{
	"GOOS": "linux",
}

func NewWithDocker(mountDir string, buildContext string, bins repo.Bins) (LimaVM, error) {
	return New("docker", mountDir, buildContext, string(bins.Runtime), string(bins.Shim))
}

func NewWithPodman(mountDir string, buildContext string, runtimeBin repo.RuntimeBin) (LimaVM, error) {
	return New("podman", mountDir, buildContext, string(runtimeBin))
}

func New(template string, mountDir string, buildContext string, binsToInstall ...string) (LimaVM, error) {
	cmd := exec.Command(
		prepareLimaVMScript,
		append([]string{template, mountDir, buildContext}, binsToInstall...)...,
	)
	streamer := runner.NewStreamingCmd(cmd).WithPrefix("prepare-vm")

	if err := streamer.Start(); err != nil {
		return LimaVM{}, fmt.Errorf("failed to start prepare script: %w", err)
	}

	if err := streamer.Wait(); err != nil {
		return LimaVM{}, fmt.Errorf("failed to prepare VM: %w", err)
	}

	vmName := strings.TrimSpace(streamer.Output())
	if vmName == "" {
		return LimaVM{}, fmt.Errorf("prepare script did not return VM name")
	}

	return LimaVM{name: vmName}, nil
}

func (vm LimaVM) Cleanup() {
	cmd := exec.Command(teardownLimaVMScript, vm.name)
	_ = cmd.Run()
}

func (vm LimaVM) cmd(name string, args ...string) *exec.Cmd {
	allArgs := append([]string{"shell", vm.name, name}, args...)
	return exec.Command("limactl", allArgs...)
}

func (vm LimaVM) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
	cmd := vm.cmd(name, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		return stdout, stderr, fmt.Errorf("cmd failed: %w\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	return stdout, stderr, nil
}
