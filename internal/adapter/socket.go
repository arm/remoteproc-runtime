package adapter

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/containerd/containerd/v2/pkg/shim"
)

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
