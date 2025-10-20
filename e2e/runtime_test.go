package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer func() { _ = sim.Stop() }()
	bin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, nil)
	require.NoError(t, err)

	containerName := "test-container"

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, remoteprocName))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateCreated)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	_, err = invokeRuntime(bin, "start", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateRunning)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	_, err = invokeRuntime(bin, "kill", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateStopped)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	_, err = invokeRuntime(bin, "delete", containerName)
	require.NoError(t, err)
}

func TestRuntimeRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	sim := remoteproc.NewSimulator(rootDir).WithName("some-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer func() { _ = sim.Stop() }()
	bin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, nil)
	require.NoError(t, err)

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, "other-processor"))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, "test-container")
	assert.ErrorContains(t, err, "remote processor other-processor does not exist, available remote processors: some-processor")
}

func TestRuntimeKillProcessByPid(t *testing.T) {
	rootDir := t.TempDir()
	sim := remoteproc.NewSimulator(rootDir).WithName("nice-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer func() { _ = sim.Stop() }()
	bin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, nil)
	require.NoError(t, err)

	const containerName = "test-container"

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, "nice-processor"))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, containerName)
	require.NoError(t, err)

	pid, err := getContainerPid(bin, containerName)
	require.NoError(t, err)
	require.Greater(t, pid, 0)

	_, err = invokeRuntime(bin, "start", containerName)
	require.NoError(t, err)
	remoteproc.RequireState(t, sim.DeviceDir(), "running")

	require.NoError(t, sendSignal(pid, syscall.SIGTERM))
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")
}

func TestRuntimeWriteProcessPid(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "oh-what-a-device"
	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer func() { _ = sim.Stop() }()
	bin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, nil)
	require.NoError(t, err)

	const containerName = "test-container"

	bundlePath := t.TempDir()
	pidFilePath := filepath.Join(t.TempDir(), "pidfile.txt")
	require.NoError(t, generateBundle(bundlePath, remoteprocName))
	_, err = invokeRuntime(
		bin, "create",
		"--bundle", bundlePath,
		"--pid-file", pidFilePath,
		containerName,
	)
	require.NoError(t, err)

	pid, err := getContainerPid(bin, containerName)
	require.NoError(t, err)
	require.Greater(t, pid, 0)

	require.FileExists(t, pidFilePath)
	assertFileContent(t, pidFilePath, fmt.Sprintf("%d", pid))
}

func TestRuntimeProxyKeepsHostNamespaceWhenNotRoot(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "a-lovely-blue-device"

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewAlpine(rootDir, runtimeBin)
	require.NoError(t, err)
	t.Cleanup(vm.Cleanup)

	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	t.Cleanup(func() { _ = sim.Stop() })

	const containerName = "not-root-namespace-container"
	bundlePath := filepath.Join(rootDir, "not-root-namespace-bundle")
	require.NoError(t, generateBundle(
		bundlePath,
		remoteprocName,
		specs.LinuxNamespace{Type: specs.MountNamespace},
	))

	_, stderr, err := vm.RunCommand(
		"remoteproc-runtime",
		"create", "--bundle", bundlePath,
		containerName,
	)
	require.NoError(t, err, "stderr: %s", stderr)
	t.Cleanup(func() {
		_, _, _ = vm.RunCommand("remoteproc-runtime", "delete", containerName)
	})

	pid, err := checkContainerRunning(func() (specs.State, error) {
		stdout, stderr, err := vm.RunCommand("remoteproc-runtime", "state", containerName)
		if err != nil {
			return specs.State{}, fmt.Errorf("stderr: %s: %w", stderr, err)
		}
		var state specs.State
		if err := json.Unmarshal([]byte(stdout), &state); err != nil {
			return specs.State{}, err
		}
		return state, nil
	})
	require.NoError(t, err)

	hostMountNS, stderr, err := vm.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)

	proxyMountNS, stderr, err := vm.RunCommand("readlink", fmt.Sprintf("/proc/%d/ns/mnt", pid))
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Equal(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(proxyMountNS))
}

func TestRuntimeProxyKeepsHostNamespaceWhenRootInLimaVM(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "a-lovely-blue-device"

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewAlpine(rootDir, runtimeBin)
	require.NoError(t, err)
	t.Cleanup(vm.Cleanup)

	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	t.Cleanup(func() { _ = sim.Stop() })

	const containerName = "root-namespace-container"
	bundlePath := filepath.Join(rootDir, "root-namespace-bundle")
	require.NoError(t, generateBundle(
		bundlePath,
		remoteprocName,
		specs.LinuxNamespace{Type: specs.MountNamespace},
	))

	_, stderr, err := vm.RunCommand(
		"sudo", "remoteproc-runtime",
		"create", "--bundle", bundlePath,
		containerName,
	)
	require.NoError(t, err, "stderr: %s", stderr)
	t.Cleanup(func() {
		_, _, _ = vm.RunCommand("sudo", "remoteproc-runtime", "delete", containerName)
	})

	pid, err := checkContainerRunning(func() (specs.State, error) {
		stdout, stderr, err := vm.RunCommand("sudo", "remoteproc-runtime", "state", containerName)
		if err != nil {
			return specs.State{}, fmt.Errorf("stderr: %s: %w", stderr, err)
		}
		var state specs.State
		if err := json.Unmarshal([]byte(stdout), &state); err != nil {
			return specs.State{}, err
		}
		return state, nil
	})
	require.NoError(t, err)

	hostMountNS, stderr, err := vm.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)

	proxyMountNS, stderr, err := vm.RunCommand("sudo", "readlink", fmt.Sprintf("/proc/%d/ns/mnt", pid))
	require.NoError(t, err, "stderr: %s", stderr)
	assert.NotEqual(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(proxyMountNS))

	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	_, stderr, err = vm.RunCommand("sudo", "remoteproc-runtime", "start", containerName)
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")
}

func assertContainerStatus(t testing.TB, bin repo.RuntimeBin, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	state, err := getContainerState(bin, containerName)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, state.Status)
}

func checkContainerRunning(fetch func() (specs.State, error)) (int, error) {
	state, err := fetch()
	if err != nil {
		return 0, err
	}
	if state.Pid <= 0 {
		return 0, fmt.Errorf("container is not running - pid is %d", state.Pid)
	}
	return state.Pid, nil
}

func getContainerPid(bin repo.RuntimeBin, containerName string) (int, error) {
	state, err := getContainerState(bin, containerName)
	if err != nil {
		return 0, err
	}
	return state.Pid, err
}

func getContainerState(bin repo.RuntimeBin, containerName string) (specs.State, error) {
	var state specs.State
	out, err := invokeRuntime(bin, "state", containerName)
	if err != nil {
		return state, fmt.Errorf("can't get container state: %w", err)
	}
	err = json.Unmarshal(out, &state)
	if err != nil {
		return state, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return state, nil
}

func generateBundle(targetDir string, remoteprocName string, namespaces ...specs.LinuxNamespace) error {
	const bundleRoot = "rootfs"
	const firmwareName = "hello_world.elf"

	if err := os.MkdirAll(filepath.Join(targetDir, bundleRoot), 0o755); err != nil {
		return err
	}
	firmwarePath := filepath.Join(targetDir, bundleRoot, firmwareName)
	if err := os.WriteFile(firmwarePath, []byte("pretend binary"), 0o644); err != nil {
		return err
	}

	spec := &specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			User: specs.User{UID: 0, GID: 0},
			Args: []string{firmwareName},
			Cwd:  "/",
		},
		Root: &specs.Root{Path: bundleRoot},
		Annotations: map[string]string{
			"remoteproc.name": remoteprocName,
		},
		Linux: &specs.Linux{Namespaces: namespaces},
	}

	specPath := filepath.Join(targetDir, "config.json")
	specData, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, specData, 0o644); err != nil {
		return err
	}
	return nil
}

func invokeRuntime(bin repo.RuntimeBin, args ...string) ([]byte, error) {
	cmd := exec.Command(string(bin), args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w\n<stderr>\n%s\n</stderr>", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func sendSignal(pid int, signal syscall.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}
	err = process.Signal(signal)
	if err != nil {
		return fmt.Errorf("failed to send signal %s to process %d: %w", signal, pid, err)
	}
	return nil
}

func assertFileContent(t *testing.T, path string, wantContent string) {
	t.Helper()
	gotContent, err := os.ReadFile(path)
	if assert.NoError(t, err) {
		assert.Equal(t, wantContent, string(gotContent))
	}
}
