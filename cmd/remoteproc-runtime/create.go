package main

import (
	"github.com/Arm-Debug/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	bundlePath string
)

var createCmd = &cobra.Command{
	Use:   "create <ID>",
	Short: "Create a new container from an OCI bundle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		if bundlePath == "" {
			bundlePath = "."
		}
		return runtime.Create(containerID, bundlePath)
	},
}

func init() {
	createCmd.Flags().StringVar(&bundlePath, "bundle", "", "Override the path to the bundle directory (defaults to the current working directory).")
	rootCmd.AddCommand(createCmd)
}
