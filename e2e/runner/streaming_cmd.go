package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

type StreamingCmd struct {
	cmd    *exec.Cmd
	prefix string
	stdout bytes.Buffer
	stopCh chan struct{}
}

func NewStreamingCmd(cmd *exec.Cmd) *StreamingCmd {
	return &StreamingCmd{
		cmd:    cmd,
		stopCh: make(chan struct{}),
	}
}

func (s *StreamingCmd) WithPrefix(prefix string) *StreamingCmd {
	s.prefix = prefix
	return s
}

func (s *StreamingCmd) Start() error {
	s.cmd.Stdout = &s.stdout

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return err
	}

	go s.streamOutput(stderr)

	return nil
}

func (s *StreamingCmd) Wait() error {
	return s.cmd.Wait()
}

func (s *StreamingCmd) Stop() error {
	if s.cmd.Process != nil {
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}
	return nil
}

func (s *StreamingCmd) Output() string {
	return s.stdout.String()
}

func (s *StreamingCmd) streamOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if s.prefix != "" {
			fmt.Printf("%s: %s\n", s.prefix, scanner.Text())
		} else {
			fmt.Println(scanner.Text())
		}
	}
}
