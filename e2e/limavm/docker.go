package limavm

import (
	"github.com/arm/remoteproc-runtime/e2e/limavm/scripts"
	"github.com/arm/remoteproc-runtime/e2e/repo"
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

func (vm Docker) BuildImage(buildContext string, imageName string) error {
	return scripts.BuildImage(vm.name, "docker", buildContext, imageName)
}
