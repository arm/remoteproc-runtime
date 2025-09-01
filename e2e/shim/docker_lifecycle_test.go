package shim

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
	"github.com/Arm-Debug/remoteproc-simulator/pkg/simulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerLifecycle(t *testing.T) {
	simConfig := simulator.Config{RootDir: t.TempDir(), Index: 1, Name: "yolo-device"}
	shimBin, err := buildShimBinary(t.TempDir(), simConfig.RootDir)
	require.NoError(t, err)

	vm, err := NewLimaVM(
		simConfig.RootDir,
		shimBin,
		"../../testdata/test-image.tar",
	)
	require.NoError(t, err)
	defer vm.Cleanup()

	sim, err := simulator.NewRemoteproc(simConfig)
	if err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Close()

	deviceDir := filepath.Join(simConfig.RootDir, "sys", "class", "remoteproc", fmt.Sprintf("remoteproc%d", simConfig.Index))
	shared.AssertRemoteprocState(t, deviceDir, "offline")

	containerID, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", simConfig.Name),
		"test-image")
	require.NoError(t, err, "stderr: %s", stderr)
	shared.AssertRemoteprocState(t, deviceDir, "running")

	_, stderr, err = vm.RunCommand("docker", "stop", containerID)
	assert.NoError(t, err, "stderr: %s", stderr)
	shared.AssertRemoteprocState(t, deviceDir, "offline")
}
