package userdirs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJoinHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}

	tests := []struct {
		name    string
		inParam []string
		want    string
	}{
		{"no extra", []string{}, home},
		{"single", []string{"foo"}, filepath.Join(home, "foo")},
		{"multiple", []string{"foo", "bar"}, filepath.Join(home, "foo", "bar")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := joinHomeDir(tt.inParam...)
			if err != nil {
				t.Fatalf("joinHomeDir failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("joinHomeDir(%v) = %q, want %q", tt.inParam, got, tt.want)
			}
		})
	}
}

func TestRuntimeDir_XDG(t *testing.T) {
	testDir := os.Getenv("XDG_RUNTIME_DIR")
	if testDir == "" {
		testDir = "/tmp/xdg_runtime_test"
		os.Setenv("XDG_RUNTIME_DIR", testDir)
		defer os.Unsetenv("XDG_RUNTIME_DIR")
	}

	got, err := RuntimeDir()
	if err != nil {
		t.Fatalf("RuntimeDir failed: %v", err)
	}
	if got != testDir {
		t.Errorf("RuntimeDir() = %q, want %q", got, testDir)
	}
}

func TestRuntimeDir_Default(t *testing.T) {
	xdgRuntimeDirOriginal := os.Getenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", xdgRuntimeDirOriginal)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}
	want := filepath.Join(home, "remoteproc-runtime")

	got, err := RuntimeDir()
	if err != nil {
		t.Fatalf("RuntimeDir failed: %v", err)
	}
	if got != want {
		t.Errorf("RuntimeDir() = %q, want %q", got, want)
	}
}
