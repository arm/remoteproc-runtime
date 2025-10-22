package limavm

import (
	"fmt"
	"strconv"
	"syscall"
)

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

func (d Debian) SendSignal(pid int, signal syscall.Signal) error {
	_, stderr, err := d.RunCommand("kill", fmt.Sprintf("-%d", signal), strconv.Itoa(pid))
	if err != nil {
		return fmt.Errorf("failed to send signal %s to process %d: %w\nstderr: %s", signal, pid, err, stderr)
	}
	return nil
}
