package main

import (
	"github.com/arm/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <ID>",
	Short: "Start an existing container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		containerID := args[0]
		return runtime.Start(containerID)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
