package remoteproc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertState(t testing.TB, deviceDir string, wantState string) {
	t.Helper()
	const waitFor = 500 * time.Millisecond
	const tickEvery = 100 * time.Millisecond
	stateFilePath := filepath.Join(deviceDir, "state")
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertFileContent(c, stateFilePath, wantState)
	}, waitFor, tickEvery)
}

func RequireState(t testing.TB, deviceDir string, wantState string) {
	t.Helper()
	const waitFor = 500 * time.Millisecond
	const tickEvery = 100 * time.Millisecond
	stateFilePath := filepath.Join(deviceDir, "state")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assertFileContent(c, stateFilePath, wantState)
	}, waitFor, tickEvery)
}

func assertFileContent(t assert.TestingT, path string, wantContent string) {
	gotContent, err := os.ReadFile(path)
	if assert.NoError(t, err) {
		assert.Equal(t, wantContent, string(gotContent))
	}
}
