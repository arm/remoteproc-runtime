package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

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

	vm, err := limavm.NewWithDocker(rootDir, "../testdata", bins)
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
	requireRecentFinishOfDockerContainer(t, vm, containerID)

	_, stderr, err = vm.RunCommand("docker", "start", containerID)
	assert.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	_, stderr, err = vm.RunCommand("docker", "stop", containerID)
	assert.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")
	requireRecentFinishOfDockerContainer(t, vm, containerID)
}

func TestDockerRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	bins, err := repo.BuildBothBins(t.TempDir(), rootDir, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewWithDocker(rootDir, "../testdata", bins)
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

	vm, err := limavm.NewWithDocker(rootDir, "../testdata", bins)
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
	requireRecentFinishOfDockerContainer(t, vm, containerID)
}

func requireDockerContainerFinished(t *testing.T, vm limavm.LimaVM, containerID string) {
	t.Helper()

	const retryWindow = 15 * time.Second

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		exitCodeStr, stderr, err := vm.RunCommand(
			"docker", "inspect", "--format={{.State.ExitCode}}", containerID,
		)
		require.NoErrorf(c, err, "docker inspect error:\nstderr:\n%s", stderr)

		exitCodeStr = strings.TrimSpace(exitCodeStr)
		exitCode, err := strconv.Atoi(exitCodeStr)
		require.NoErrorf(c, err, "failed to parse ExitCode %q", exitCodeStr)

		require.Equal(c, 0, exitCode)
	}, retryWindow, time.Second)
}

func requireRecentFinishOfDockerContainer(t *testing.T, vm limavm.LimaVM, containerID string) {
	t.Helper()

	requireDockerContainerFinished(t, vm, containerID)

	const acceptableTimeDelta = 3 * time.Second

	finishedAt, stderr, err := vm.RunCommand(
		"docker", "inspect", "--format={{.State.FinishedAt}}", containerID,
	)
	require.NoError(t, err, "docker inspect error:\nstderr:\n%s", stderr)

	raw := strings.TrimSpace(finishedAt)

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	require.NoError(t, err, "failed to parse FinishedAt %q", raw)

	now := time.Now().UTC()
	delta := now.Sub(parsed)
	if delta < 0 {
		delta = -delta
	}

	require.LessOrEqual(t, delta, acceptableTimeDelta, "FinishedAt delta exceeds acceptable range: expected ≤%v, actual %v", acceptableTimeDelta, delta)
}
