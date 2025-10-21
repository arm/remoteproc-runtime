package limavm

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/runner"
)

var (
	scriptsDir           = filepath.Join(repo.MustFindRootDir(), "e2e", "limavm", "scripts")
	prepareLimaVMScript  = filepath.Join(scriptsDir, "prepare-lima-vm.sh")
	installBinScript     = filepath.Join(scriptsDir, "install-bin.sh")
	buildImageScript     = filepath.Join(scriptsDir, "build-image.sh")
	teardownLimaVMScript = filepath.Join(scriptsDir, "teardown-lima-vm.sh")
)

type LimaVM struct {
	name     string
	template string
}

var BinBuildEnv = map[string]string{
	"GOOS": "linux",
}

func NewWithDocker(mountDir string, buildContext string, bins repo.Bins) (LimaVM, error) {
	vm, err := new("docker", mountDir)
	if err != nil {
		return LimaVM{}, err
	}
	for _, bin := range []string{string(bins.Runtime), string(bins.Shim)} {
		if err := vm.InstallBin(bin); err != nil {
			vm.Cleanup()
			return vm, err
		}
	}
	if err := vm.BuildImage(buildContext, "test-image"); err != nil {
		vm.Cleanup()
		return vm, err
	}
	return vm, nil
}

func NewWithPodman(mountDir string, buildContext string, runtimeBin repo.RuntimeBin) (LimaVM, error) {
	vm, err := new("podman", mountDir)
	if err != nil {
		return LimaVM{}, err
	}
	if err := vm.InstallBin(string(runtimeBin)); err != nil {
		vm.Cleanup()
		return vm, err
	}
	if err := vm.BuildImage(buildContext, "test-image"); err != nil {
		vm.Cleanup()
		return vm, err
	}
	return vm, nil
}

func new(template string, mountDir string) (LimaVM, error) {
	prepareCmd := exec.Command(prepareLimaVMScript, template, mountDir)
	prepareStreamer := runner.NewStreamingCmd(prepareCmd).WithPrefix("prepare-vm")

	if err := prepareStreamer.Start(); err != nil {
		return LimaVM{}, fmt.Errorf("failed to start prepare-lima script: %w", err)
	}

	if err := prepareStreamer.Wait(); err != nil {
		return LimaVM{}, fmt.Errorf("failed to prepare VM: %w", err)
	}

	vmName := strings.TrimSpace(prepareStreamer.Output())
	if vmName == "" {
		return LimaVM{}, fmt.Errorf("prepare script did not return VM name")
	}

	return LimaVM{name: vmName, template: template}, nil
}

func (vm LimaVM) InstallBin(binToInstall string) error {
	installCmd := exec.Command(installBinScript, vm.name, binToInstall)
	installStreamer := runner.NewStreamingCmd(installCmd).WithPrefix("install-bin")

	if err := installStreamer.Start(); err != nil {
		return fmt.Errorf("failed to start install-bin script: %w", err)
	}

	if err := installStreamer.Wait(); err != nil {
		return fmt.Errorf("failed to install binaries: %w", err)
	}

	return nil
}

func (vm LimaVM) BuildImage(buildContext string, imageName string) error {
	buildCmd := exec.Command(buildImageScript, vm.name, vm.template, buildContext, imageName)
	buildStreamer := runner.NewStreamingCmd(buildCmd).WithPrefix("build-image")

	if err := buildStreamer.Start(); err != nil {
		return fmt.Errorf("failed to start build-image script: %w", err)
	}

	if err := buildStreamer.Wait(); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
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

func Require(t *testing.T) {
	if _, err := exec.LookPath("limactl"); err != nil {
		t.Skip("limactl not found. Install limavm: https://lima-vm.io/")
	}
}
