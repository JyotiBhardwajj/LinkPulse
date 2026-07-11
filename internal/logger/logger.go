// Package logger provides structured logging capabilities using slog.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger initializes the global slog structured logger based on environment and level settings.
func InitLogger(env string, levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if strings.ToLower(env) == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Pretty text handler for local development
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
