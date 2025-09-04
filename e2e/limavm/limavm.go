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

var prepareLimaVMScript = filepath.Join(repo.MustFindRootDir(), "e2e", "limavm", "prepare-lima-vm.sh")
var teardownLimaVMScript = filepath.Join(repo.MustFindRootDir(), "e2e", "limavm", "teardown-lima-vm.sh")

type LimaVM struct {
	name string
}

var BinBuildEnv = map[string]string{
	"GOOS": "linux",
}

func New(mountDir string, bins repo.Bins, absImageTar string) (LimaVM, error) {
	cmd := exec.Command(
		prepareLimaVMScript,
		mountDir,
		string(bins.Runtime),
		string(bins.Shim),
		absImageTar,
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
