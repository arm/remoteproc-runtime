package userdirs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/remoteproc-runtime/internal/userdirs"
)

func TestRuntimeDir(t *testing.T) {
	t.Run("when XDG_RUNTIME_DIR is set, returns its value", testRuntimeDir_XDGSet)
	t.Run("when XDG_RUNTIME_DIR is not set, defaults to $HOME/.foo-bar", testRuntimeDir_XDGUnset)
}

func testRuntimeDir_XDGSet(t *testing.T) {
	testDir := "/tmp/xdg_runtime_test"
	cleanup, err := overrideEnv("XDG_RUNTIME_DIR", testDir)
	if err != nil {
		t.Fatalf("overrideEnv failed: %v", err)
	}
	defer cleanup()

	want := filepath.Join(testDir, "remoteproc-runtime")
	got, err := userdirs.RuntimeDir()
	if err != nil {
		t.Fatalf("RuntimeDir failed: %v", err)
	}
	if got != want {
		t.Errorf("RuntimeDir() = %q, want %q", got, want)
	}
}

func overrideEnv(t *testing.T, key, value string) (func(), error) {
	originalValue, hadOriginal := os.LookupEnv(key)
	err := os.Setenv(key, value)
	if err != nil {
		t.Fatalf("os.Setenv failed: %v", err)
	}
	return func() {
		var err error
		if hadOriginal {
			err = os.Setenv(key, originalValue)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("env restore failed: %v", err)
		}
	}, nil
}

func testRuntimeDir_XDGUnset(t *testing.T) {
	xdgRuntimeDirOriginal := os.Getenv("XDG_RUNTIME_DIR")
	cleanup, err := overrideEnv("XDG_RUNTIME_DIR", "")
	if err != nil {
		t.Fatalf("overrideEnv failed: %v", err)
	}
	defer cleanup()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}

	want := filepath.Join(home, "remoteproc-runtime")
	got, err := userdirs.RuntimeDir()
	if err != nil {
		t.Fatalf("RuntimeDir failed: %v", err)
	}
	if got != want {
		t.Errorf("RuntimeDir() = %q, want %q", got, want)
	}
}
