package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/repo"
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

	os.Remove(archivePath)

	if err := os.Chmod(executablePath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make file executable: %w", err)
	}

	return executablePath, nil
}

func getReleaseAssetURL(ctx context.Context, owner, repoName, version, goos, goarch string) (string, string, error) {
	var assetName string
	if goos == "windows" {
		assetName = fmt.Sprintf("%s_%s_%s_%s.zip", repoName, version, goos, goarch)
	} else {
		assetName = fmt.Sprintf("%s_%s_%s_%s.tar.gz", repoName, version, goos, goarch)
	}

	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repoName, version)
	req, err := http.NewRequestWithContext(ctx, "GET", releaseURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

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

	return "", "", fmt.Errorf("asset %s not found in release %s", assetName, version)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download returned status %d: %s", resp.StatusCode, string(body))
	}

	tmpPath := targetPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, targetPath)
}

func extractArchive(archivePath, extractDir, goos string) error {
	if goos == "windows" {
		return extractZip(archivePath, extractDir)
	}
	return extractTarGz(archivePath, extractDir)
}

func extractTarGz(archivePath, extractDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(extractDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

func extractZip(archivePath, extractDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(extractDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
