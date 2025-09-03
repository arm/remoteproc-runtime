package e2e

import (
	"fmt"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/limavm"
	"github.com/Arm-Debug/remoteproc-runtime/e2e/remoteproc"
	"github.com/Arm-Debug/remoteproc-runtime/e2e/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	shimBin, err := buildShimBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	vm, err := limavm.New(
		rootDir,
		shimBin,
		"../testdata/test-image.tar",
	)
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	containerID, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", remoteprocName),
		"test-image")
	require.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	_, stderr, err = vm.RunCommand("docker", "stop", containerID)
	assert.NoError(t, err, "stderr: %s", stderr)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")
}

func TestRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	shimBin, err := buildShimBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	vm, err := limavm.New(
		rootDir,
		shimBin,
		"../testdata/test-image.tar",
	)
	require.NoError(t, err)
	defer vm.Cleanup()

	sim := remoteproc.NewSimulator(rootDir).WithName("a-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()

	_, stderr, err := vm.RunCommand(
		"docker", "run", "-d",
		"--network=host",
		"--runtime", "io.containerd.remoteproc.v1",
		"--annotation", fmt.Sprintf("remoteproc.name=%s", "other-processor"),
		"test-image")
	assert.Error(t, err)
	assert.Contains(t, stderr, "other-processor is not in the list of available remote processors")
}

func buildShimBinary(binOutDir string, rootPathPrefix string) (string, error) {
	env := map[string]string{
		"GOOS": "linux",
	}
	return repo.BuildBinary("containerd-shim-remoteproc-v1", binOutDir, rootPathPrefix, env)
}
