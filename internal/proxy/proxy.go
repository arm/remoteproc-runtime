package proxy

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func NewProcess(logger *slog.Logger, namespaces []specs.LinuxNamespace, devicePath string) (int, error) {
	execPath, err := os.Executable()
	if err != nil {
		return -1, fmt.Errorf("failed to get executable path: %w", err)
	}

	isRoot := os.Geteuid() == 0

	namespaceFlags, err := LinuxCloneFlags(logger, isRoot, namespaces)
	if err != nil {
		return -1, err
	}

	cmd := exec.Command(execPath, "proxy", "--device-path", devicePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:    true,
		Cloneflags: namespaceFlags,
	}

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start proxy process: %w", err)
	}

	return cmd.Process.Pid, nil
}

func StopFirmware(pid int) error {
	return SendSignal(pid, syscall.SIGTERM)
}

func StartFirmware(pid int) error {
	return SendSignal(pid, syscall.SIGUSR1)
}

func SendSignal(pid int, signal syscall.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}
	if err := process.Signal(signal); err != nil {
		return fmt.Errorf("failed to send %s: %w", signal, err)
	}
	return nil
}
