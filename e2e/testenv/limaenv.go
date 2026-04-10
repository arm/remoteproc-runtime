package testenv

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const limaVMName = "remoteproc-e2e"

type limaEnv struct {
	vmName string
}

var (
	sharedLimaEnv     *limaEnv
	sharedLimaEnvOnce sync.Once
	sharedLimaEnvErr  error
)

func getOrCreateLimaEnv() (*limaEnv, error) {
	sharedLimaEnvOnce.Do(func() {
		sharedLimaEnv, sharedLimaEnvErr = newLimaEnv()
	})
	return sharedLimaEnv, sharedLimaEnvErr
}

func newLimaEnv() (*limaEnv, error) {
	env := &limaEnv{vmName: limaVMName}

	switch vmStatus(limaVMName) {
	case "Running":
		return env, nil
	case "Stopped":
		fmt.Printf("Starting existing VM %s...\n", limaVMName)
		cmd := exec.Command("limactl", "start", limaVMName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("start VM: %w", err)
		}
		return env, nil
	default:
		return env, createVM(env)
	}
}

func createVM(env *limaEnv) error {
	fmt.Printf("Creating VM %s (first run may take a few minutes)...\n", env.vmName)

	create := exec.Command("limactl", "create", "--tty=false", "--name", env.vmName, "template://docker")
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("create VM: %w", err)
	}

	start := exec.Command("limactl", "start", env.vmName)
	start.Stdout = os.Stdout
	start.Stderr = os.Stderr
	if err := start.Run(); err != nil {
		return fmt.Errorf("start VM: %w", err)
	}

	fmt.Println("Installing podman in VM...")
	if _, _, err := env.RunCommand("sudo", "apt-get", "update", "-qq"); err != nil {
		return fmt.Errorf("apt update: %w", err)
	}
	if _, _, err := env.RunCommand("sudo", "apt-get", "install", "-y", "podman"); err != nil {
		return fmt.Errorf("install podman: %w", err)
	}

	return nil
}

func vmStatus(vmName string) string {
	out, err := exec.Command("limactl", "list", vmName, "--format", "{{.Status}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (e *limaEnv) RunCommand(name string, args ...string) (string, string, error) {
	cmd := e.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("cmd failed: %w\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), nil
}

func (e *limaEnv) Command(name string, args ...string) *exec.Cmd {
	allArgs := append([]string{"shell", e.vmName, name}, args...)
	return exec.Command("limactl", allArgs...)
}

func (e *limaEnv) InstallBin(binPath string) (InstalledBin, error) {
	name := filepath.Base(binPath)
	tmpDst := filepath.Join("/tmp", name)

	cpCmd := exec.Command("limactl", "copy", binPath, e.vmName+":"+tmpDst)
	if out, err := cpCmd.CombinedOutput(); err != nil {
		return InstalledBin{}, fmt.Errorf("copy %s to VM: %w: %s", name, err, out)
	}

	dst := filepath.Join("/usr/local/bin", name)
	if _, _, err := e.RunCommand("sudo", "mv", tmpDst, dst); err != nil {
		return InstalledBin{}, fmt.Errorf("install %s: %w", name, err)
	}
	if _, _, err := e.RunCommand("sudo", "chmod", "+x", dst); err != nil {
		return InstalledBin{}, fmt.Errorf("chmod %s: %w", name, err)
	}

	return InstalledBin{env: e, pathToBin: dst}, nil
}

func (e *limaEnv) CopyDir(src, dst string) error {
	if _, _, err := e.RunCommand("mkdir", "-p", dst); err != nil {
		return err
	}
	cpCmd := exec.Command("limactl", "copy", "-r", src+"/.", e.vmName+":"+dst+"/")
	if out, err := cpCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy dir to VM: %w: %s", err, out)
	}
	return nil
}

func (e *limaEnv) RemoveAll(path string) error {
	_, _, err := e.RunCommand("rm", "-rf", path)
	return err
}

func (e *limaEnv) ReadFile(path string) (string, error) {
	stdout, stderr, err := e.RunCommand("cat", path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w\nstderr:\n%s", path, err, stderr)
	}
	return stdout, nil
}

func (e *limaEnv) ReadDir(path string) ([]string, error) {
	stdout, _, err := e.RunCommand("ls", "-1", path)
	if err != nil {
		return nil, err
	}
	var entries []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			entries = append(entries, line)
		}
	}
	return entries, nil
}

func (e *limaEnv) BuildImage(engine, contextDir, imageName string) error {
	tmpCtx := fmt.Sprintf("/tmp/build-ctx-%d", time.Now().UnixNano())
	if _, _, err := e.RunCommand("mkdir", "-p", tmpCtx); err != nil {
		return err
	}

	cpCmd := exec.Command("limactl", "copy", "-r", contextDir+"/.", e.vmName+":"+tmpCtx+"/")
	if out, err := cpCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy context to VM: %w: %s", err, out)
	}

	if _, stderr, err := e.RunCommand(engine, "build", "-t", imageName, tmpCtx); err != nil {
		return fmt.Errorf("build image: %w: %s", err, stderr)
	}

	_, _, _ = e.RunCommand("rm", "-rf", tmpCtx)
	return nil
}
