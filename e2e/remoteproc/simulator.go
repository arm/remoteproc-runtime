package remoteproc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/download"
	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/runner"
)

func DownloadSimulator(ctx context.Context) (string, error) {
	const version = "v0.0.8"
	const goos = "linux"   // we only use simulator in linux VM
	arch := runtime.GOARCH // vm inherits the host's arch
	return download.GithubRelease(ctx, "arm", "remoteproc-simulator", version, goos, arch)
}

type Simulator struct {
	vm      limavm.VM
	bin     limavm.InstalledBin
	cmd     *runner.StreamingCmd
	name    string
	index   uint
	rootDir string
}

func NewSimulator(bin limavm.InstalledBin, vm limavm.VM, rootDir string) *Simulator {
	return &Simulator{
		vm:      vm,
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
	var killErr error
	_, stderr, err := r.vm.RunCommand("pkill", "-f", "remoteproc-simulator")
	if err != nil {
		killErr = fmt.Errorf("pkill failed: %w: stderr: %s", err, stderr)
	}

	if r.cmd != nil {
		if err := r.cmd.Stop(); err != nil {
			if killErr != nil {
				return fmt.Errorf("multiple errors: %v; %v", killErr, err)
			}
			return err
		}
	}

	return killErr
}

func (r *Simulator) DeviceDir() string {
	return filepath.Join(
		r.rootDir, "sys", "class", "remoteproc", fmt.Sprintf("remoteproc%d", r.index),
	)
}
