package limavm

import (
	"github.com/arm/remoteproc-runtime/e2e/limavm/scripts"
)

type Podman struct {
	VM
}

func NewPodman(mountDir string) (Podman, error) {
	vm, err := newVM("podman", mountDir)
	if err != nil {
		return Podman{}, err
	}
	return Podman{VM: vm}, nil
}

func (vm Podman) BuildImage(buildContext string, imageName string) error {
	return scripts.BuildImage(vm.name, "podman", buildContext, imageName)
}
