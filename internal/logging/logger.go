package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey string

const correlationIDKey ctxKey = "correlation_id"

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

// ContextWithCorrelation returns a new context with the correlation ID.
func ContextWithCorrelation(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// L returns a context-aware logger.
func L(ctx context.Context) *slog.Logger {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return slog.With("correlation_id", id)
	}
	return slog.Default()
}
