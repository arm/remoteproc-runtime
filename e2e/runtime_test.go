package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Arm-Debug/remoteproc-runtime/e2e/remoteproc"
	"github.com/Arm-Debug/remoteproc-runtime/e2e/repo"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeContainerLifecycle(t *testing.T) {
	rootDir := t.TempDir()
	remoteprocName := "yolo-device"
	sim := remoteproc.NewSimulator(rootDir).WithName(remoteprocName)
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()
	bin, err := buildRuntimeBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	containerName := "test-container"

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, remoteprocName))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateCreated)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	_, err = invokeRuntime(bin, "start", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateRunning)
	remoteproc.AssertState(t, sim.DeviceDir(), "running")

	_, err = invokeRuntime(bin, "kill", containerName)
	require.NoError(t, err)
	assertContainerStatus(t, bin, containerName, specs.StateStopped)
	remoteproc.AssertState(t, sim.DeviceDir(), "offline")

	_, err = invokeRuntime(bin, "delete", containerName)
	require.NoError(t, err)
}

func TestRuntimeRemoteprocNameMismatch(t *testing.T) {
	rootDir := t.TempDir()
	sim := remoteproc.NewSimulator(rootDir).WithName("some-processor")
	if err := sim.Start(); err != nil {
		t.Fatalf("failed to run simulator: %s", err)
	}
	defer sim.Stop()
	bin, err := buildRuntimeBinary(t.TempDir(), rootDir)
	require.NoError(t, err)

	bundlePath := t.TempDir()
	require.NoError(t, generateBundle(bundlePath, "other-processor"))
	_, err = invokeRuntime(bin, "create", "--bundle", bundlePath, "test-container")
	assert.ErrorContains(t, err, "other-processor is not in the list of available remote processors")
}

func assertContainerStatus(t testing.TB, bin string, containerName string, wantStatus specs.ContainerState) {
	t.Helper()
	out, err := invokeRuntime(bin, "state", containerName)
	require.NoError(t, err)
	var state specs.State
	require.NoError(t, json.Unmarshal(out, &state))
	assert.Equal(t, wantStatus, state.Status)
}

func generateBundle(targetDir string, remoteprocName string) error {
	const bundleRoot = "rootfs"
	const firmwareName = "hello_world.elf"

	if err := os.MkdirAll(filepath.Join(targetDir, bundleRoot), 0755); err != nil {
		return err
	}
	firmwarePath := filepath.Join(targetDir, bundleRoot, firmwareName)
	if err := os.WriteFile(firmwarePath, []byte("pretend binary"), 0644); err != nil {
		return err
	}

	spec := &specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			User: specs.User{UID: 0, GID: 0},
			Args: []string{firmwareName},
			Cwd:  "/",
		},
		Root: &specs.Root{Path: bundleRoot},
		Annotations: map[string]string{
			"remoteproc.name": remoteprocName,
		},
	}

	specPath := filepath.Join(targetDir, "config.json")
	specData, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, specData, 0644); err != nil {
		return err
	}
	return nil
}

func buildRuntimeBinary(binOutDir string, rootPathPrefix string) (string, error) {
	return repo.BuildBinary("remoteproc-runtime", binOutDir, rootPathPrefix, nil)
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
