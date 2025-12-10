package limavm

import (
	"github.com/arm/remoteproc-runtime/e2e/limavm/scripts"
)

type Docker struct {
	VM
}

func NewDocker() (Docker, error) {
	vm, err := newVM("docker")
	if err != nil {
		return Docker{}, err
	}
	return Docker{VM: vm}, nil
}

func (vm Docker) BuildImage(buildContext string, imageName string) error {
	return scripts.BuildImage(vm.name, "docker", buildContext, imageName)
}
