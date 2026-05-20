package main

import (
	"context"
	"fmt"
	"os"

	"github.com/arm/remoteproc-runtime/internal/shim"
	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
)

func main() {
	manager := shim.NewManager("io.containerd.remoteproc.v1")
	if handled, err := shim.RunStartCompat(context.Background(), manager); handled {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s", manager.Name(), err)
			os.Exit(1)
		}
		return
	}
	containerdshim.RunShim(context.Background(), manager)
}
