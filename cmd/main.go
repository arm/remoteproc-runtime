package main

import (
	"context"

	"github.com/Arm-Debug/remoteproc-shim/internal/remoteproc"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/log"
)

func main() {
	// TODO: Set this conditionally
	log.SetLevel(log.DebugLevel.String())

	manager := remoteproc.NewManager("io.containerd.remoteproc.v1")
	shim.Run(context.Background(), manager)
}
