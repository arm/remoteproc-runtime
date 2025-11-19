package remoteproc

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/runner"
	"github.com/fsnotify/fsnotify"
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

func (r *Simulator) Start() error {
	cmd := r.installedVM.Command(
		"--root-dir", r.rootDir,
		"--index", fmt.Sprintf("%d", r.index),
		"--name", r.name,
	)
	streamer := runner.NewStreamingCmd(cmd).WithPrefix("simulator")
	if err := streamer.Start(); err != nil {
		return fmt.Errorf("failed to start simulator: %w", err)
	}
	r.cmd = streamer

	deviceDir := r.DeviceDir()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher for remoteproc directory: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(r.rootDir); err != nil {
		return fmt.Errorf("failed to watch remoteproc device directory %q: %w", deviceDir, err)
	}

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	// Wait for the remoteproc device directory to appear before returning.
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("remoteproc parent directory watcher closed")
			}
			fmt.Printf("Received fsnotify event: %s\n", event.Name)
			rel, err := filepath.Rel(event.Name, deviceDir)
			if err == nil && rel != "" && rel != "." {
				info, err := os.Stat(deviceDir)
				if err == nil && info.IsDir() {
					return nil
				}
				if err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat remoteproc directory %q: %w", deviceDir, err)
				}
			}
		case err := <-watcher.Errors:
			return fmt.Errorf("remoteproc directory watcher error: %w", err)
		case <-timer.C:
			return fmt.Errorf("remoteproc directory %q not created within 15s", deviceDir)
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
