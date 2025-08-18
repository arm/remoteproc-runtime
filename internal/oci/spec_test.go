package oci_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestReadSpec(t *testing.T) {
	t.Run("it errors if required annotations are missing", func(t *testing.T) {
		bundlePath := generateBundle(t, &specs.Spec{})
		_, err := oci.ReadSpec(bundlePath)

		assert.ErrorContains(t, err, "missing remoteproc.mcu in annotations")
	})

	t.Run("it returns container configuration read from given bundle path", func(t *testing.T) {
		bundlePath := generateBundle(t, &specs.Spec{
			Annotations: map[string]string{
				"remoteproc.mcu": "some-path",
			},
		})
		_, err := oci.ReadSpec(bundlePath)

		assert.NoError(t, err)
	})
}

func generateBundle(t *testing.T, spec *specs.Spec) string {
	t.Helper()
	bundlePath := t.TempDir()
	configData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal spec to JSON: %v", err)
	}
	configPath := filepath.Join(bundlePath, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config.json to %s: %v", configPath, err)
	}
	return bundlePath
}
