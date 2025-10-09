package proxy

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestNamespaceCloneFlags(t *testing.T) {
	t.Run("returns zero when spec is nil", func(t *testing.T) {
		got, err := namespaceCloneFlags(nil)

		assert.NoError(t, err)
		assert.Equal(t, uintptr(0), got)
	})

	t.Run("marks for creation namespaces where paths are not provided", func(t *testing.T) {
		spec := &specs.Spec{
			Linux: &specs.Linux{
				Namespaces: []specs.LinuxNamespace{
					{Type: specs.NetworkNamespace},
					{Type: specs.PIDNamespace, Path: "/proc/self/ns/pid"},
					{Type: specs.UTSNamespace},
				},
			},
		}

		got, err := namespaceCloneFlags(spec)

		assert.NoError(t, err)
		want := uintptr(unix.CLONE_NEWNET | unix.CLONE_NEWUTS)
		assert.Equal(t, want, got)
	})

	t.Run("errors with unknown namespace types", func(t *testing.T) {
		spec := &specs.Spec{
			Linux: &specs.Linux{
				Namespaces: []specs.LinuxNamespace{
					{Type: specs.LinuxNamespaceType("unknown")},
				},
			},
		}

		_, err := namespaceCloneFlags(spec)

		assert.Error(t, err)
		assert.Equal(t, "unknown namespace type \"unknown\"", err.Error())
	})
}
