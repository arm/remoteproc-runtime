package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/arm/remoteproc-runtime/e2e/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodman(t *testing.T) {
	rootpathPrefix := filepath.Join("/tmp", "fake-root")

	env := testenv.New(t)

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootpathPrefix, testenv.BuildEnv())
	require.NoError(t, err)

	installedRuntimeBin, err := env.InstallBin(runtimeBin)
	require.NoError(t, err)

	simulatorBin, err := remoteproc.DownloadSimulator(context.Background())
	require.NoError(t, err)
	installedSimulator, err := env.InstallBin(simulatorBin)
	require.NoError(t, err)

	imageName := "fancy-image"
	require.NoError(t, env.BuildImage("podman", "../testdata", imageName))

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-podman-device"
		sim := remoteproc.NewSimulator(installedSimulator, env, rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

		stdout, stderr, err := env.RunCommand(
			"podman",
			fmt.Sprintf("--runtime=%s", installedRuntimeBin.Path()),
			"run", "-d",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		containerID := strings.TrimSpace(stdout)
		_, stderr, err = env.RunCommand("podman", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")

		_, stderr, err = env.RunCommand("podman", "start", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "running")

		_, stderr, err = env.RunCommand("podman", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, env, sim.DeviceDir(), "offline")
	})
}
