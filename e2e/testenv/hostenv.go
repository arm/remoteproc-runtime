package testenv

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type hostEnv struct{}

func (e *hostEnv) RunCommand(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("cmd failed: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), nil
}

func (e *hostEnv) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func (e *hostEnv) InstallBin(binPath string) (InstalledBin, error) {
	name := filepath.Base(binPath)
	dst := filepath.Join("/usr/local/bin", name)
	if out, err := exec.Command("sudo", "cp", binPath, dst).CombinedOutput(); err != nil {
		return InstalledBin{}, fmt.Errorf("install %s: %w: %s", name, err, out)
	}
	if out, err := exec.Command("sudo", "chmod", "+x", dst).CombinedOutput(); err != nil {
		return InstalledBin{}, fmt.Errorf("chmod %s: %w: %s", name, err, out)
	}
	return InstalledBin{env: e, pathToBin: dst}, nil
}

func (e *hostEnv) CopyDir(src, dst string) error {
	if out, err := exec.Command("cp", "-r", src, dst).CombinedOutput(); err != nil {
		return fmt.Errorf("copy %s to %s: %w: %s", src, dst, err, out)
	}
	return nil
}

func (e *hostEnv) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (e *hostEnv) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func (e *hostEnv) ReadDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

func (e *hostEnv) BuildImage(engine, contextDir, imageName string) error {
	cmd := exec.Command(engine, "build", "-t", imageName, contextDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build image: %w: %s", err, out)
	}
	return nil
}

func (e *hostEnv) Cleanup() {}
