//go:build !fake_sysfs

package devicetree

import (
	"fmt"
	"os"
)

const modelPath = "/sys/firmware/devicetree/base/model"

func GetModel() (string, error) {
	model, err := os.ReadFile(modelPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", modelPath, err)
	}
	return string(model), nil
}
