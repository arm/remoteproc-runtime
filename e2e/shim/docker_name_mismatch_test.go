package shim

import (
	"fmt"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	shimBin, err := buildShimBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	vm, err := NewLimaVM(
		rootDir,
		shimBin,
		"../../testdata/test-image.tar",
	)
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := shared.NewRemoteprocSimulator(rootDir).WithName("a-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	_, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", "other-processor"),
		"test-image")
	assert.Error(t, err)
	assert.Contains(t, stderr, "other-processor is not in the list of available remote processors")
}
