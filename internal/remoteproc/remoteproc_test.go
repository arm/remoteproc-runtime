package remoteproc_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/remoteproc-runtime/internal/remoteproc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreFirmware(t *testing.T) {
	t.Run("writes firmware into custom directory with unique suffix", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceFirmwareFile, err := createFirmwareFile(tempDir)
		require.NoError(t, err)
		customFirmwareDestDir := filepath.Join(tempDir, "custom_firmware")

		gotDestPath, err := remoteproc.StoreFirmware(sourceFirmwareFile.path, customFirmwareDestDir)
		require.NoError(t, err)

		gotDirectory := filepath.Dir(gotDestPath)
		assert.Equal(t, customFirmwareDestDir, gotDirectory, "firmware stored in incorrect directory")
		gotFileName := filepath.Base(gotDestPath)
		assert.Regexp(t, `^example_.+\.bin$`, gotFileName, "generated firmware file name should contain unique suffix")
		gotContent, err := os.ReadFile(gotDestPath)
		require.NoError(t, err)
		assert.Equal(t, string(sourceFirmwareFile.content), string(gotContent), "stored firmware content does not match source content")
	})
}

func TestGetCustomFirmwarePath(t *testing.T) {
	t.Run("reads custom firmware path from sysfs", func(t *testing.T) {
		tempDir := t.TempDir()
		fakeCustomPathFile := filepath.Join(tempDir, "firmware_path")
		wantFirmwarePath := "/custom/firmware/path"
		err := os.WriteFile(fakeCustomPathFile, []byte(wantFirmwarePath), 0o644)
		require.NoError(t, err, "failed to write custom firmware path file")

		gotFirmwarePath := remoteproc.GetCustomFirmwarePath(fakeCustomPathFile)
		require.NoError(t, err, "retrieve custom firmware path failed")

		assert.Equal(t, wantFirmwarePath, gotFirmwarePath, "GetCustomFirmwarePath returned incorrect path")
	})
}

type FirmwareFile struct {
	content []byte
	path    string
}

func createFirmwareFile(targetDir string) (FirmwareFile, error) {
	firmwareFilePath := filepath.Join(targetDir, "example.bin")
	firmwareContent := []byte("test firmware data")
	if err := os.WriteFile(firmwareFilePath, firmwareContent, 0o644); err != nil {
		return FirmwareFile{}, fmt.Errorf("failed to create firmware file: %w", err)
	}
	return FirmwareFile{
		content: []byte(firmwareContent),
		path:    firmwareFilePath,
	}, nil
}
