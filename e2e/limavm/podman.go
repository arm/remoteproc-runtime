package limavm

import (
	"github.com/arm/remoteproc-runtime/e2e/limavm/scripts"
	"github.com/arm/remoteproc-runtime/e2e/repo"
)

type Podman struct {
	VM
}

func NewPodman(mountDir string, buildContext string, runtimeBin repo.RuntimeBin) (Podman, error) {
	vm, err := newVM("podman", mountDir)
	if err != nil {
		return Podman{}, err
	}

	p := Podman{VM: vm}

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

func (vm Podman) BuildImage(buildContext string, imageName string) error {
	return scripts.BuildImage(vm.name, "podman", buildContext, imageName)
}
