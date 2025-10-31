package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/arm/remoteproc-runtime/internal/log"
	"github.com/arm/remoteproc-runtime/internal/version"
	"github.com/spf13/cobra"
)

var (
	logLevel string
	logger   *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:     "remoteproc-runtime",
	Short:   "An OCI-compliant container runtime using remoteproc",
	Version: fmt.Sprintf("%s (commit: %s)", version.Version, version.GitCommit),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level, err := parseLogLevel(logLevel)
		if err != nil {
			return err
		}
		logger = log.NewLogger(level)
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set the logging level (trace, debug, info, warn, error, fatal, panic)")

	ignoreUnsupportedPodmanFlag(rootCmd)
}

func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level %q, must be one of: debug, info, warn, error", level)
	}
}

// Silently ignore unsupported `--systemd-cgroup` flag.
//
// Podman will automatically pass `--systemd-cgroup` to the runtime's `create`, unless `--cgroup-manager=cgroupfs` is used when invoking Podman.
// Since Remoteproc Runtime does not leverage cgroups at all, we're making user's lifes easier by not requiring them to pass that argument.
func ignoreUnsupportedPodmanFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("systemd-cgroup", false, "")
	cmd.PersistentFlags().MarkHidden("systemd-cgroup")
}
