package limavm

type Debian struct {
	VM
}

func NewDebian() (Debian, error) {
	vm, err := newVM("debian")
	if err != nil {
		return Debian{}, err
	}
	return Debian{VM: vm}, nil
}
