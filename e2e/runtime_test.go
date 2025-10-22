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

func assertContainerStatus(t testing.TB, runtime limavm.InstalledBin, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	state, err := getContainerState(runtime, containerName)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, state.Status)
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

func generateBundle(targetDir string, remoteprocName string) error {
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
