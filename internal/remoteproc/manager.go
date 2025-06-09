package remoteproc

import (
	"context"
	"errors"
	"fmt"
	"io"

	apitypes "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
)

func NewManager(name string) shim.Manager {
	return manager{name: name}
}

type manager struct {
	name string
}

func (m manager) Name() string {
	return m.name
}

func (m manager) Start(ctx context.Context, id string, opts shim.StartOpts) (shim.BootstrapParams, error) {
	var params shim.BootstrapParams
	params.Version = 2
	params.Protocol = "ttrpc"

	socket, err := newShimSocket(ctx, opts.Address, id)
	if err != nil {
		if errors.Is(err, errSocketAlreadyExists) {
			params.Address = socket.addr
			return params, nil
		}
		return params, fmt.Errorf("failed to create socket: %w", err)
	}

	var retErr error
	defer func() {
		if retErr != nil {
			socket.Close()
		}
	}()

	cmd, retErr := newCommand(ctx, opts.Address, id)
	if retErr != nil {
		return params, fmt.Errorf("failed to create command: %w", err)
	}

	// ⚠️ Shim framework expects socket attached as file descriptor 3.
	cmd.ExtraFiles = append(cmd.ExtraFiles, socket.file)

	retErr = cmd.Start()
	if retErr != nil {
		return params, fmt.Errorf("failed to daemonise shim: %w", err)
	}

	params.Address = socket.addr
	return params, nil
}

func (m manager) Stop(ctx context.Context, id string) (shim.StopStatus, error) {
	return shim.StopStatus{}, errdefs.ErrNotImplemented
}

func (m manager) Info(ctx context.Context, optionsR io.Reader) (*apitypes.RuntimeInfo, error) {
	info := &apitypes.RuntimeInfo{
		Name: "io.containerd.example.v1",
		Version: &apitypes.RuntimeVersion{
			Version: "v1.0.0",
		},
	}
	return info, nil
}
