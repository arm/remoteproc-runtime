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

	rootpathPrefixInVM := filepath.Join("/tmp", fmt.Sprintf("remoteproc-fake-root-%s", testID(t)))

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootpathPrefixInVM, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewDebian()
	require.NoError(t, err)
	defer vm.Cleanup()

	installedRuntime, err := vm.InstallBin(runtimeBin)
	require.NoError(t, err)

	simulatorBin, err := repo.BuildRemoteprocSimulator(t.TempDir(), limavm.BinBuildEnv)
	require.NoError(t, err)

	installedSimulator, err := vm.InstallBin(simulatorBin)
	require.NoError(t, err)

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-device"
		sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() {
			_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
			_ = sim.Stop()
		})

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePathInVM, err := generateBundleInVM(t, vm, remoteprocName)
		require.NoError(t, err)

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePathInVM,
			containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateCreated)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateRunning)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "running")

		_, stderr, err = installedRuntime.Run("kill", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		assertContainerStatus(t, installedRuntime, containerName, specs.StateStopped)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")

		_, stderr, err = installedRuntime.Run("delete", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
	})

	t.Run("errors when requested remoteproc name doesn't exist", func(t *testing.T) {
		processorName := "some-processor"
		sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(processorName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() {
			_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
			_ = sim.Stop()
		})

		uniqueID := testID(t)

		containerName := uniqueID
		bundlePathInVM, err := generateBundleInVM(t, vm, "other-processor")
		require.NoError(t, err)

		expectedErrorSubstring := "remote processor other-processor does not exist, available remote processors: "
		_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePathInVM, containerName)
		assert.ErrorContains(t, err, expectedErrorSubstring, "error doesn't contain: %s: stderr: %s", expectedErrorSubstring, stderr)
		assert.ErrorContains(t, err, processorName, "error doesn't contain expected processor name: %s: stderr: %s", processorName, stderr)
	})

	t.Run("killing process by pid stops the running container", func(t *testing.T) {
		remoteprocName := "nice-processor"
		sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() {
			_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
			_ = sim.Stop()
		})

		uniqueID := testID(t)
		containerName := uniqueID
		bundlePathInVM, err := generateBundleInVM(t, vm, remoteprocName)
		require.NoError(t, err)

		_, stderr, err := installedRuntime.Run("create", "--bundle", bundlePathInVM, containerName)
		require.NoError(t, err, "stderr: %s", stderr)

		pid, err := getContainerPid(installedRuntime, containerName)
		require.NoError(t, err)
		require.Greater(t, pid, 0)

		_, stderr, err = installedRuntime.Run("start", containerName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "running")

		_, stderr, err = vm.RunCommand("kill", "-TERM", fmt.Sprintf("%d", pid))
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")
	})

	t.Run("writes pid to file specified by --pid-file", func(t *testing.T) {
		remoteprocName := "oh-what-a-device"
		sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() {
			_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
			_ = sim.Stop()
		})

		containerName := testID(t)
		bundlePathInVM, err := generateBundleInVM(t, vm, remoteprocName)
		require.NoError(t, err)
		pidFile := filepath.Join(bundlePathInVM, "container.pid")

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePathInVM,
			"--pid-file", pidFile,
			containerName,
		)
		require.NoError(t, err, "stderr: %s", stderr)

		pid, err := getContainerPid(installedRuntime, containerName)
		require.NoError(t, err)
		require.Greater(t, pid, 0)
		requireFileExistsInVM(t, vm, pidFile)
		assertFileContentInVM(t, vm, pidFile, fmt.Sprintf("%d", pid))
	})

	t.Run("proxy process namespacing", func(t *testing.T) {
		installedRuntimeSudo := limavm.NewSudo(installedRuntime)

		t.Run("creates process in requested namespace when root", func(t *testing.T) {
			remoteprocName := "lovely-blue-device"
			sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
			if err := sim.Start(); err != nil {
				t.Fatalf("failed to run simulator: %s", err)
			}
			t.Cleanup(func() {
				_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
				_ = sim.Stop()
			})

			uniqueID := testID(t)
			containerName := uniqueID
			bundlePathInVM, err := generateBundleInVM(
				t,
				vm,
				remoteprocName,
				specs.LinuxNamespace{Type: specs.MountNamespace},
			)
			require.NoError(t, err)
			_, stderr, err := installedRuntimeSudo.Run(
				"create",
				"--bundle", bundlePathInVM,
				containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			t.Cleanup(func() {
				_, _, _ = installedRuntimeSudo.Run("delete", containerName)
			})

			pid, err := getContainerPid(installedRuntimeSudo, containerName)
			require.NoError(t, err)

			requireDifferentMountNamespace(t, vm, pid)

			remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")

			_, stderr, err = installedRuntimeSudo.Run("start", containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "running")
		})

		t.Run("creates process in user's namespace when not root", func(t *testing.T) {
			remoteprocName := "lovely-green-device"
			sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
			if err := sim.Start(); err != nil {
				t.Fatalf("failed to run simulator: %s", err)
			}
			t.Cleanup(func() {
				_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
				_ = sim.Stop()
			})

			containerName := testID(t)
			bundlePathInVM, err := generateBundleInVM(
				t,
				vm,
				remoteprocName,
				specs.LinuxNamespace{Type: specs.MountNamespace},
			)
			require.NoError(t, err)
			_, stderr, err := installedRuntime.Run(
				"create",
				"--bundle", bundlePathInVM,
				containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			t.Cleanup(func() {
				_, _, _ = installedRuntime.Run("delete", containerName)
			})

			pid, err := getContainerPid(installedRuntime, containerName)
			require.NoError(t, err)

			requireSameMountNamespace(t, vm, uint(pid))

			remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")

			_, stderr, err = installedRuntime.Run("start", containerName)
			require.NoError(t, err, "stderr: %s", stderr)
			remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "running")
		})
	})

	t.Run("When a custom path is set in /sys/module/firmware_class/parameters/path, the firmware will be stored there", func(t *testing.T) {
		remoteprocName := "nice-device"
		sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() {
			_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
			_ = sim.Stop()
		})

		containerName := testID(t)
		bundlePathInVM, err := generateBundleInVM(t, vm, remoteprocName)
		require.NoError(t, err)

		_, stderr, err := installedRuntime.Run(
			"create",
			"--bundle", bundlePathInVM,
			containerName)
		require.NoError(t, err, "stderr: %s", stderr)

		customFirmwareStorageDirectory := filepath.Join(rootpathPrefixInVM, "my", "firmware", "path")

		_, _, err = vm.RunCommand("sh", "-c", fmt.Sprintf("echo -n %s > %s",
			customFirmwareStorageDirectory,
			filepath.Join(
				rootpathPrefixInVM,
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

		assertFirmwareFileExistsInVM(t, vm, customFirmwareStorageDirectory)
	})
}

func assertFirmwareFileExistsInVM(t *testing.T, vm limavm.Debian, firmwareStorageDirectory string) {
	t.Helper()
	entries, err := vm.ReadDir(firmwareStorageDirectory)
	require.NoError(t, err)

	require.Greater(t, len(entries), 0, "expected at least one firmware file in %s", firmwareStorageDirectory)
}

func assertContainerStatus(t testing.TB, runtime limavm.Runnable, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	state, err := getContainerState(runtime, containerName)
	require.NoError(t, err)
	assert.Equal(t, wantStatus, state.Status)
}

func assertFileContentInVM(t *testing.T, vm limavm.Debian, path string, wantContent string) {
	t.Helper()
	gotContent, err := vm.ReadFileAsString(path)
	if assert.NoError(t, err) {
		assert.Equal(t, wantContent, gotContent)
	}
}

func requireFileExistsInVM(t *testing.T, vm limavm.Debian, path string) {
	t.Helper()
	_, stderr, err := vm.RunCommand("test", "-e", path)
	require.NoError(t, err, "failed to check file existence %s in VM: stderr: %s", path, stderr)
}

func requireSameMountNamespace(t testing.TB, vm limavm.Debian, pid uint) {
	t.Helper()
	hostMountNS, stderr, err := vm.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)
	pidMountNS, stderr, err := vm.RunCommand("sudo", "readlink", fmt.Sprintf("/proc/%d/ns/mnt", pid))
	require.NoError(t, err, "stderr: %s", stderr)

	require.Equal(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(pidMountNS))
}

func requireDifferentMountNamespace(t testing.TB, vm limavm.Debian, pid int) {
	t.Helper()
	hostMountNS, stderr, err := vm.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)
	pidMountNS, stderr, err := vm.RunCommand("sudo", "readlink", fmt.Sprintf("/proc/%d/ns/mnt", pid))
	require.NoError(t, err, "stderr: %s", stderr)

	require.NotEqual(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(pidMountNS))
}

func testID(t testing.TB) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func getContainerPid(runtime limavm.Runnable, containerName string) (int, error) {
	state, err := getContainerState(runtime, containerName)
	if err != nil {
		return 0, err
	}
	return state.Pid, err
}

func getContainerState(runtime limavm.Runnable, containerName string) (specs.State, error) {
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

func generateBundleInVM(t *testing.T, vm limavm.Debian, remoteprocName string, namespaces ...specs.LinuxNamespace) (string, error) {
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

	bundlePathInVM, err := vm.Copy(bundlePathOnHost, filepath.Join("/tmp", testID(t)))
	if err != nil {
		return "", err
	}
	t.Cleanup(func() { _ = vm.RemoveFile(bundlePathInVM) })

	return bundlePathInVM, nil
}
