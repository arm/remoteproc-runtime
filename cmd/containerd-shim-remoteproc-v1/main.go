package main

import (
	"context"

	"github.com/Arm-Debug/remoteproc-runtime/internal/adapter"
	"github.com/containerd/containerd/v2/pkg/shim"
)

func main() {
	manager := adapter.NewManager("io.containerd.remoteproc.v1")
	shim.Run(context.Background(), manager)
}
