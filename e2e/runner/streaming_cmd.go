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

func (s *StreamingCmd) Start(output io.Writer) error {
	s.cmd.Stdout = &s.stdout

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return err
	}

	go s.streamOutput(stderr, output)

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

func (s *StreamingCmd) streamOutput(reader io.Reader, output io.Writer) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		text := scanner.Text()
		var writable string
		if s.prefix != "" {
			writable = fmt.Sprintf("%s: %s", s.prefix, text)
		} else {
			writable = text
		}
		s.writeOutput(writable, output)
		fmt.Println(writable)
	}
}

func (s *StreamingCmd) writeOutput(line string, output io.Writer) {
	if output != nil {
		_, err := fmt.Fprintf(output, "%s\n", line)
		if err != nil {
			fmt.Printf("failed to write to output: %v\n", err)
		}
	}
}
