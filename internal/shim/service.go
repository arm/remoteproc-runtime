package shim

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	eventstypes "github.com/containerd/containerd/api/events"
	taskAPI "github.com/containerd/containerd/api/runtime/task/v2"
	ttypes "github.com/containerd/containerd/api/types/task"
	"github.com/containerd/containerd/v2/core/mount"
	containerdRuntime "github.com/containerd/containerd/v2/core/runtime"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	ptypes "github.com/containerd/containerd/v2/pkg/protobuf/types"
	"github.com/containerd/log"
	"github.com/sirupsen/logrus"

	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
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
			return newTaskService(ic.Context, pp.(containerdshim.Publisher), ss.(shutdown.Service))
		},
	})
}

func newTaskService(ctx context.Context, publisher containerdshim.Publisher, sd shutdown.Service) (taskAPI.TaskService, error) {
	// The containerdshim.Publisher and shutdown.Service are usually useful for your task service,
	// but we don't need them in the exampleTaskService.
	service := &remoteprocTaskService{
		events:         make(chan any, 128),
		shutdown:       sd,
		logger:         log.G(ctx),
		processWatcher: nil,
	}

	sd.RegisterCallback(func(context.Context) error {
		close(service.events)
		return nil
	})

	go service.forward(ctx, publisher)

	return service, nil
}

var (
	_ = containerdshim.TTRPCService(&remoteprocTaskService{})
)

type remoteprocTaskService struct {
	events   chan any
	shutdown shutdown.Service
	logger   logrus.FieldLogger

	processWatcherMu sync.Mutex
	processWatcher   *ProcessWatcher
}

// RegisterTTRPC allows TTRPC services to be registered with the underlying server
func (s *remoteprocTaskService) RegisterTTRPC(server *ttrpc.Server) error {
	taskAPI.RegisterTaskService(server, s)
	return nil
}

// Create a new container
func (s *remoteprocTaskService) Create(ctx context.Context, r *taskAPI.CreateTaskRequest) (*taskAPI.CreateTaskResponse, error) {
	s.logPayload("-> service.Create", r)
	const shimRootFS = "rootfs" // Assumption copied from containerd runc.NewContainer
	rootFS := filepath.Join(r.Bundle, shimRootFS)
	toMount := listMounts(r)
	if err := mount.All(toMount, rootFS); err != nil {
		return nil, fmt.Errorf("failed to mount rootfs: %w", err)
	}
	err := executeCreate(r.ID, r.Bundle)
	if err != nil {
		if err := mount.UnmountMounts(toMount, rootFS, 0); err != nil {
			s.logger.WithError(err).Warn("failed to cleanup rootfs mount")
		}
		return nil, err
	}

	pid, err := getPid(r.ID)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to get PID, defaulting to %d", pid)
	}

	s.send(&eventstypes.TaskCreate{
		ContainerID: r.ID,
		Bundle:      r.Bundle,
		// Rootfs:      r.Rootfs,
		// IO:          &eventstypes.TaskIO{},
		// Checkpoint:  "",
		Pid: uint32(pid),
	})

	response := &taskAPI.CreateTaskResponse{Pid: uint32(pid)}
	s.logPayload("<- service.Create", response)
	return response, nil
}

func listMounts(req *taskAPI.CreateTaskRequest) []mount.Mount {
	mounts := make([]mount.Mount, len(req.Rootfs))
	for i, pm := range req.Rootfs {
		mounts[i] = mount.Mount{
			Type:    pm.Type,
			Source:  pm.Source,
			Target:  pm.Target,
			Options: pm.Options,
		}
	}
	return mounts
}

// Start the primary user process inside the container
func (s *remoteprocTaskService) Start(ctx context.Context, r *taskAPI.StartRequest) (*taskAPI.StartResponse, error) {
	s.logPayload("-> service.Start", r)
	err := executeStart(r.ID)
	if err != nil {
		return nil, err
	}

	pid, err := getPid(r.ID)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to get PID, defaulting to %d", pid)
	}

	if pid > 0 {
		s.startProcessWatcher(r.ID, pid)
	}

	s.send(&eventstypes.TaskStart{
		ContainerID: r.ID,
		Pid:         uint32(pid),
	})

	response := &taskAPI.StartResponse{Pid: uint32(pid)}
	s.logPayload("<- service.Start", response)
	return response, nil
}

// Delete a process or container
func (s *remoteprocTaskService) Delete(ctx context.Context, r *taskAPI.DeleteRequest) (*taskAPI.DeleteResponse, error) {
	s.logPayload("-> service.Delete", r)

	pid, err := getPid(r.ID)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to get PID, defaulting to %d", pid)
	}

	if err := executeDelete(r.ID); err != nil {
		return nil, err
	}

	s.send(&eventstypes.TaskDelete{
		ContainerID: r.ID,
		Pid:         uint32(pid),
		// ExitStatus: 0,
		// ExitedAt:   &timestamppb.Timestamp{},
		// ID:         "",
	})

	response := &taskAPI.DeleteResponse{}
	s.logPayload("<- service.Delete", response)
	return response, nil
}

// Exec an additional process inside the container
func (s *remoteprocTaskService) Exec(ctx context.Context, r *taskAPI.ExecProcessRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Exec", r)
	return nil, errdefs.ErrNotImplemented
}

// ResizePty of a process
func (s *remoteprocTaskService) ResizePty(ctx context.Context, r *taskAPI.ResizePtyRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.ResizePty", r)
	return nil, errdefs.ErrNotImplemented
}

// State returns runtime state of a process
func (s *remoteprocTaskService) State(ctx context.Context, r *taskAPI.StateRequest) (*taskAPI.StateResponse, error) {
	s.logPayload("-> service.State", r)
	state, err := executeState(r.ID)
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

	s.logPayload("<- service.State", response)
	return response, nil
}

// Pause the container
func (s *remoteprocTaskService) Pause(ctx context.Context, r *taskAPI.PauseRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Pause", r)
	return nil, errdefs.ErrNotImplemented
}

// Resume the container
func (s *remoteprocTaskService) Resume(ctx context.Context, r *taskAPI.ResumeRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Resume", r)
	return nil, errdefs.ErrNotImplemented
}

// Kill a process
func (s *remoteprocTaskService) Kill(ctx context.Context, r *taskAPI.KillRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Kill", r)

	s.stopProcessWatcher()

	pid, err := getPid(r.ID)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to get PID, defaulting to %d", pid)
	}

	var signal syscall.Signal
	switch r.Signal {
	case 9:
		signal = syscall.SIGKILL
	case 15:
		signal = syscall.SIGTERM
	default:
		signal = syscall.SIGTERM
	}

	err = executeKill(r.ID, signal)
	if err != nil {
		return nil, err
	}

	s.send(&eventstypes.TaskExit{
		ContainerID: r.ID,
		ID:          r.ID,
		Pid:         uint32(pid),
		// ExitStatus:  uint32(e.Status),
		// ExitedAt:    protobuf.ToTimestamp(p.ExitedAt()),
	})

	response := &ptypes.Empty{}
	s.logPayload("<- service.Kill", response)
	return response, nil
}

// Pids returns all pids inside the container
func (s *remoteprocTaskService) Pids(ctx context.Context, r *taskAPI.PidsRequest) (*taskAPI.PidsResponse, error) {
	s.logPayload("-> service.Pids", r)
	return nil, errdefs.ErrNotImplemented
}

// CloseIO of a process
func (s *remoteprocTaskService) CloseIO(ctx context.Context, r *taskAPI.CloseIORequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.CloseIO", r)
	return nil, errdefs.ErrNotImplemented
}

// Checkpoint the container
func (s *remoteprocTaskService) Checkpoint(ctx context.Context, r *taskAPI.CheckpointTaskRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Checkpoint", r)
	return nil, errdefs.ErrNotImplemented
}

// Connect returns shim information of the underlying service
func (s *remoteprocTaskService) Connect(ctx context.Context, r *taskAPI.ConnectRequest) (*taskAPI.ConnectResponse, error) {
	s.logPayload("-> service.Connect", r)
	pid, err := getPid(r.ID)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to get PID, defaulting to %d", pid)
	}
	response := &taskAPI.ConnectResponse{
		ShimPid: uint32(os.Getpid()),
		TaskPid: uint32(pid),
	}
	s.logPayload("<- service.Connect", response)
	return response, nil
}

// Shutdown is called after the underlying resources of the shim are cleaned up and the service can be stopped
func (s *remoteprocTaskService) Shutdown(ctx context.Context, r *taskAPI.ShutdownRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Shutdown", r)
	s.shutdown.Shutdown()
	<-s.shutdown.Done()
	os.Exit(0)
	return &ptypes.Empty{}, nil
}

// Stats returns container level system stats for a container and its processes
func (s *remoteprocTaskService) Stats(ctx context.Context, r *taskAPI.StatsRequest) (*taskAPI.StatsResponse, error) {
	s.logPayload("-> service.Stats", r)
	return nil, errdefs.ErrNotImplemented
}

// Update the live container
func (s *remoteprocTaskService) Update(ctx context.Context, r *taskAPI.UpdateTaskRequest) (*ptypes.Empty, error) {
	s.logPayload("-> service.Update", r)
	return nil, errdefs.ErrNotImplemented
}

// Wait for a process to exit
func (s *remoteprocTaskService) Wait(ctx context.Context, r *taskAPI.WaitRequest) (*taskAPI.WaitResponse, error) {
	s.logPayload("-> service.Wait", r)
	const interval = 1 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			state, err := executeState(r.ID)
			if err != nil {
				return nil, err
			}
			if state.Status == specs.StateStopped {
				response := &taskAPI.WaitResponse{ExitStatus: 0}
				s.logPayload("<- service.Wait", response)
				return response, nil
			}
		}
	}
}

func (s *remoteprocTaskService) send(event any) {
	s.events <- event
}

func (s *remoteprocTaskService) forward(ctx context.Context, publisher containerdshim.Publisher) {
	ns, _ := namespaces.Namespace(ctx)
	ctx = namespaces.WithNamespace(context.Background(), ns)
	for e := range s.events {
		err := publisher.Publish(ctx, containerdRuntime.GetTopic(e), e)
		if err != nil {
			s.logger.WithError(err).Error("post event")
		}
	}
	publisher.Close()
}

func (s *remoteprocTaskService) logPayload(name string, payload any) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithField("err", err).Debug(name)
	}
	s.logger.WithField("payload", string(payloadJSON)).Debug(name)
}

func (s *remoteprocTaskService) startProcessWatcher(containerID string, pid int) {
	watcher, err := NewProcessWatcher(pid)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to create process watcher for container %s, pid %d", containerID, pid)
		return
	}
	s.processWatcherMu.Lock()
	s.processWatcher = watcher
	s.processWatcherMu.Unlock()

	go func() {
		reason := watcher.WaitForExit()

		if reason == ProcessExited {
			s.send(&eventstypes.TaskExit{
				ContainerID: containerID,
				ID:          containerID,
				Pid:         uint32(pid),
			})

			s.shutdown.Shutdown()
			<-s.shutdown.Done()
		}
	}()
}

func (s *remoteprocTaskService) stopProcessWatcher() {
	s.processWatcherMu.Lock()
	defer s.processWatcherMu.Unlock()
	if s.processWatcher != nil {
		s.processWatcher.StopWatching()
		s.processWatcher = nil
	}
}
