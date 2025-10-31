package proxy_test

import (
	"io"
	"log/slog"
	"testing"

	proxy "github.com/arm/remoteproc-runtime/internal/proxy"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestLinuxCloneFlags(t *testing.T) {
	t.Run("converts known linux namespace flags to unix", func(t *testing.T) {
		isRoot := true
		namespaces := []specs.LinuxNamespace{
			{Type: specs.CgroupNamespace},
			{Type: specs.UserNamespace},
		}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		got, err := proxy.LinuxCloneFlags(logger, isRoot, namespaces)

		require.NoError(t, err)
		want := uintptr(unix.CLONE_NEWCGROUP | unix.CLONE_NEWUSER)
		require.Equal(t, want, got)
	})

	t.Run("non-root disables cloning", func(t *testing.T) {
		isRoot := false
		namespaces := []specs.LinuxNamespace{
			{Type: specs.CgroupNamespace},
			{Type: specs.UserNamespace},
		}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		got, err := proxy.LinuxCloneFlags(logger, isRoot, namespaces)

		require.NoError(t, err)
		require.Equal(t, uintptr(0), got)
	})

	t.Run("errors given unknown namespace", func(t *testing.T) {
		isRoot := true
		namespaces := []specs.LinuxNamespace{
			{Type: specs.LinuxNamespaceType("weird-name")},
		}
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		_, err := proxy.LinuxCloneFlags(logger, isRoot, namespaces)

		require.Error(t, err)
	})
}
