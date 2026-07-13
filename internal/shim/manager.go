package shim

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/arm/remoteproc-runtime/internal/version"
	bootapi "github.com/containerd/containerd/api/runtime/bootstrap/v1"
	apitypes "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/defaults"
	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
)

func NewManager(name string) containerdshim.Shim {
	return manager{name: name}
}

type manager struct {
	name string
}

func (m manager) Name() string {
	return m.name
}

func (m manager) Start(ctx context.Context, opts *bootapi.BootstrapParams) (_ *bootapi.BootstrapResult, retErr error) {
	params := &bootapi.BootstrapResult{
		Version:  2,
		Protocol: "ttrpc",
	}

	id := opts.GetInstanceID()
	containerdAddress := opts.GetContainerdGrpcAddress()
	socketDir := opts.GetSocketDir()
	if socketDir == "" {
		socketDir = filepath.Join(defaults.DefaultStateDir, "s")
	}

	socket, err := newShimSocket(ctx, socketDir, containerdAddress, id)
	if err != nil {
		if errors.Is(err, errSocketAlreadyExists) {
			params.Address = socket.addr
			return params, nil
		}
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	defer func() {
		if retErr != nil {
			socket.Close()
		}
	}()

	debug := opts.GetLogLevel() <= bootapi.LogLevel_LOG_LEVEL_DEBUG
	cmd, err := newCommand(ctx, containerdAddress, opts.GetContainerdTtrpcAddress(), id, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create command: %w", err)
	}

	// ⚠️ Shim framework expects socket attached as file descriptor 3.
	cmd.ExtraFiles = append(cmd.ExtraFiles, socket.file)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to daemonise shim: %w", err)
	}

	params.Address = socket.addr
	return params, nil
}

func (m manager) Stop(ctx context.Context, id string) (containerdshim.StopStatus, error) {
	return containerdshim.StopStatus{
		ExitedAt: time.Now(),
		Pid:      os.Getpid(),
	}, nil
}

func (m manager) Info(ctx context.Context, optionsR io.Reader) (*apitypes.RuntimeInfo, error) {
	info := &apitypes.RuntimeInfo{
		Name: "io.containerd.remoteproc.v1",
		Version: &apitypes.RuntimeVersion{
			Version:  version.Version,
			Revision: version.GitCommit,
		},
	}
	return info, nil
}
