package rootpath_test

import (
	"os"
	"testing"

	"github.com/arm/remoteproc-runtime/internal/rootpath"
	"github.com/stretchr/testify/assert"
)

func TestTildeResolve(t *testing.T) {
	homedir, _ := os.UserHomeDir()

	t.Run("returns absolute homedir path when ~ is provided", func(t *testing.T) {
		got, _ := rootpath.ExpandTilde("~")

		want := homedir

		assert.Equal(t, want, got)
	})

	t.Run("returns absolute path/relative path against ~ when ~ is provided", func(t *testing.T) {
		got, _ := rootpath.ExpandTilde("~/foo/bar")

		want := homedir + "/foo/bar"

		assert.Equal(t, want, got)
	})
}

func TestJoin(t *testing.T) {
	t.Run("returns prefix when no path segments specified", func(t *testing.T) {
		got := rootpath.Join()

		want := "/"
		assert.Equal(t, want, got)
	})

	t.Run("prepends prefix to given path segements", func(t *testing.T) {
		got := rootpath.Join("foo", "bar")

		want := "/foo/bar"
		assert.Equal(t, want, got)
	})
}
