package shim

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type ExitReason int

const (
	ProcessExited ExitReason = iota
	WatcherStopped
)

type ProcessWatcher struct {
	pidfd  int
	stopCh chan struct{}
}

func NewProcessWatcher(pid int) (*ProcessWatcher, error) {
	pidfd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain file descriptor: %w", err)
	}
	return &ProcessWatcher{
		pidfd:  pidfd,
		stopCh: make(chan struct{}),
	}, nil
}

func (pw *ProcessWatcher) StopWatching() {
	pw.stopCh <- struct{}{}
}

func (pw *ProcessWatcher) WaitForExit() ExitReason {
	defer func() {
		_ = unix.Close(pw.pidfd)
	}()
	pfds := []unix.PollFd{
		{Fd: int32(pw.pidfd), Events: unix.POLLIN},
	}

	exitCh := make(chan struct{}, 1)
	go func() {
		// Block indefinitely until process exits
		// If poll fails (err return), something went wrong - assume process exited
		_, _ = unix.Poll(pfds, -1)
		exitCh <- struct{}{}
	}()

	select {
	case <-pw.stopCh:
		return WatcherStopped
	case <-exitCh:
		return ProcessExited
	}
}
