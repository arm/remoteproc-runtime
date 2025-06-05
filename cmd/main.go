package main

import (
	"context"

	"github.com/Arm-Debug/remoteproc-shim/internal/remoteproc"
	"github.com/containerd/containerd/v2/pkg/shim"
)

func main() {
	shim.Run(context.Background(), remoteproc.NewManager("io.containerd.example.v1"))
}
