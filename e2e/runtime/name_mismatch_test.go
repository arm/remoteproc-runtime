package runtime

import (
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	sim := shared.NewRemoteprocSimulator(rootDir).WithName("some-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()
	bin, err := buildRuntimeBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, "other-processor"))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, "test-container")
	assert.ErrorContains(t, err, "other-processor is not in the list of available remote processors")
}
