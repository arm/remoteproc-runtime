package adapter

import (
	"context"
	"os"
	"time"

	"github.com/Arm-Debug/remoteproc-shim/internal/runtime"
	taskAPI "github.com/containerd/containerd/api/runtime/task/v2"
	ttypes "github.com/containerd/containerd/api/types/task"
	ptypes "github.com/containerd/containerd/v2/pkg/protobuf/types"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/containerd/v2/pkg/shutdown"
	"github.com/containerd/containerd/v2/plugins"
	"github.com/containerd/errdefs"
	"github.com/containerd/plugin"
	"github.com/containerd/plugin/registry"
	"github.com/containerd/ttrpc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func init() {
	registry.Register(&plugin.Registration{
		Type: plugins.TTRPCPlugin,
		ID:   "task",
		Requires: []plugin.Type{
			plugins.EventPlugin,
			plugins.InternalPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			pp, err := ic.GetByID(plugins.EventPlugin, "publisher")
			if err != nil {
				return nil, err
			}
			ss, err := ic.GetByID(plugins.InternalPlugin, "shutdown")
			if err != nil {
				return nil, err
			}
			return newTaskService(ic.Context, pp.(shim.Publisher), ss.(shutdown.Service))
		},
	})
}

func newTaskService(ctx context.Context, publisher shim.Publisher, sd shutdown.Service) (taskAPI.TaskService, error) {
	// The shim.Publisher and shutdown.Service are usually useful for your task service,
	// but we don't need them in the exampleTaskService.
	return &remoteprocTaskService{}, nil
}

var (
	_ = shim.TTRPCService(&remoteprocTaskService{})
)

type remoteprocTaskService struct {
}

// RegisterTTRPC allows TTRPC services to be registered with the underlying server
func (s *remoteprocTaskService) RegisterTTRPC(server *ttrpc.Server) error {
	taskAPI.RegisterTaskService(server, s)
	return nil
}

// Create a new container
func (s *remoteprocTaskService) Create(ctx context.Context, r *taskAPI.CreateTaskRequest) (*taskAPI.CreateTaskResponse, error) {
	// TODO: figure out ttrpc interceptors
	logRequest("service.Create", r)
	err := CreateContainer(r)
	if err != nil {
		return nil, err
	}

	// TODO: publish event
	response := &taskAPI.CreateTaskResponse{}
	logResponse("service.Create", response)
	return response, nil
}

// Start the primary user process inside the container
func (s *remoteprocTaskService) Start(ctx context.Context, r *taskAPI.StartRequest) (*taskAPI.StartResponse, error) {
	logRequest("service.Start", r)
	err := runtime.Start(r.ID)
	if err != nil {
		return nil, err
	}
	response := &taskAPI.StartResponse{}
	logResponse("service.Start", response)
	return response, nil
}

// Delete a process or container
func (s *remoteprocTaskService) Delete(ctx context.Context, r *taskAPI.DeleteRequest) (*taskAPI.DeleteResponse, error) {
	logRequest("service.Delete", r)
	if err := runtime.Delete(r.ID); err != nil {
		return nil, err
	}
	response := &taskAPI.DeleteResponse{}
	logResponse("service.Delete", response)
	return response, nil
}

// Exec an additional process inside the container
func (s *remoteprocTaskService) Exec(ctx context.Context, r *taskAPI.ExecProcessRequest) (*ptypes.Empty, error) {
	logRequest("service.Exec", r)
	return nil, errdefs.ErrNotImplemented
}

// ResizePty of a process
func (s *remoteprocTaskService) ResizePty(ctx context.Context, r *taskAPI.ResizePtyRequest) (*ptypes.Empty, error) {
	logRequest("service.ResizePty", r)
	return nil, errdefs.ErrNotImplemented
}

// State returns runtime state of a process
func (s *remoteprocTaskService) State(ctx context.Context, r *taskAPI.StateRequest) (*taskAPI.StateResponse, error) {
	logRequest("service.State", r)
	state, err := runtime.State(r.ID)
	if err != nil {
		return nil, err
	}

	response := &taskAPI.StateResponse{
		ID:     state.ID,
		Bundle: state.Bundle,
		Pid:    uint32(state.Pid),
	}

	switch state.Status {
	case specs.StateCreated:
		response.Status = ttypes.Status_CREATED
	case specs.StateCreating:
		response.Status = ttypes.Status_CREATED
	case specs.StateRunning:
		response.Status = ttypes.Status_RUNNING
	case specs.StateStopped:
		response.Status = ttypes.Status_STOPPED
	default:
		response.Status = ttypes.Status_UNKNOWN
	}

	logResponse("service.State", response)
	return response, nil
}

// Pause the container
func (s *remoteprocTaskService) Pause(ctx context.Context, r *taskAPI.PauseRequest) (*ptypes.Empty, error) {
	logRequest("service.Pause", r)
	return nil, errdefs.ErrNotImplemented
}

// Resume the container
func (s *remoteprocTaskService) Resume(ctx context.Context, r *taskAPI.ResumeRequest) (*ptypes.Empty, error) {
	logRequest("service.Resume", r)
	return nil, errdefs.ErrNotImplemented
}

// Kill a process
func (s *remoteprocTaskService) Kill(ctx context.Context, r *taskAPI.KillRequest) (*ptypes.Empty, error) {
	logRequest("service.Kill", r)
	err := runtime.Kill(r.ID)
	if err != nil {
		return nil, err
	}
	response := &ptypes.Empty{}
	logResponse("service.Kill", response)
	return response, nil
}

// Pids returns all pids inside the container
func (s *remoteprocTaskService) Pids(ctx context.Context, r *taskAPI.PidsRequest) (*taskAPI.PidsResponse, error) {
	logRequest("service.Pids", r)
	return nil, errdefs.ErrNotImplemented
}

// CloseIO of a process
func (s *remoteprocTaskService) CloseIO(ctx context.Context, r *taskAPI.CloseIORequest) (*ptypes.Empty, error) {
	logRequest("service.CloseIO", r)
	return nil, errdefs.ErrNotImplemented
}

// Checkpoint the container
func (s *remoteprocTaskService) Checkpoint(ctx context.Context, r *taskAPI.CheckpointTaskRequest) (*ptypes.Empty, error) {
	logRequest("service.Checkpoint", r)
	return nil, errdefs.ErrNotImplemented
}

// Connect returns shim information of the underlying service
func (s *remoteprocTaskService) Connect(ctx context.Context, r *taskAPI.ConnectRequest) (*taskAPI.ConnectResponse, error) {
	logRequest("service.Connect", r)
	response := &taskAPI.ConnectResponse{
		ShimPid: uint32(os.Getpid()),
	}
	logResponse("service.Connect", response)
	return response, nil
}

// Shutdown is called after the underlying resources of the shim are cleaned up and the service can be stopped
func (s *remoteprocTaskService) Shutdown(ctx context.Context, r *taskAPI.ShutdownRequest) (*ptypes.Empty, error) {
	logRequest("service.Shutdown", r)
	os.Exit(0)
	return &ptypes.Empty{}, nil
}

// Stats returns container level system stats for a container and its processes
func (s *remoteprocTaskService) Stats(ctx context.Context, r *taskAPI.StatsRequest) (*taskAPI.StatsResponse, error) {
	logRequest("service.Stats", r)
	return nil, errdefs.ErrNotImplemented
}

// Update the live container
func (s *remoteprocTaskService) Update(ctx context.Context, r *taskAPI.UpdateTaskRequest) (*ptypes.Empty, error) {
	logRequest("service.Update", r)
	return nil, errdefs.ErrNotImplemented
}

// Wait for a process to exit
func (s *remoteprocTaskService) Wait(ctx context.Context, r *taskAPI.WaitRequest) (*taskAPI.WaitResponse, error) {
	logRequest("service.Wait", r)
	const interval = 1 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			state, err := runtime.State(r.ID)
			if err != nil {
				return nil, err
			}
			if state.Status == specs.StateStopped {
				response := &taskAPI.WaitResponse{ExitStatus: 0}
				logResponse("service.Wait", response)
				return response, nil
			}
		}
	}
}
