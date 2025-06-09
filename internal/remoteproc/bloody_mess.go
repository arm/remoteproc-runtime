package remoteproc

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/shim"
)

func newCommand(ctx context.Context, containerdAddress, id string) (*exec.Cmd, error) {
	namespace, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	args := []string{
		"-namespace", namespace,
		"-id", id,
		"-address", containerdAddress,
		"-publish-binary", self,
	}
	cmd := exec.Command(self, args...)
	cmd.Dir = cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return cmd, nil
}

type shimSocket struct {
	addr   string
	socket *net.UnixListener
	file   *os.File
}

func (s *shimSocket) Close() {
	if s.socket != nil {
		s.socket.Close()
	}
	if s.file != nil {
		s.file.Close()
	}
	_ = shim.RemoveSocket(s.addr)
}

var errSocketAlreadyExists = errors.New("socket aready exists")

func newShimSocket(ctx context.Context, path, id string) (*shimSocket, error) {
	address, err := shim.SocketAddress(ctx, path, id, false)
	if err != nil {
		return nil, err
	}
	socket, err := shim.NewSocket(address)
	if err != nil {
		return &shimSocket{addr: address}, errSocketAlreadyExists
	}
	s := &shimSocket{
		addr:   address,
		socket: socket,
	}
	file, err := socket.File()
	if err != nil {
		s.Close()
		return nil, err
	}
	s.file = file
	return s, nil
}
