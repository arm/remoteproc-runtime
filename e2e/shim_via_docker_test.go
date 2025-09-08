package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/limavm"
	"github.com/Arm-Debug/remoteproc-runtime/e2e/remoteproc"
	"github.com/Arm-Debug/remoteproc-runtime/e2e/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	bins, err := repo.BuildBothBins(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.New(rootDir, bins, "../testdata/test-image.tar")
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	containerID, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
		"test-image")
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	_, stderr, err = vm.RunCommand("docker", "stop", containerID)
	assert.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")
}

func TestDockerRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	bins, err := repo.BuildBothBins(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.New(rootDir, bins, "../testdata/test-image.tar")
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := remoteproc.NewSimulator(rootDir).WithName("a-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	_, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", "other-processor"),
		"test-image")
	assert.Error(t, err)
	assert.Contains(t, stderr, "other-processor is not in the list of available remote processors")
}

func TestDockerKillProcessByPid(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	bins, err := repo.BuildBothBins(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.New(rootDir, bins, "../testdata/test-image.tar")
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	containerID, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
		"test-image")
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	stdout, stderr, err := vm.RunCommand("docker", "inspect", "--format={{.State.Pid}}", containerID)
	require.NoError(t, err, "stderr: %s", stderr)
	pid, err := strconv.Atoi(strings.TrimSpace(stdout))

	_, _, err = vm.RunCommand("kill", "-TERM", fmt.Sprintf("%d", pid))
	require.NoError(t, err)
	remoteproc.RequireState(t, sim.DeviceDir(), "offline")
}
