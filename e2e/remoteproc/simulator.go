package remoteproc

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/runner"
)

type Simulator struct {
	cmd     *runner.StreamingCmd
	name    string
	index   uint
	rootDir string
}

func NewSimulator(rootDir string) *Simulator {
	return &Simulator{
		rootDir: rootDir,
		index:   0,
		name:    "some-cpu",
	}
}

func (r *Simulator) WithName(name string) *Simulator {
	r.name = name
	return r
}

func (r *Simulator) Start() error {
	cmd := exec.Command(
		"remoteproc-simulator",
		"--root-dir", r.rootDir,
		"--index", fmt.Sprintf("%d", r.index),
		"--name", r.name,
	)
	streamer := runner.NewStreamingCmd(cmd).WithPrefix("remoteproc-simulator")
	if err := streamer.Start(); err != nil {
		return fmt.Errorf("failed to start simulator: %w", err)
	}
	r.cmd = streamer
	return nil
}

func (r *Simulator) Stop() error {
	if r.cmd != nil {
		return r.cmd.Stop()
	}
	return nil
}

func (r *Simulator) DeviceDir() string {
	return filepath.Join(
		r.rootDir, "sys", "class", "remoteproc", fmt.Sprintf("remoteproc%d", r.index),
	)
}
