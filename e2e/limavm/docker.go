package limavm

import (
	"fmt"
	"os/exec"

	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/runner"
)

type Docker struct {
	LimaVM
}

func NewDocker(mountDir string, buildContext string, bins repo.Bins) (Docker, error) {
	vm, err := newVM("docker", mountDir)
	if err != nil {
		return Docker{}, err
	}

	d := Docker{LimaVM: vm}

	for _, bin := range []string{string(bins.Runtime), string(bins.Shim)} {
		if err := d.InstallBin(bin); err != nil {
			d.Cleanup()
			return Docker{}, err
		}
	}

	if err := d.BuildImage(buildContext, "test-image"); err != nil {
		d.Cleanup()
		return Docker{}, err
	}

	return d, nil
}

func (d Docker) BuildImage(buildContext string, imageName string) error {
	buildCmd := exec.Command(buildImageScript, d.name, "docker", buildContext, imageName)
	buildStreamer := runner.NewStreamingCmd(buildCmd).WithPrefix("build-image")

	if err := buildStreamer.Start(); err != nil {
		return fmt.Errorf("failed to start build-image script: %w", err)
	}

	if err := buildStreamer.Wait(); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}
