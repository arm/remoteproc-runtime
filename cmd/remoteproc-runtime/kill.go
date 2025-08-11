package main

import (
	"github.com/Arm-Debug/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use: "kill <ID>",
	// Short: "Send a signal to the container process",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		return runtime.Kill(containerID)
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}
