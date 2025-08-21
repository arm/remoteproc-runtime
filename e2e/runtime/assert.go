package runtime

import (
	"encoding/json"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertContainerStatus(t testing.TB, bin string, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	out, err := invokeRuntime(bin, "state", containerName)
	require.NoError(t, err)
	var state specs.State
	require.NoError(t, json.Unmarshal(out, &state))
	assert.Equal(t, wantStatus, state.Status)
}
