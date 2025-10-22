package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/arm/remoteproc-runtime/e2e/remoteproc"
	"github.com/arm/remoteproc-runtime/e2e/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodman(t *testing.T) {
	limavm.Require(t)

	rootpathPrefix := t.TempDir()
	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootpathPrefix, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewPodman(rootpathPrefix, runtimeBin)
	require.NoError(t, err)
	defer vm.Cleanup()

	imageName := "fancy-image"
	require.NoError(t, vm.BuildImage("../testdata", imageName))

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-device"
		sim := remoteproc.NewSimulator(rootpathPrefix).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		defer func() { _ = sim.Stop() }()

		remoteproc.AssertState(t, sim.DeviceDir(), "offline")

		stdout, stderr, err := vm.RunCommand(
			"podman",
			"--cgroup-manager=cgroupfs",
			"--runtime=/usr/local/bin/remoteproc-runtime", // <- hardcoding ain't great
			"run", "-d",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "running")

		containerID := strings.TrimSpace(stdout)
		_, stderr, err = vm.RunCommand("podman", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "offline")

		_, stderr, err = vm.RunCommand("podman", "start", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "running")

		_, stderr, err = vm.RunCommand("podman", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, sim.DeviceDir(), "offline")
	})
}
