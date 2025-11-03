package remoteproc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/remoteproc-runtime/internal/remoteproc"
	"github.com/stretchr/testify/assert"
)

func TestStoreFirmware(t *testing.T) {
	t.Run("writes firmware into custom directory with unique suffix", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Cleanup(func() {
			_ = os.RemoveAll(tempDir)
		})

		customFirmwareDirDest := filepath.Join(tempDir, "custom_firmware")
		sourcePath := filepath.Join(tempDir, "example.bin")
		wantContent := []byte("test firmware data")
		err := os.WriteFile(sourcePath, wantContent, 0o644)
		assert.NoError(t, err, "failed to write source firmware file")

		gotDestPath, err := remoteproc.StoreFirmware(sourcePath, customFirmwareDirDest)
		assert.NoError(t, err, "StoreFirmware returned unexpected error")

		gotDirectory := filepath.Dir(gotDestPath)
		wantDirectory := customFirmwareDirDest
		assert.Equal(t, wantDirectory, gotDirectory, "StoreFirmware should write to custom firmware directory")

		gotFileName := filepath.Base(gotDestPath)
		assert.Regexp(t, `^example_.+\.bin$`, gotFileName, "generated firmware file name should contain unique suffix")

		gotContent, err := os.ReadFile(gotDestPath)
		assert.NoError(t, err, "failed to read stored firmware file")

		assert.Equal(t, wantContent, gotContent, "stored firmware file content mismatch")
	})
}
