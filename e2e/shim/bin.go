package shim

import (
	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
)

func buildShimBinary(binOutDir string, rootPathPrefix string) (string, error) {
	env := map[string]string{
		"GOOS": "linux",
	}
	return shared.BuildBinary("containerd-shim-remoteproc-v1", binOutDir, rootPathPrefix, env)
}
