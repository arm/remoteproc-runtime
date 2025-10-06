package main

import (
	"github.com/arm/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var forceDelete bool

var deleteCmd = &cobra.Command{
	Use:   "delete <ID>",
	Short: "Delete a container",
	Long:  "Delete a container. Use --force to delete a running container.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		containerID := args[0]
		return runtime.Delete(logger, containerID, forceDelete)
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete a running container")
	rootCmd.AddCommand(deleteCmd)
}
