//go:build fake_sysfs

package remoteproc

func FindDevicePath(_ string) (string, error) {
	return "/sys/class/remoteproc/remoteproc0", nil
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
