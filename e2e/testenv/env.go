package testenv

import (
	"os/exec"
	"runtime"
	"testing"
)

type Env interface {
	RunCommand(name string, args ...string) (stdout, stderr string, err error)
	Command(name string, args ...string) *exec.Cmd
	InstallBin(binPath string) (InstalledBin, error)
	CopyDir(hostSrc, envDst string) error
	RemoveAll(path string) error
	ReadFile(path string) (string, error)
	ReadDir(path string) ([]string, error)
	BuildImage(engine, contextDir, imageName string) error
}

func New(t *testing.T) Env {
	t.Helper()
	if runtime.GOOS == "linux" {
		return &hostEnv{}
	}
	if _, err := exec.LookPath("limactl"); err != nil {
		t.Skip("limactl not found, install Lima: https://lima-vm.io/")
	}
	env, err := getOrCreateLimaEnv()
	if err != nil {
		t.Fatalf("failed to create lima env: %s", err)
	}
	return env
}

func BuildEnv() map[string]string {
	if runtime.GOOS == "linux" {
		return nil
	}
	return map[string]string{"GOOS": "linux"}
}

type Runnable interface {
	Run(args ...string) (stdout, stderr string, err error)
}

type InstalledBin struct {
	env       Env
	pathToBin string
}

func (b InstalledBin) Run(args ...string) (stdout, stderr string, err error) {
	return b.env.RunCommand(b.pathToBin, args...)
}

func (b InstalledBin) Command(args ...string) *exec.Cmd {
	return b.env.Command(b.pathToBin, args...)
}

func (b InstalledBin) Path() string {
	return b.pathToBin
}

type Sudo struct {
	env       Env
	pathToBin string
}

func NewSudo(b InstalledBin) Sudo {
	return Sudo(b)
}

func (s Sudo) Run(args ...string) (stdout, stderr string, err error) {
	return s.env.RunCommand("sudo", append([]string{s.pathToBin}, args...)...)
}
