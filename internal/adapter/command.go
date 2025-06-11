package adapter

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func newCommand(ctx context.Context, containerdAddress, id string) (*exec.Cmd, error) {
	namespace, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	args := []string{
		"-namespace", namespace,
		"-id", id,
		"-address", containerdAddress,
		"-publish-binary", self,
	}
	cmd := exec.Command(self, args...)
	cmd.Dir = cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return cmd, nil
}
