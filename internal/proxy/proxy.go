package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var namespaceFlags = map[specs.LinuxNamespaceType]uintptr{
	specs.CgroupNamespace:  unix.CLONE_NEWCGROUP,
	specs.IPCNamespace:     unix.CLONE_NEWIPC,
	specs.MountNamespace:   unix.CLONE_NEWNS,
	specs.NetworkNamespace: unix.CLONE_NEWNET,
	specs.PIDNamespace:     unix.CLONE_NEWPID,
	specs.TimeNamespace:    unix.CLONE_NEWTIME,
	specs.UserNamespace:    unix.CLONE_NEWUSER,
	specs.UTSNamespace:     unix.CLONE_NEWUTS,
}

var getEUID = os.Geteuid

func namespaceCloneFlags(spec *specs.Spec) (uintptr, error) {
	if spec == nil {
		return 0, nil
	}

	var flags uintptr
	for _, ns := range spec.Linux.Namespaces {
		if ns.Path != "" {
			continue
		}
		flag, ok := namespaceFlags[ns.Type]
		if !ok {
			return 0, fmt.Errorf("Unknown namespace type %q", ns.Type)
		}
		flags |= flag
	}
	return flags, nil
}

func effectiveNamespaceFlags(spec *specs.Spec) (uintptr, error) {
	flags, err := namespaceCloneFlags(spec)
	if err != nil {
		return 0, err
	}

	if getEUID() != 0 {
		if flags != 0 {
			fmt.Fprintln(os.Stderr, "[WARN] running without root; namespace isolation disabled")
		}
		return 0, nil
	}

	return flags, nil
}

func NewProcess(spec *specs.Spec, devicePath string) (int, error) {
	execPath, err := os.Executable()
	if err != nil {
		return -1, fmt.Errorf("failed to get executable path: %w", err)
	}

	namespaceFlags, err := effectiveNamespaceFlags(spec)
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
