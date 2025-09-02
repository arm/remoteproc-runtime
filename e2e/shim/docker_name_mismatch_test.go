package shim

import (
	"fmt"
	"testing"

	"github.com/Arm-Debug/remoteproc-simulator/pkg/simulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteprocNameMismatch(t *testing.T) {
	simConfig := simulator.Config{
		RootDir: t.TempDir(),
		Index:   1,
		Name:    "some-processor",
	}

	shimBin, err := buildShimBinary(t.TempDir(), simConfig.RootDir)
	require.NoError(t, err)

	sim, err := simulator.NewRemoteproc(simConfig)
	if err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}

	vm, err := NewLimaVM(
		simConfig.RootDir,
		shimBin,
		"../../testdata/test-image.tar",
	)
	require.NoError(t, err)
	defer vm.Cleanup()
	defer sim.Close()

	_, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", "other-processor"),
		"test-image")
	assert.Error(t, err)
	assert.Contains(t, stderr, "other-processor is not in the list of available remote processors")
}
