package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/testenv"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntime(t *testing.T) {
	rootpathPrefix := filepath.Join("/tmp", "fake-root")

	env := testenv.New(t)

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootpathPrefix, testenv.BuildEnv())
	require.NoError(t, err)

	installedRuntime, err := env.InstallBin(runtimeBin)
	require.NoError(t, err)

	simulatorBin, err := remoteproc.DownloadSimulator(context.Background())
	require.NoError(t, err)
	installedSimulator, err := env.InstallBin(simulatorBin)
	require.NoError(t, err)

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-device"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePath, err := generateBundle(t, env, remoteprocName)
		require.NoError(t, err)

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePath,
			containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateCreated)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateRunning)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		_, stderr, err = installedRuntime.Run("kill", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateStopped)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

		_, stderr, err = installedRuntime.Run("delete", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
	})

	t.Run("errors when requested remoteproc name doesn't exist", func(t *testing.T) {
		processorName := "some-processor"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(processorName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)

		containerName := uniqueID
		bundlePath, err := generateBundle(t, env, "other-processor")
		require.NoError(t, err)

		expectedErrorSubstring := "remote processor other-processor does not exist, available remote processors: "
		_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePath, containerName)
		assert.ErrorContains(t, err, expectedErrorSubstring, "error doesn't contain: %s: stderr: %s", expectedErrorSubstring, stderr)
		assert.ErrorContains(t, err, processorName, "error doesn't contain expected processor name: %s: stderr: %s", processorName, stderr)
	})

	t.Run("killing process by pid stops the running container", func(t *testing.T) {
		remoteprocName := "nice-processor"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePath, err := generateBundle(t, env, remoteprocName)
		require.NoError(t, err)

		_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePath, containerName)
		require.NoError(t, err, "stderr: %s", stderr)

		pid, err := getContainerPid(installedRuntime, containerName)
		require.NoError(t, err)
		require.Greater(t, pid, 0)

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		_, stderr, err = env.RunCommand("kill", "-TERM", fmt.Sprintf("%d", pid))
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")
	})

	t.Run("writes pid to file specified by --pid-file", func(t *testing.T) {
		remoteprocName := "oh-what-a-device"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		containerName := testID(t)
		bundlePath, err := generateBundle(t, env, remoteprocName)
		require.NoError(t, err)
		pidFile := filepath.Join(bundlePath, "container.pid")

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
		requireFileExists(t, env, pidFile)
		assertFileContent(t, env, pidFile, fmt.Sprintf("%d", pid))
	})

	t.Run("proxy process namespacing", func(t *testing.T) {
		installedRuntimeSudo := testenv.NewSudo(installedRuntime)

		t.Run("creates process in requested namespace when root", func(t *testing.T) {
			remoteprocName := "lovely-blue-device"
			sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
			if err := sim.Start(); err != nil {
				t.Fatalf("failed to run simulator: %s", err)
			}
			defer func() { _ = sim.Stop() }()

			uniqueID := testID(t)
			containerName := uniqueID
			bundlePath, err := generateBundle(
				t,
				env,
				remoteprocName,
				specs.LinuxNamespace{Type: specs.MountNamespace},
			)
			require.NoError(t, err)
			_, stderr, err := installedRuntimeSudo.Run(
				"create",
				"--bundle", bundlePath,
				containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			t.Cleanup(func() {
				_, _, _ = installedRuntimeSudo.Run("delete", containerName)
			})

			pid, err := getContainerPid(installedRuntimeSudo, containerName)
			require.NoError(t, err)

			requireDifferentMountNamespace(t, env, pid)

			remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

			_, stderr, err = installedRuntimeSudo.Run("start", containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			remoteproc.AssertState(t, env, sim.DeviceDir(), "running")
		})

		t.Run("creates process in user's namespace when not root", func(t *testing.T) {
			remoteprocName := "lovely-green-device"
			sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
			if err := sim.Start(); err != nil {
				t.Fatalf("failed to run simulator: %s", err)
			}
			defer func() { _ = sim.Stop() }()

			containerName := testID(t)
			bundlePath, err := generateBundle(
				t,
				env,
				remoteprocName,
				specs.LinuxNamespace{Type: specs.MountNamespace},
			)
			require.NoError(t, err)
			_, stderr, err := installedRuntime.Run(
				"create",
				"--bundle", bundlePath,
				containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			t.Cleanup(func() {
				_, _, _ = installedRuntime.Run("delete", containerName)
			})

			pid, err := getContainerPid(installedRuntime, containerName)
			require.NoError(t, err)

			requireSameMountNamespace(t, env, uint(pid))

			remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

			_, stderr, err = installedRuntime.Run("start", containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			remoteproc.AssertState(t, env, sim.DeviceDir(), "running")
		})
	})

	t.Run("When a custom path is set in /sys/module/firmware_class/parameters/path, the firmware will be stored there", func(t *testing.T) {
		remoteprocName := "nice-device"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		containerName := testID(t)
		bundlePath, err := generateBundle(t, env, remoteprocName)
		require.NoError(t, err)

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePath,
			containerName)
		require.NoError(t, err, "stderr: %s", stderr)

		customFirmwareStorageDirectory := filepath.Join(rootpathPrefix, "my", "firmware", "path")

		_, _, err = env.RunCommand("sh", "-c", fmt.Sprintf("echo -n %s > %s",
			customFirmwareStorageDirectory,
			filepath.Join(
				rootpathPrefix,
				"sys",
				"module",
				"firmware_class",
				"parameters",
				"path",
			),
		))
		require.NoError(t, err, "failed to update custom firmware path")

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateRunning)

		assertFirmwareFileExists(t, env, customFirmwareStorageDirectory)
	})
}

func assertFirmwareFileExists(t *testing.T, env testenv.Env, firmwareStorageDirectory string) {
	t.Helper()
	entries, err := env.ReadDir(firmwareStorageDirectory)
	require.NoError(t, err)

	require.Greater(t, len(entries), 0, "expected at least one firmware file in %s", firmwareStorageDirectory)
}

func assertContainerStatus(t testing.TB, runtime testenv.Runnable, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	state, err := getContainerState(runtime, containerName)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, state.Status)
}

func assertFileContent(t *testing.T, env testenv.Env, path string, wantContent string) {
	t.Helper()
	gotContent, err := env.ReadFile(path)
	if assert.NoError(t, err) {
		assert.Equal(t, wantContent, gotContent)
	}
}

func requireFileExists(t *testing.T, env testenv.Env, path string) {
	t.Helper()
	_, stderr, err := env.RunCommand("test", "-e", path)
	require.NoError(t, err, "failed to check file existence %s: stderr: %s", path, stderr)
}

func requireSameMountNamespace(t testing.TB, env testenv.Env, pid uint) {
	t.Helper()
	hostMountNS, stderr, err := env.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)
	pidMountNS, stderr, err := env.RunCommand("sudo", "readlink", fmt.Sprintf("/proc/%d/ns/mnt", pid))
	require.NoError(t, err, "stderr: %s", stderr)

	require.Equal(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(pidMountNS))
}

func requireDifferentMountNamespace(t testing.TB, env testenv.Env, pid int) {
	t.Helper()
	hostMountNS, stderr, err := env.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)
	pidMountNS, stderr, err := env.RunCommand("sudo", "readlink", fmt.Sprintf("/proc/%d/ns/mnt", pid))
	require.NoError(t, err, "stderr: %s", stderr)

	require.NotEqual(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(pidMountNS))
}

func testID(t testing.TB) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "'", "")
	return name
}

func getContainerPid(runtime testenv.Runnable, containerName string) (int, error) {
	state, err := getContainerState(runtime, containerName)
	if err != nil {
		return 0, err
	}
	return state.Pid, err
}

func getContainerState(runtime testenv.Runnable, containerName string) (specs.State, error) {
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

func generateBundle(t *testing.T, env testenv.Env, remoteprocName string, namespaces ...specs.LinuxNamespace) (string, error) {
	bundlePathOnHost := t.TempDir()
	const bundleRoot = "rootfs"
	const firmwareName = "hello_world.elf"

	if err := os.MkdirAll(filepath.Join(bundlePathOnHost, bundleRoot), 0o755); err != nil {
		return "", err
	}

	firmwarePath := filepath.Join(bundlePathOnHost, bundleRoot, firmwareName)
	if err := os.WriteFile(firmwarePath, []byte("pretend binary"), 0o644); err != nil {
		return "", err
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

	specPath := filepath.Join(bundlePathOnHost, "config.json")
	specData, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(specPath, specData, 0o644); err != nil {
		return "", err
	}

	bundlePath := filepath.Join("/tmp", testID(t))
	if err := env.CopyDir(bundlePathOnHost, bundlePath); err != nil {
		return "", err
	}
	t.Cleanup(func() { _ = env.RemoveAll(bundlePath) })

	return bundlePath, nil
}
