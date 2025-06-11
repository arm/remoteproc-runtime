//go:build fake_sysfs

package remoteproc

func ListMCUs() ([]string, error) {
	return []string{"imx-rproc"}, nil
}
