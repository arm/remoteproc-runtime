package runtime

import (
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
	sim, err := simulator.NewRemoteproc(simConfig)
	if err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Close()
	bin, err := buildRuntimeBinary(t.TempDir(), simConfig.RootDir)
	require.NoError(t, err)

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, "other-processor"))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, "test-container")
	assert.ErrorContains(t, err, "other-processor is not in the list of available remote processors")
}
