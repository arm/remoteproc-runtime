package main

import (
	"encoding/json"
	"fmt"

	"github.com/arm/remoteproc-runtime/internal/runtime"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state <ID>",
	Short: "Get the state of a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		containerID := args[0]

		state, err := runtime.State(containerID)
		if err != nil {
			return err
		}

		output, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stateCmd)
}
