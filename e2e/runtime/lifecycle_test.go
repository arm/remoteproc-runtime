package runtime

import (
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestRuntimeContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	sim := shared.NewRemoteprocSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()
	bin, err := buildRuntimeBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	containerName := "test-container"

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, remoteprocName))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateCreated)
	shared.AssertRemoteprocState(t, sim.DeviceDir(), "offline")

	_, err = invokeRuntime(bin, "start", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateRunning)
	shared.AssertRemoteprocState(t, sim.DeviceDir(), "running")

	_, err = invokeRuntime(bin, "kill", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateStopped)
	shared.AssertRemoteprocState(t, sim.DeviceDir(), "offline")

	_, err = invokeRuntime(bin, "delete", containerName)
	require.NoError(t, err)
}
