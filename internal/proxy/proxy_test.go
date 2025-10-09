package proxy

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func stubNamespaceCloneFlags(t *testing.T, fn func(*specs.Spec) (uintptr, error)) {
	t.Helper()
	original := namespaceCloneFlagsFn
	namespaceCloneFlagsFn = fn
	t.Cleanup(func() {
		namespaceCloneFlagsFn = original
	})
}

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

func TestEffectiveNamespaceFlags(t *testing.T) {
	t.Run("returns clone flags when running as root", func(t *testing.T) {
		stubNamespaceCloneFlags(t, func(spec *specs.Spec) (uintptr, error) {
			return 0x1507, nil
		})

		got, err := effectiveNamespaceFlags(true, &specs.Spec{})

		assert.NoError(t, err)
		assert.Equal(t, uintptr(0x1507), got)
	})

	t.Run("warns and does not set namespace if root is not set but flags are given", func(t *testing.T) {
		stubNamespaceCloneFlags(t, func(spec *specs.Spec) (uintptr, error) {
			return 0x2, nil
		})

		stderr := os.Stderr
		reader, writer, err := os.Pipe()
		assert.NoError(t, err)
		os.Stderr = writer
		t.Cleanup(func() {
			os.Stderr = stderr
		})

		got, err := effectiveNamespaceFlags(false, &specs.Spec{})
		assert.NoError(t, err)
		assert.Equal(t, uintptr(0), got)

		assert.NoError(t, writer.Close())
		out, readErr := io.ReadAll(reader)
		assert.NoError(t, readErr)
		assert.Contains(t, string(out), "[WARN] running non-root; namespace isolation disabled")
		assert.NoError(t, reader.Close())
	})

	t.Run("propagates namespaceCloneFlags errors", func(t *testing.T) {
		expected := errors.New("error!")
		stubNamespaceCloneFlags(t, func(spec *specs.Spec) (uintptr, error) {
			return 0, expected
		})

		_, err := effectiveNamespaceFlags(true, &specs.Spec{})
		assert.ErrorIs(t, err, expected)
	})
}
