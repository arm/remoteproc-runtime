package assert

import (
	"strings"
	"testing"
)

func ErrorContains(t testing.TB, err error, expectedSubstring string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !strings.Contains(err.Error(), expectedSubstring) {
		t.Errorf("expected error to contain %q, got %q", expectedSubstring, err.Error())
	}
}

func NoError(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("expected no error, got %q", err.Error())
	}
}
