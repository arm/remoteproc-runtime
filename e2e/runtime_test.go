package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntime(t *testing.T) {
	limavm.Require(t)

	dirMountedInVM := t.TempDir()

	rootpathPrefix := filepath.Join(dirMountedInVM, "fake-root")
	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootpathPrefix, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewDebian(dirMountedInVM)
	require.NoError(t, err)
	defer vm.Cleanup()

	installedRuntime, err := vm.InstallBin(runtimeBin)
	require.NoError(t, err)

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-device"
		sim := remoteproc.NewSimulator(rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePath := filepath.Join(dirMountedInVM, uniqueID)
		require.NoError(t, generateBundle(bundlePath, remoteprocName))

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePath,
			containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateCreated)
		remoteproc.AssertState(t, sim.DeviceDir(), "offline")

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateRunning)
		remoteproc.AssertState(t, sim.DeviceDir(), "running")

		_, stderr, err = installedRuntime.Run("kill", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateStopped)
		remoteproc.AssertState(t, sim.DeviceDir(), "offline")

		_, stderr, err = installedRuntime.Run("delete", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
	})

	t.Run("errors when requested remoteproc name doesn't exist", func(t *testing.T) {
		sim := remoteproc.NewSimulator(rootpathPrefix).WithName("some-processor")
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePath := filepath.Join(dirMountedInVM, uniqueID)
		require.NoError(t, generateBundle(bundlePath, "other-processor"))

		_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePath, containerName)
		assert.ErrorContains(t, err, "remote processor other-processor does not exist, available remote processors: some-processor", "stderr: %s", stderr)
	})

	t.Run("killing process by pid stops the running container", func(t *testing.T) {
		remoteprocName := "nice-processor"
		sim := remoteproc.NewSimulator(rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePath := filepath.Join(dirMountedInVM, uniqueID)
		require.NoError(t, generateBundle(bundlePath, remoteprocName))

		_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePath, containerName)
		require.NoError(t, err, "stderr: %s", stderr)

		pid, err := getContainerPid(installedRuntime, containerName)
		require.NoError(t, err)
		require.Greater(t, pid, 0)

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "running")

		_, stderr, err = vm.RunCommand("kill", "-TERM", fmt.Sprintf("%d", pid))
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "offline")
	})

	t.Run("writes pid to file specified by --pid-file", func(t *testing.T) {
		remoteprocName := "oh-what-a-device"
		sim := remoteproc.NewSimulator(rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePath := filepath.Join(dirMountedInVM, uniqueID)
		require.NoError(t, generateBundle(bundlePath, remoteprocName))
		pidFile := filepath.Join(dirMountedInVM, uniqueID, "container.pid")

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePath,
			"--pid-file", pidFile,
			containerName,
		)
		require.NoError(t, err, "stderr: %s", stderr)

		pid, err := getContainerPid(installedRuntime, containerName)
		require.NoError(t, err)
		require.Greater(t, pid, 0)

		require.FileExists(t, pidFile)
		assertFileContent(t, pidFile, fmt.Sprintf("%d", pid))
	})
}

func testID(t testing.TB) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func TestRuntimeKillProcessByPid(t *testing.T) {
	limavm.Require(t)

	rootDir := t.TempDir()

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewDebian(rootDir)
	require.NoError(t, err)
	defer vm.Cleanup()

	installedRuntime, err := vm.InstallBin(runtimeBin)
	require.NoError(t, err)

	sim := remoteproc.NewSimulator(rootDir).WithName("nice-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer func() { _ = sim.Stop() }()

	const containerName = "test-container"

	bundlePath := filepath.Join(rootDir, "bundle")
	require.NoError(t, generateBundle(bundlePath, "nice-processor"))
	_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePath, containerName)
	require.NoError(t, err, "stderr: %s", stderr)

	pid, err := getContainerPid(installedRuntime, containerName)
	require.NoError(t, err)
	require.Greater(t, pid, 0)

	_, stderr, err = installedRuntime.Run("start", containerName)
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.RequireState(t, sim.DeviceDir(), "running")

	_, stderr, err = vm.RunCommand("kill", "-TERM", fmt.Sprintf("%d", pid))
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")
}

func TestRuntimeWriteProcessPid(t *testing.T) {
	limavm.Require(t)

	rootDir := t.TempDir()
	remoteprocName := "oh-what-a-device"

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewDebian(rootDir)
	require.NoError(t, err)
	defer vm.Cleanup()

	installedRuntime, err := vm.InstallBin(runtimeBin)
	require.NoError(t, err)

	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer func() { _ = sim.Stop() }()

	const containerName = "test-container"

	bundlePath := filepath.Join(rootDir, "bundle")
	pidFilePath := filepath.Join(rootDir, "pidfile.txt")
	require.NoError(t, generateBundle(bundlePath, remoteprocName))
	_, stderr, err := installedRuntime.Run(
		"create",
		"--bundle", bundlePath,
		"--pid-file", pidFilePath,
		containerName,
	)
	require.NoError(t, err, "stderr: %s", stderr)

	pid, err := getContainerPid(installedRuntime, containerName)
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

	vm, err := limavm.NewDebian(rootDir)
	require.NoError(t, err)
	t.Cleanup(vm.Cleanup)

	installedRuntime, err := vm.InstallBin(runtimeBin)
	require.NoError(t, err)

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

	_, stderr, err := installedRuntime.Run(
		"create", "--bundle", bundlePath,
		containerName,
	)
	require.NoError(t, err, "stderr: %s", stderr)
	t.Cleanup(func() {
		_, _, _ = installedRuntime.Run("delete", containerName)
	})

	pid, err := checkContainerRunning(func() (specs.State, error) {
		stdout, stderr, err := installedRuntime.Run("state", containerName)
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

	vm, err := limavm.NewDebian(rootDir)
	require.NoError(t, err)
	t.Cleanup(vm.Cleanup)

	_, err = vm.InstallBin(runtimeBin)
	require.NoError(t, err)

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

func assertContainerStatus(t testing.TB, runtime limavm.InstalledBin, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	state, err := getContainerState(runtime, containerName)
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

func getContainerPid(runtime limavm.InstalledBin, containerName string) (int, error) {
	state, err := getContainerState(runtime, containerName)
	if err != nil {
		return 0, err
	}
	return state.Pid, err
}

func getContainerState(runtime limavm.InstalledBin, containerName string) (specs.State, error) {
	var state specs.State
	out, stderr, err := runtime.Run("state", containerName)
	if err != nil {
		return state, fmt.Errorf("command failed: %w\n<stderr>\n%s\n</stderr>", err, stderr)
	}
	err = json.Unmarshal([]byte(out), &state)
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

func assertFileContent(t *testing.T, path string, wantContent string) {
	t.Helper()
	gotContent, err := os.ReadFile(path)
	if assert.NoError(t, err) {
		assert.Equal(t, wantContent, string(gotContent))
	}
}
