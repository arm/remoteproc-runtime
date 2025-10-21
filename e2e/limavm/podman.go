package limavm

import (
	"fmt"
	"os/exec"

	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/runner"
)

type Podman struct {
	LimaVM
}

func NewPodman(mountDir string, buildContext string, runtimeBin repo.RuntimeBin) (Podman, error) {
	vm, err := newVM("podman", mountDir)
	if err != nil {
		return Podman{}, err
	}

	p := Podman{LimaVM: vm}

	if err := p.InstallBin(string(runtimeBin)); err != nil {
		p.Cleanup()
		return Podman{}, err
	}

	if err := p.BuildImage(buildContext, "test-image"); err != nil {
		p.Cleanup()
		return Podman{}, err
	}

	return p, nil
}

func (p Podman) BuildImage(buildContext string, imageName string) error {
	buildCmd := exec.Command(buildImageScript, p.name, "podman", buildContext, imageName)
	buildStreamer := runner.NewStreamingCmd(buildCmd).WithPrefix("build-image")

	if err := buildStreamer.Start(); err != nil {
		return fmt.Errorf("failed to start build-image script: %w", err)
	}

	if err := buildStreamer.Wait(); err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}
