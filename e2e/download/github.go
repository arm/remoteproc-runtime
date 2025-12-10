package download

import (
	"context"
	"encoding/json"
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

func GithubRelease(ctx context.Context, owner, repoName, version, goos, goarch string) (string, error) {
	executableName := repoName
	if goos == "windows" {
		executableName += ".exe"
	}

	extractDir := filepath.Join(cacheDir(), repoName, goos, goarch, version)
	executablePath := filepath.Join(extractDir, executableName)

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

	fmt.Printf("Downloading %s/%s %s for %s/%s...\n", owner, repoName, version, goos, goarch)

	assetURL, assetName, err := getReleaseAssetURL(ctx, owner, repoName, version, goos, goarch)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	archivePath := filepath.Join(extractDir, assetName)
	if err := downloadFile(ctx, assetURL, archivePath); err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	if err := extractArchive(archivePath, extractDir, goos); err != nil {
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}

	_ = os.Remove(archivePath)

	if err := os.Chmod(executablePath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make file executable: %w", err)
	}

	return executablePath, nil
}

func getReleaseAssetURL(ctx context.Context, owner, repoName, version, goos, goarch string) (string, string, error) {
	versionWithoutV := strings.TrimPrefix(version, "v")

	var assetName string
	if goos == "windows" {
		assetName = fmt.Sprintf("%s_%s_%s_%s.zip", repoName, versionWithoutV, goos, goarch)
	} else {
		assetName = fmt.Sprintf("%s_%s_%s_%s.tar.gz", repoName, versionWithoutV, goos, goarch)
	}

	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repoName, version)
	req, err := http.NewRequestWithContext(ctx, "GET", releaseURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return asset.BrowserDownloadURL, assetName, nil
		}
	}

	availableAssets := make([]string, len(release.Assets))
	for i, asset := range release.Assets {
		availableAssets[i] = asset.Name
	}

	return "", "", fmt.Errorf("asset %s not found in release %s, available assets: %v", assetName, version, availableAssets)
}

func downloadFile(ctx context.Context, url, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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

func extractArchive(archivePath, extractDir, goos string) error {
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

	if goos == "windows" {
		return archiver.Zip{}.Extract(context.Background(), f, handler)
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
