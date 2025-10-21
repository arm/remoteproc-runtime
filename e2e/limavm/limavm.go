package limavm

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/limavm/scripts"
)

type LimaVM struct {
	name string
}

var BinBuildEnv = map[string]string{
	"GOOS": "linux",
}

func newVM(template string, mountDir string) (LimaVM, error) {
	vmName, err := scripts.PrepareLimaVM(template, mountDir)
	return LimaVM{name: vmName}, err
}

func (vm LimaVM) InstallBin(binToInstall string) error {
	return scripts.InstallBin(vm.name, binToInstall)
}

func (vm LimaVM) Cleanup() {
	_ = scripts.TeardownLimaVM(vm.name)
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

func Require(t *testing.T) {
	if _, err := exec.LookPath("limactl"); err != nil {
		t.Skip("limactl not found. Install limavm: https://lima-vm.io/")
	}
}
