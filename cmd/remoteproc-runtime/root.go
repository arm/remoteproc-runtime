package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "container-runtime",
	Short: "A simple OCI-compliant container runtime using remoteproc",
}

func Execute() error {
	return rootCmd.Execute()
}
