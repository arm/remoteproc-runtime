package runtime

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/shared"
)

func buildRuntimeBinary(binOutDir string, rootPathPrefix string) (string, error) {
	return shared.BuildBinary("remoteproc-runtime", binOutDir, rootPathPrefix)
}

func invokeRuntime(bin string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w\n<stderr>\n%s\n</stderr>", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
