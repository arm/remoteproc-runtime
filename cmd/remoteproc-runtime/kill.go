package main

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/Arm-Debug/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill <ID> [SIGNAL]",
	Short: "Send a signal to the container process",
	Long:  "Send a signal to the container process. Supported signals: TERM (15), KILL (9), INT (2). Default is TERM.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]

		signal := syscall.SIGTERM
		if len(args) > 1 {
			var err error
			signal, err = parseSignal(args[1])
			if err != nil {
				return err
			}
		}

		return runtime.Kill(containerID, signal)
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}

func parseSignal(input string) (syscall.Signal, error) {
	signalStr := strings.ToUpper(input)
	switch signalStr {
	case "KILL", "SIGKILL", "9":
		return syscall.SIGKILL, nil
	case "TERM", "SIGTERM", "15":
		return syscall.SIGTERM, nil
	case "INT", "SIGINT", "2":
		return syscall.SIGINT, nil
	default:
		return 0, fmt.Errorf("unsupported signal: %s (supported: TERM (15), KILL (9), INT (2))", input)
	}
}
