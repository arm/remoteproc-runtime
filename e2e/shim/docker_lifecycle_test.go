package shim

import (
	"fmt"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	shimBin, err := buildShimBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	vm, err := NewLimaVM(
		rootDir,
		shimBin,
		"../../testdata/test-image.tar",
	)
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := shared.NewRemoteprocSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	shared.AssertRemoteprocState(t, sim.DeviceDir(), "offline")

	containerID, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
		"test-image")
	require.NoError(t, err, "stderr: %s", stderr)
	shared.AssertRemoteprocState(t, sim.DeviceDir(), "running")

	_, stderr, err = vm.RunCommand("docker", "stop", containerID)
	assert.NoError(t, err, "stderr: %s", stderr)
	shared.AssertRemoteprocState(t, sim.DeviceDir(), "offline")
}
