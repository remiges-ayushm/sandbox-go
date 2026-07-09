// Package logging configures the process-wide structured logger used across the app.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger (JSON, leveled) from the LOG_LEVEL env var
// (debug|info|warn|error, default info) so every package logs consistently without
// needing a logger passed around explicitly.
func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     parseLevel(os.Getenv("LOG_LEVEL")),
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler))
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
