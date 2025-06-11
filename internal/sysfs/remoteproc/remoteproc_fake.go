//go:build fake_sysfs

package remoteproc

func FindMCUDirectory(_ string) (string, error) {
	return "/sys/class/remoteproc/remoteproc0", nil
}

func GetState(_ string) (State, error) {
	return StateOffline, nil
}
