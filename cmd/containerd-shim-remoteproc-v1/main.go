package main

import (
	"context"

	"github.com/Arm-Debug/remoteproc-runtime/internal/shim"
	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
)

func main() {
	manager := shim.NewManager("io.containerd.remoteproc.v1")
	containerdshim.Run(context.Background(), manager)
}
