package rootpath_test

import (
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/internal/rootpath"
	"github.com/stretchr/testify/assert"
)

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
