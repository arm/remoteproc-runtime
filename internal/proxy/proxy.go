package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	cmd *exec.Cmd
	Pid int
}

func NewProcess(devicePath string) (*Process, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(execPath, "proxy", "--device-path", devicePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start proxy process: %w", err)
	}

	return &Process{
		cmd: cmd,
		Pid: cmd.Process.Pid,
	}, nil
}

func FindProcess(pid int) (*Process, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	return &Process{
		Pid: pid,
		cmd: &exec.Cmd{Process: process},
	}, nil
}

func (p *Process) StartFirmware() error {
	return p.SendSignal(syscall.SIGUSR1)
}

func (p *Process) StopFirmware() error {
	return p.SendSignal(syscall.SIGTERM)
}

func (p *Process) SendSignal(signal syscall.Signal) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("proxy process not available")
	}

	if err := p.cmd.Process.Signal(signal); err != nil {
		return fmt.Errorf("failed to send %s to proxy: %w", signal, err)
	}

	return nil
}
