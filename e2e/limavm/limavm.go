package limavm

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/limavm/scripts"
)

type VM struct {
	name string
}

var BinBuildEnv = map[string]string{
	"GOOS": "linux",
}

func newVM(template string, mountDir string) (VM, error) {
	vmName, err := scripts.PrepareLimaVM(template, mountDir)
	return VM{name: vmName}, err
}

func (vm VM) InstallBin(binToInstall string) (InstalledBin, error) {
	installPath, err := scripts.InstallBin(vm.name, binToInstall)
	if err != nil {
		return InstalledBin{}, err
	}
	return InstalledBin{vm: vm, pathToBin: installPath}, nil
}

func (vm VM) Cleanup() {
	_ = scripts.TeardownLimaVM(vm.name)
}

func (vm VM) cmd(name string, args ...string) *exec.Cmd {
	allArgs := append([]string{"shell", vm.name, name}, args...)
	return exec.Command("limactl", allArgs...)
}

func (vm VM) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
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

type Runnable interface {
	Run(args ...string) (stdout, stderr string, err error)
}

type InstalledBin struct {
	vm        VM
	pathToBin string
}

func (b InstalledBin) Run(args ...string) (stdout, stderr string, err error) {
	return b.vm.RunCommand(b.pathToBin, args...)
}

func (b InstalledBin) Command(args ...string) *exec.Cmd {
	return b.vm.cmd(b.pathToBin, args...)
}

func (b InstalledBin) Path() string {
	return b.pathToBin
}

type Sudo struct {
	vm        VM
	pathToBin string
}

func NewSudo(b InstalledBin) Sudo {
	return Sudo(b)
}

func (r Sudo) Run(args ...string) (stdout, stderr string, err error) {
	return r.vm.RunCommand("sudo", append([]string{r.pathToBin}, args...)...)
}

func Require(t *testing.T) {
	if _, err := exec.LookPath("limactl"); err != nil {
		t.Skip("limactl not found. Install limavm: https://lima-vm.io/")
	}
}
