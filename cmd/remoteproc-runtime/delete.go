package main

import (
	"github.com/Arm-Debug/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <ID>",
	Short: "Delete a stopped container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		return runtime.Delete(containerID)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
