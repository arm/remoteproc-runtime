package e2e

import (
	"encoding/json"
	"fmt"
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

func TestRuntimeProxyKeepsHostNamespaceWhenNotRootInLimaVM(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "a-lovely-blue-device"

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewWithPodman(rootDir, "../testdata", runtimeBin)
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

	stdout, stderr, err := vm.RunCommand("remoteproc-runtime", "state", containerName)
	require.NoError(t, err, "stderr: %s", stderr)

	var state specs.State
	require.NoError(t, json.Unmarshal([]byte(stdout), &state))
	require.Greater(t, state.Pid, 0)

	hostMountNS, stderr, err := vm.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)

	proxyMountNS, stderr, err := vm.RunCommand("readlink", fmt.Sprintf("/proc/%d/ns/mnt", state.Pid))
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Equal(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(proxyMountNS))
}

func TestRuntimeProxyKeepsHostNamespaceWhenRootInLimaVM(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "a-lovely-blue-device"

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewWithPodman(rootDir, "../testdata", runtimeBin)
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

	stdout, stderr, err := vm.RunCommand("sudo", "remoteproc-runtime", "state", containerName)
	require.NoError(t, err, "stderr: %s", stderr)

	var state specs.State
	require.NoError(t, json.Unmarshal([]byte(stdout), &state))
	require.Greater(t, state.Pid, 0)

	hostMountNS, stderr, err := vm.RunCommand("readlink", "/proc/self/ns/mnt")
	require.NoError(t, err, "stderr: %s", stderr)

	proxyMountNS, stderr, err := vm.RunCommand("sudo", "readlink", fmt.Sprintf("/proc/%d/ns/mnt", state.Pid))
	require.NoError(t, err, "stderr: %s", stderr)
	assert.NotEqual(t, strings.TrimSpace(hostMountNS), strings.TrimSpace(proxyMountNS))

	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	_, stderr, err = vm.RunCommand("sudo", "remoteproc-runtime", "start", containerName)
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	_, stderr, err = vm.RunCommand("sudo", "remoteproc-runtime", "kill", containerName)
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")
}
