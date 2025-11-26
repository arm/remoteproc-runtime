package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocker(t *testing.T) {
	limavm.Require(t)

	rootpathPrefix := t.TempDir()
	bins, err := repo.BuildBothBins(t.TempDir(), rootpathPrefix, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewDocker(rootpathPrefix)
	require.NoError(t, err)
	defer vm.Cleanup()

	for _, bin := range bins {
		_, err := vm.InstallBin(bin)
		require.NoError(t, err)
	}
	simulatorArea := t.TempDir()
	simulatorBin, err := repo.BuildRemoteprocSimulator(simulatorArea, limavm.BinBuildEnv)
	require.NoError(t, err)
	vmSimulator, err := vm.InstallBin(simulatorBin)
	require.NoError(t, err)

	imageName := "test-image"
	require.NoError(t, vm.BuildImage("../testdata", imageName))

	simulatorIndex := uint(100)
	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-docker-device"
		sim := remoteproc.NewSimulator(vmSimulator, simulatorArea).WithName(remoteprocName).WithIndex(simulatorIndex)
		simulatorIndex++
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() { _ = sim.Stop() })

		remoteproc.AssertState(t, sim.DeviceDir(), "offline")

		containerID, stderr, err := vm.RunCommand(
			"docker", "run", "-d",
			"--runtime", "io.containerd.remoteproc.v1",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
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
	})

	t.Run("errors when requested remoteproc name doesn't exist", func(t *testing.T) {
		sim := remoteproc.NewSimulator(vmSimulator, simulatorArea).WithName("a-processor").WithIndex(simulatorIndex)
		simulatorIndex++
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() { _ = sim.Stop() })

		_, stderr, err := vm.RunCommand(
			"docker", "run", "-d",
			"--runtime", "io.containerd.remoteproc.v1",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", "other-processor"),
			imageName)
		assert.Error(t, err)
		assert.Contains(t, stderr, "remote processor other-processor does not exist, available remote processors: ")
		assert.Contains(t, stderr, "a-processor")
	})

	t.Run("killing process by pid stops the running container", func(t *testing.T) {
		remoteprocName := "another-yolo-docker-device"
		sim := remoteproc.NewSimulator(vmSimulator, simulatorArea).WithName(remoteprocName).WithIndex(simulatorIndex)
		simulatorIndex++
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		containerID, stderr, err := vm.RunCommand(
			"docker", "run", "-d",
			"--runtime", "io.containerd.remoteproc.v1",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "running")

		stdout, stderr, err := vm.RunCommand("docker", "inspect", "--format={{.State.Pid}}", containerID)
		require.NoError(t, err, "stderr: %s", stderr)
		pid, parseErr := strconv.Atoi(strings.TrimSpace(stdout))
		require.NoError(t, parseErr)

		_, _, err = vm.RunCommand("kill", "-TERM", fmt.Sprintf("%d", pid))
		require.NoError(t, err)
		remoteproc.RequireState(t, sim.DeviceDir(), "offline")
		requireRecentFinishOfDockerContainer(t, vm, containerID)
	})
}

func requireDockerContainerFinished(t *testing.T, vm limavm.Docker, containerID string) {
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

func requireRecentFinishOfDockerContainer(t *testing.T, vm limavm.Docker, containerID string) {
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
