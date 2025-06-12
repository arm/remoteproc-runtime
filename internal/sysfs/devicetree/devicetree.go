//go:build !fake_sysfs

package devicetree

import (
	"fmt"
	"os"
	"strings"
)

const modelPath = "/sys/firmware/devicetree/base/model"

func GetModel() (string, error) {
	return readDeviceTreeFile(modelPath)
}

func readDeviceTreeFile(path string) (string, error) {
	model, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}
	return strings.TrimRight(string(model), "\x00"), nil
}
