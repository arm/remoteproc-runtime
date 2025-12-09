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
	bin     limavm.InstalledBin
	cmd     *runner.StreamingCmd
	name    string
	index   uint
	rootDir string
}

func NewSimulator(bin limavm.InstalledBin, rootDir string) *Simulator {
	return &Simulator{
		bin:     bin,
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
	cmd := r.bin.Command(
		"--root-dir", r.rootDir,
		"--index", fmt.Sprintf("%d", r.index),
		"--name", r.name,
	)
	reader, writer := io.Pipe()
	streamer := runner.NewStreamingCmd(cmd).WithPrefix("simulator: " + r.name + ": ").WithAdditionalOutput(writer)
	if err := streamer.Start(); err != nil {
		return fmt.Errorf("failed to start simulator: %w", err)
	}
	r.cmd = streamer

	if err := r.waitForBoot(15*time.Second, reader); err != nil {
		stopError := r.Stop()
		if stopError != nil {
			return fmt.Errorf("simulator failed to create remoteproc device: %w: %s", err, stopError)
		}
		return fmt.Errorf("simulator failed to create remoteproc device: %w", err)
	}

	return nil
}

func (r *Simulator) waitForBoot(waitingTime time.Duration, outputBuf *io.PipeReader) error {
	const targetMessage = "Remoteproc initialized at"

	deadline := time.Now().Add(waitingTime)
	scanner := bufio.NewScanner(outputBuf)
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	scanCh := make(chan string)
	errCh := make(chan error)

	go func() {
		for scanner.Scan() {
			scanCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			close(scanCh)
		}
	}()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout waiting for simulator to create remoteproc device")
		case line, ok := <-scanCh:
			if !ok {
				return fmt.Errorf("simulator output closed before remoteproc device was created")
			}
			if strings.Contains(line, targetMessage) {
				_ = outputBuf.Close()
				return nil
			}
		case err := <-errCh:
			return fmt.Errorf("scanner error: %w", err)
		}
	}
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
