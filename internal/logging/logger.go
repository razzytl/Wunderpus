package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init sets up the global slog logger based on config values.
func Init(level, format, output string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	w := os.Stderr
	if strings.ToLower(output) == "stdout" {
		w = os.Stdout
	}
	// Note: file output can be added later if needed.

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(w, opts)
	} else {
		handler = slog.NewJSONHandler(w, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// WithCorrelation returns a logger with a correlation ID attached.
func WithCorrelation(id string) *slog.Logger {
	return slog.With("correlation_id", id)
}
