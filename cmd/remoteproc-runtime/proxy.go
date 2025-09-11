package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Arm-Debug/remoteproc-runtime/internal/remoteproc"
	"github.com/spf13/cobra"
)

var (
	devicePath string
)

var proxyCmd = &cobra.Command{
	Use:    "proxy",
	Short:  "Proxy process for managing remoteproc lifecycle",
	Hidden: true, // Internal command, not for direct user interaction
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if devicePath == "" {
			return fmt.Errorf("--device-path is required")
		}

		// Phase 1: Wait for SIGUSR1 start signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGINT)

		sig := <-sigCh
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			os.Exit(0)
		}

		// Phase 2: Start the firmware and wait for its termination or SIGTERM
		if err := remoteproc.Start(devicePath); err != nil {
			return fmt.Errorf("failed to start remoteproc: %w", err)
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case sig := <-sigCh:
				if sig == syscall.SIGTERM || sig == syscall.SIGINT {
					if err := remoteproc.Stop(devicePath); err != nil {
						fmt.Fprintf(os.Stderr, "failed to stop remoteproc: %v\n", err)
					}
					os.Exit(0)
				}
			case <-ticker.C:
				state, err := remoteproc.GetState(devicePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get remoteproc state: %v\n", err)
					continue
				}
				if state != remoteproc.StateRunning {
					fmt.Fprintf(os.Stderr, "remoteproc not running, current state: %s\n", state)
					os.Exit(1)
				}
			}
		}
	},
}

func init() {
	proxyCmd.Flags().StringVar(&devicePath, "device-path", "", "Remoteproc device path (required)")
	proxyCmd.MarkFlagRequired("device-path")
	rootCmd.AddCommand(proxyCmd)
}
