package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/mholt/archiver/v4"
)

var downloadLocks sync.Map

func cacheDir() string {
	return filepath.Join(repo.MustFindRootDir(), ".downloads")
}

func RemoteprocSimulator(ctx context.Context, version, goarch string) (string, error) {
	extractDir := filepath.Join(cacheDir(), goarch, version)
	executablePath := filepath.Join(extractDir, "remoteproc-simulator")

	if fileExists(executablePath) {
		return executablePath, nil
	}

	lockKey := executablePath
	mu, _ := downloadLocks.LoadOrStore(lockKey, &sync.Mutex{})
	lock := mu.(*sync.Mutex)

	lock.Lock()
	defer lock.Unlock()

	if fileExists(executablePath) {
		return executablePath, nil
	}

	fmt.Printf("Downloading remoteproc-simulator %s for %s...\n", version, goarch)

	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	versionWithoutV := strings.TrimPrefix(version, "v")
	assetName := fmt.Sprintf("remoteproc-simulator_%s_linux_%s.tar.gz", versionWithoutV, goarch)
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

	handler := func(ctx context.Context, file archiver.FileInfo) error {
		if file.IsDir() {
			return nil
		}

		target := filepath.Join(extractDir, file.NameInArchive)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()

		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer func() { _ = rc.Close() }()

		_, err = io.Copy(out, rc)
		return err
	}

	gz := archiver.Gz{}
	gzReader, err := gz.OpenReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gzReader.Close() }()

	return archiver.Tar{}.Extract(context.Background(), gzReader, handler)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
