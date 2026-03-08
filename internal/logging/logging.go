package logging

import (
	"log/slog"
	"os"
	"strings"
)

const defaultBufferSize = 1000

// New creates and returns a configured slog.Logger backed by a TeeHandler.
// The returned RingBuffer captures the last 1000 log entries for the API.
//
// Format "json" produces structured JSON output (default, for production).
// Format "text" produces human-readable key=value output (for development).
//
// Level is one of: debug, info, warn, error. Default: info.
func New(level, format string) (*slog.Logger, *RingBuffer) {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	var output slog.Handler
	if strings.ToLower(format) == "text" {
		output = slog.NewTextHandler(os.Stdout, opts)
	} else {
		output = slog.NewJSONHandler(os.Stdout, opts)
	}

	buf := NewRingBuffer(defaultBufferSize)
	handler := NewTeeHandler(output, buf)

	return slog.New(handler), buf
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
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
