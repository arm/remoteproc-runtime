//go:build fake_sysfs

package devicetree

func GetModel() (string, error) {
	return "NXP i.MX93 11X11 FRDM board", nil
}
