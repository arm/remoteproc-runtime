package remoteproc

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/runner"
)

type Simulator struct {
	installedVM limavm.InstalledBin
	cmd         *runner.StreamingCmd
	name        string
	index       uint
	rootDir     string
}

func NewSimulator(installedVM limavm.InstalledBin, rootDir string) *Simulator {
	return &Simulator{
		installedVM: installedVM,
		rootDir:     rootDir,
		index:       0,
		name:        "some-cpu",
	}
}

func (r *Simulator) WithName(name string) *Simulator {
	r.name = name
	return r
}

func (r *Simulator) WithIndex(index uint) *Simulator {
	r.index = index
	return r
}

func (r *Simulator) Start() error {
	cmd := r.installedVM.Command(
		"--root-dir", r.rootDir,
		"--index", fmt.Sprintf("%d", r.index),
		"--name", r.name,
	)
	streamer := runner.NewStreamingCmd(cmd).WithPrefix("simulator: " + r.name + ": ")
	reader, writer := io.Pipe()
	if err := streamer.Start(writer); err != nil {
		return fmt.Errorf("failed to start simulator: %w", err)
	}
	r.cmd = streamer

	if err := r.waitForBoot(15*time.Second, reader); err != nil {
		_ = r.Stop()
		return fmt.Errorf("simulator failed to create remoteproc device: %w", err)
	}

	return nil
}

func (r *Simulator) waitForBoot(waitingTime time.Duration, outputBuf *io.PipeReader) error {
	deadline := time.Now().Add(waitingTime)
	scanner := bufio.NewScanner(outputBuf)
	for scanner.Scan() {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for simulator to create remoteproc device")
		}
		line := scanner.Text()
		if strings.Contains(line, "Remoteproc initialized at") {
			err := outputBuf.Close()
			if err != nil {
				return fmt.Errorf("failed to close output buffer: %w", err)
			}
			return nil
		}
	}
	panic("unreachable")
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
