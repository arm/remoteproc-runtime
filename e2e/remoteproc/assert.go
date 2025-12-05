package remoteproc

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/remoteproc-runtime/e2e/limavm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertState(t testing.TB, vm limavm.VM, deviceDir, wantState string) {
	t.Helper()
	const waitFor = 500 * time.Millisecond
	const tickEvery = 100 * time.Millisecond
	stateFilePath := filepath.Join(deviceDir, "state")
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertFileContent(c, vm, stateFilePath, wantState)
	}, waitFor, tickEvery)
}

func RequireState(t testing.TB, vm limavm.VM, deviceDir, wantState string) {
	t.Helper()
	const waitFor = 500 * time.Millisecond
	const tickEvery = 100 * time.Millisecond
	stateFilePath := filepath.Join(deviceDir, "state")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assertFileContent(c, vm, stateFilePath, wantState)
	}, waitFor, tickEvery)
}

func assertFileContent(t assert.TestingT, vm limavm.VM, path, wantContent string) {
	gotContent, err := vm.ReadFile(path)
	if assert.NoError(t, err) {
		assert.Equal(t, wantContent, gotContent)
	}
}
