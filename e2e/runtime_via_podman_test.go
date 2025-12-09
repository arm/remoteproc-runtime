package e2e

import (
	"fmt"
	"path/filepath"
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

	rootpathPrefixInVM := filepath.Join("/tmp", "fake-root")

	runtimeBin, err := repo.BuildRuntimeBin(t.TempDir(), rootpathPrefixInVM, limavm.BinBuildEnv)
	require.NoError(t, err)

	vm, err := limavm.NewPodman()
	require.NoError(t, err)
	defer vm.Cleanup()

	installedRuntimeBin, err := vm.InstallBin(runtimeBin)
	require.NoError(t, err)
	simulatorBin, err := repo.GetRemoteprocSimulator(t.TempDir())
	require.NoError(t, err)

	installedSimulator, err := vm.InstallBin(simulatorBin)
	require.NoError(t, err)

	imageName := "fancy-image"
	require.NoError(t, vm.BuildImage("../testdata", imageName))

	t.Run("basic container lifecycle", func(t *testing.T) {
		remoteprocName := "yolo-podman-device"
		sim := remoteproc.NewSimulator(installedSimulator, rootpathPrefixInVM).WithName(remoteprocName)
		if err := sim.Start(); err != nil {
			t.Fatalf("failed to run simulator: %s", err)
		}
		t.Cleanup(func() {
			_, _, _ = vm.RunCommand("pkill", "-f", "remoteproc-simulator")
			_ = sim.Stop()
		})

		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")

		stdout, stderr, err := vm.RunCommand(
			"podman",
			fmt.Sprintf("--runtime=%s", installedRuntimeBin.Path()),
			"run", "-d",
			"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
			imageName)
		require.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "running")

		containerID := strings.TrimSpace(stdout)
		_, stderr, err = vm.RunCommand("podman", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")

		_, stderr, err = vm.RunCommand("podman", "start", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "running")

		_, stderr, err = vm.RunCommand("podman", "stop", containerID)
		assert.NoError(t, err, "stderr: %s", stderr)
		remoteproc.AssertState(t, vm.VM, sim.DeviceDir(), "offline")
	})
}
