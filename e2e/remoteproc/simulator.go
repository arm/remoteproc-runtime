package remoteproc

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/runner"
)

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

var downloadLocks sync.Map

func DownloadSimulator(ctx context.Context) (string, error) {
	const version = "v0.0.8"
	arch := runtime.GOARCH

	cacheDir := filepath.Join(repo.MustFindRootDir(), ".downloads")
	extractDir := filepath.Join(cacheDir, arch, version)
	executablePath := filepath.Join(extractDir, "remoteproc-simulator")

	if fileExists(executablePath) {
		return executablePath, nil
	}

	lockKey := executablePath
	mu, _ := downloadLocks.LoadOrStore(lockKey, &sync.Mutex{})
	lock := mu.(*sync.Mutex)

	lock.Lock()
	defer lock.Unlock()

	fmt.Printf("Downloading remoteproc-simulator %s for %s...\n", version, arch)

	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	versionWithoutV := strings.TrimPrefix(version, "v")
	assetName := fmt.Sprintf("remoteproc-simulator_%s_linux_%s.tar.gz", versionWithoutV, arch)
	assetURL := fmt.Sprintf("https://github.com/arm/remoteproc-simulator/releases/download/%s/%s", version, assetName)
	archivePath := filepath.Join(extractDir, assetName)

	if err := downloadFile(ctx, assetURL, archivePath); err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	if err := extractTarGz(archivePath, extractDir); err != nil {
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}

	_ = os.Remove(archivePath)

	if err := os.Chmod(executablePath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make file executable: %w", err)
	}

	return executablePath, nil
}

func downloadFile(ctx context.Context, url, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download returned status %d: %s", resp.StatusCode, string(body))
	}

	tmpPath := targetPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, targetPath)
}

func extractTarGz(archivePath, extractDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeDir {
			continue
		}

		target := filepath.Join(extractDir, header.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		_ = out.Close()
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
