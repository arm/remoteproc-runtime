package userdirs_test

import (
	"os/user"
	"path/filepath"
	"testing"

	"github.com/arm/remoteproc-runtime/internal/userdirs"
	"github.com/stretchr/testify/require"
)

func TestRuntimeDir(t *testing.T) {
	t.Run("when XDG_RUNTIME_DIR is set, returns its value", func(t *testing.T) {
		testDir := "/tmp/xdg_runtime_test"
		t.Setenv("XDG_RUNTIME_DIR", testDir)

		got, err := userdirs.RuntimeDir()

		want := filepath.Join(testDir, ".remoteproc-runtime")
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("when XDG_RUNTIME_DIR is not set, defaults to $HOME/.remoteproc-runtime", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", "")
		user, err := user.Current()
		require.NoError(t, err)
		home := user.HomeDir

		got, err := userdirs.RuntimeDir()

		want := filepath.Join(home, ".remoteproc-runtime")
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}
