package scripts

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

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

func PrepareLimaVM(template string, mountDir string) (string, error) {
	prepareCmd := exec.Command(prepareLimaVMScript, template, mountDir)
	prepareStreamer := runner.NewStreamingCmd(prepareCmd).WithPrefix("prepare-vm")

	if err := prepareStreamer.Start(nil); err != nil {
		return "", fmt.Errorf("failed to start prepare-lima script: %w", err)
	}

	if err := prepareStreamer.Wait(); err != nil {
		return "", fmt.Errorf("failed to prepare VM: %w", err)
	}

	vmName := strings.TrimSpace(prepareStreamer.Output())
	if vmName == "" {
		return "", fmt.Errorf("prepare script did not return VM name")
	}

	return vmName, nil
}

func InstallBin(vmName string, binToInstall string) (string, error) {
	installCmd := exec.Command(installBinScript, vmName, binToInstall)
	installStreamer := runner.NewStreamingCmd(installCmd).WithPrefix("install-bin")

	if err := installStreamer.Start(nil); err != nil {
		return "", fmt.Errorf("failed to start install-bin script: %w", err)
	}

	if err := installStreamer.Wait(); err != nil {
		return "", fmt.Errorf("failed to install binary: %w", err)
	}

	installedBinLocation := strings.TrimSpace(installStreamer.Output())
	if installedBinLocation == "" {
		return "", fmt.Errorf("install script did not return installed binary location")
	}

	return installedBinLocation, nil
}

func TeardownLimaVM(vmName string) error {
	cmd := exec.Command(teardownLimaVMScript, vmName)
	return cmd.Run()
}

func BuildImage(vmName string, template string, buildContext string, imageName string) error {
	buildCmd := exec.Command(buildImageScript, vmName, template, buildContext, imageName)
	buildStreamer := runner.NewStreamingCmd(buildCmd).WithPrefix("build-image")

	if err := buildStreamer.Start(nil); err != nil {
		return fmt.Errorf("failed to start build-image script: %w", err)
	}

	if err := buildStreamer.Wait(); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}
