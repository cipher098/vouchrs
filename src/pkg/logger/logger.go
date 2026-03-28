package logger

import (
	"log/slog"
	"os"
)

// New creates a structured slog.Logger.
// In production (env="production") it outputs JSON; otherwise text.
func New(env string) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
