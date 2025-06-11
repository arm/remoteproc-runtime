package adapter

import (
	"context"
	"fmt"
	"os"

	taskAPI "github.com/containerd/containerd/api/runtime/task/v2"
	ptypes "github.com/containerd/containerd/v2/pkg/protobuf/types"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/containerd/v2/pkg/shutdown"
	"github.com/containerd/containerd/v2/plugins"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/plugin"
	"github.com/containerd/plugin/registry"
	"github.com/containerd/ttrpc"
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
	return &exampleTaskService{}, nil
}

var (
	_ = shim.TTRPCService(&exampleTaskService{})
)

type exampleTaskService struct {
}

// RegisterTTRPC allows TTRPC services to be registered with the underlying server
func (s *exampleTaskService) RegisterTTRPC(server *ttrpc.Server) error {
	taskAPI.RegisterTaskService(server, s)
	return nil
}

// Create a new container
func (s *exampleTaskService) Create(ctx context.Context, r *taskAPI.CreateTaskRequest) (*taskAPI.CreateTaskResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Create")
	pid, err := CreateContainer(r)
	if err != nil {
		return nil, err
	}

	// TODO: publish event
	return &taskAPI.CreateTaskResponse{
		Pid: pid,
	}, nil
}

// Start the primary user process inside the container
func (s *exampleTaskService) Start(ctx context.Context, r *taskAPI.StartRequest) (*taskAPI.StartResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Start")
	return nil, errdefs.ErrNotImplemented
}

// Delete a process or container
func (s *exampleTaskService) Delete(ctx context.Context, r *taskAPI.DeleteRequest) (*taskAPI.DeleteResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Delete")
	return nil, errdefs.ErrNotImplemented
}

// Exec an additional process inside the container
func (s *exampleTaskService) Exec(ctx context.Context, r *taskAPI.ExecProcessRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Exec")
	return nil, errdefs.ErrNotImplemented
}

// ResizePty of a process
func (s *exampleTaskService) ResizePty(ctx context.Context, r *taskAPI.ResizePtyRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.ResizePty")
	return nil, errdefs.ErrNotImplemented
}

// State returns runtime state of a process
func (s *exampleTaskService) State(ctx context.Context, r *taskAPI.StateRequest) (*taskAPI.StateResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.State")
	return nil, errdefs.ErrNotImplemented
}

// Pause the container
func (s *exampleTaskService) Pause(ctx context.Context, r *taskAPI.PauseRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Pause")
	return nil, errdefs.ErrNotImplemented
}

// Resume the container
func (s *exampleTaskService) Resume(ctx context.Context, r *taskAPI.ResumeRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Resume")
	return nil, errdefs.ErrNotImplemented
}

// Kill a process
func (s *exampleTaskService) Kill(ctx context.Context, r *taskAPI.KillRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Kill")
	return nil, errdefs.ErrNotImplemented
}

// Pids returns all pids inside the container
func (s *exampleTaskService) Pids(ctx context.Context, r *taskAPI.PidsRequest) (*taskAPI.PidsResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Pids")
	return nil, errdefs.ErrNotImplemented
}

// CloseIO of a process
func (s *exampleTaskService) CloseIO(ctx context.Context, r *taskAPI.CloseIORequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.CloseIO")
	return nil, errdefs.ErrNotImplemented
}

// Checkpoint the container
func (s *exampleTaskService) Checkpoint(ctx context.Context, r *taskAPI.CheckpointTaskRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Checkpoint")
	return nil, errdefs.ErrNotImplemented
}

// Connect returns shim information of the underlying service
func (s *exampleTaskService) Connect(ctx context.Context, r *taskAPI.ConnectRequest) (*taskAPI.ConnectResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Connect")
	return nil, errdefs.ErrNotImplemented
}

// Shutdown is called after the underlying resources of the shim are cleaned up and the service can be stopped
func (s *exampleTaskService) Shutdown(ctx context.Context, r *taskAPI.ShutdownRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Shutdown")
	os.Exit(0)
	return &ptypes.Empty{}, nil
}

// Stats returns container level system stats for a container and its processes
func (s *exampleTaskService) Stats(ctx context.Context, r *taskAPI.StatsRequest) (*taskAPI.StatsResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Stats")
	return nil, errdefs.ErrNotImplemented
}

// Update the live container
func (s *exampleTaskService) Update(ctx context.Context, r *taskAPI.UpdateTaskRequest) (*ptypes.Empty, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Update")
	return nil, errdefs.ErrNotImplemented
}

// Wait for a process to exit
func (s *exampleTaskService) Wait(ctx context.Context, r *taskAPI.WaitRequest) (*taskAPI.WaitResponse, error) {
	log.L.WithField("request", fmt.Sprintf("%#v", r)).Info("service.Wait")
	return nil, errdefs.ErrNotImplemented
}
