package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocker(t *testing.T) {
	rootpathPrefix := filepath.Join("/tmp", "fake-root")

	env := testenv.New(t)

	bins, err := repo.BuildBothBins(t.TempDir(), rootpathPrefix, testenv.BuildEnv())
	require.NoError(t, err)

	for _, bin := range bins {
		_, err := env.InstallBin(bin)
		require.NoError(t, err)
	}

	simulatorBin, err := remoteproc.DownloadSimulator(context.Background())
	require.NoError(t, err)
	installedSimulator, err := env.InstallBin(simulatorBin)
	require.NoError(t, err)

	imageName := "test-image"
	require.NoError(t, env.BuildImage("docker", "../testdata", imageName))

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-docker-device"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

		containerID, stderr, err := env.RunCommand(
			"docker", "run", "-d",
			"--runtime", "io.containerd.remoteproc.v1",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		_, stderr, err = env.RunCommand("docker", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")
		requireRecentFinishOfDockerContainer(t, env, containerID)

		_, stderr, err = env.RunCommand("docker", "start", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		_, stderr, err = env.RunCommand("docker", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")
		requireRecentFinishOfDockerContainer(t, env, containerID)
	})

	t.Run("errors when requested remoteproc name doesn't exist", func(t *testing.T) {
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName("a-processor")
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		_, stderr, err := env.RunCommand(
			"docker", "run", "-d",
			"--runtime", "io.containerd.remoteproc.v1",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", "other-processor"),
			imageName)
		assert.Error(t, err)
		assert.Contains(t, stderr, "remote processor other-processor does not exist, available remote processors: a-processor")
	})

	t.Run("killing process by pid stops the running container", func(t *testing.T) {
		remoteprocName := "another-yolo-docker-device"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		containerID, stderr, err := env.RunCommand(
			"docker", "run", "-d",
			"--runtime", "io.containerd.remoteproc.v1",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		stdout, stderr, err := env.RunCommand("docker", "inspect", "--format={{.State.Pid}}", containerID)
		require.NoError(t, err, "stderr: %s", stderr)
		pid, parseErr := strconv.Atoi(strings.TrimSpace(stdout))
		require.NoError(t, parseErr)

		_, _, err = env.RunCommand("sudo", "kill", "-TERM", fmt.Sprintf("%d", pid))
		require.NoError(t, err)
		remoteproc.RequireState(t, env, sim.DeviceDir(), "offline")
		requireRecentFinishOfDockerContainer(t, env, containerID)
	})
}

func requireDockerContainerFinished(t *testing.T, env testenv.Env, containerID string) {
	t.Helper()

	const retryWindow = 15 * time.Second

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		exitCodeStr, stderr, err := env.RunCommand(
			"docker", "inspect", "--format={{.State.ExitCode}}", containerID,
		)
		require.NoErrorf(c, err, "docker inspect error:\nstderr:\n%s", stderr)

		exitCodeStr = strings.TrimSpace(exitCodeStr)
		exitCode, err := strconv.Atoi(exitCodeStr)
		require.NoErrorf(c, err, "failed to parse ExitCode %q", exitCodeStr)

		require.Equal(c, 0, exitCode)
	}, retryWindow, time.Second)
}

func requireRecentFinishOfDockerContainer(t *testing.T, env testenv.Env, containerID string) {
	t.Helper()

	requireDockerContainerFinished(t, env, containerID)

	const acceptableTimeDelta = 5 * time.Second

	finishedAt, stderr, err := env.RunCommand(
		"docker", "inspect", "--format={{.State.FinishedAt}}", containerID,
	)
	require.NoError(t, err, "docker inspect error:\nstderr:\n%s", stderr)

	raw := strings.TrimSpace(finishedAt)

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	require.NoError(t, err, "failed to parse FinishedAt %q", raw)

	nowStr, _, err := env.RunCommand("date", "-u", "+%Y-%m-%dT%H:%M:%S.%NZ")
	require.NoError(t, err, "failed to get current time from env")
	now, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(nowStr))
	require.NoError(t, err, "failed to parse env time")

	delta := now.Sub(parsed)
	if delta < 0 {
		delta = -delta
	}

	require.LessOrEqual(t, delta, acceptableTimeDelta, "FinishedAt delta exceeds acceptable range: expected <=%v, actual %v", acceptableTimeDelta, delta)
}
