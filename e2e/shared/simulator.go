package shared

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type RemoteprocSimulator struct {
	cmd     *StreamingCmd
	name    string
	index   uint
	rootDir string
}

func NewRemoteprocSimulator(rootDir string) *RemoteprocSimulator {
	return &RemoteprocSimulator{
		rootDir: rootDir,
		index:   0,
		name:    "some-cpu",
	}
}

func (r *RemoteprocSimulator) WithName(name string) *RemoteprocSimulator {
	r.name = name
	return r
}

func (r *RemoteprocSimulator) Start() error {
	cmd := exec.Command(
		"remoteproc-simulator",
		"--root-dir", r.rootDir,
		"--index", fmt.Sprintf("%d", r.index),
		"--name", r.name,
	)
	streamer := NewStreamingCmd(cmd).WithPrefix("remoteproc-simulator")
	if err := streamer.Start(); err != nil {
		return fmt.Errorf("failed to start simulator: %w", err)
	}
	r.cmd = streamer
	return nil
}

func (r *RemoteprocSimulator) Stop() error {
	if r.cmd != nil {
		return r.cmd.Stop()
	}
	return nil
}

func (r *RemoteprocSimulator) DeviceDir() string {
	return filepath.Join(
		r.rootDir, "sys", "class", "remoteproc", fmt.Sprintf("remoteproc%d", r.index),
	)
}
