//go:build fake_sysfs

package remoteproc

import "fmt"

func FindDevicePath(mcu string) (string, error) {
	if mcu == "imx-rproc" {
		return "/sys/class/remoteproc/remoteproc0", nil
	}
	availableMCUs := []string{"imx-rproc"}
	return "", fmt.Errorf("%s is not in the list of available mcus %v", mcu, availableMCUs)
}

func GetState(_ string) (State, error) {
	return StateOffline, nil
}

func SetFirmwareAndStart(_, _ string) error {
	return nil
}

func Stop(_ string) error {
	return nil
}
