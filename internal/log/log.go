package log

import (
	"log/slog"
	"os"
)

func NewLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)
	return logger
}
