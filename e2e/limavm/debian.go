package limavm

type Debian struct {
	VM
}

func NewDebian(mountDir string) (Debian, error) {
	vm, err := newVM("debian", mountDir)
	if err != nil {
		return Debian{}, err
	}
	return Debian{VM: vm}, nil
}
