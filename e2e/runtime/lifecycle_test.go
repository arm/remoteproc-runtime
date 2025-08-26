package runtime

import (
	"path/filepath"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
	"github.com/Arm-Debug/remoteproc-simulator/pkg/simulator"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
)

func TestContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	deviceName := "fancy-mcu"

	simConfig := simulator.Config{RootDir: rootDir, Index: 1, Name: deviceName}
	sim, err := simulator.NewRemoteproc(simConfig)
	if err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Close()
	deviceDir := filepath.Join(rootDir, "sys", "class", "remoteproc", "remoteproc1")
	bin, err := buildRuntimeBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	containerName := "test-container"

	bundlePath := t.TempDir()
	require.NoError(t, shared.GenerateBundle(bundlePath, deviceName))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateCreated)
	shared.AssertRemoteprocState(t, deviceDir, "offline")

	_, err = invokeRuntime(bin, "start", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateRunning)
	shared.AssertRemoteprocState(t, deviceDir, "running")

	_, err = invokeRuntime(bin, "kill", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateStopped)
	shared.AssertRemoteprocState(t, deviceDir, "offline")

	_, err = invokeRuntime(bin, "delete", containerName)
	require.NoError(t, err)
}
