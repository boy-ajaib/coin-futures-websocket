package server

import (
	"context"
	"log/slog"
	"os"

	"github.com/centrifugal/centrifuge"
)

// SlogLogger wraps a slog.Logger for use with Centrifuge
type SlogLogger struct {
	logger *slog.Logger
	level  centrifuge.LogLevel
}

// NewSlogLogger creates a new slog-based logger wrapper
func NewSlogLogger(logger *slog.Logger, level centrifuge.LogLevel) centrifuge.LogHandler {
	sl := &SlogLogger{
		logger: logger,
		level:  level,
	}

	// Return a function that implements centrifuge.LogHandler
	return func(entry centrifuge.LogEntry) {
		sl.handle(entry)
	}
}

// handle processes a log entry
func (sl *SlogLogger) handle(entry centrifuge.LogEntry) {
	// Map Centrifuge log levels to slog levels
	var level slog.Level
	switch entry.Level {
	case centrifuge.LogLevelDebug:
		level = slog.LevelDebug
	case centrifuge.LogLevelInfo:
		level = slog.LevelInfo
	case centrifuge.LogLevelWarn:
		level = slog.LevelWarn
	case centrifuge.LogLevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Only log if the entry level is at or above our configured level
	if !sl.enabled(entry.Level) {
		return
	}

	sl.logger.Log(context.Background(), level, entry.Message)
}

// enabled returns true if the given log level should be logged
func (sl *SlogLogger) enabled(level centrifuge.LogLevel) bool {
	levelOrder := map[centrifuge.LogLevel]int{
		centrifuge.LogLevelDebug: 0,
		centrifuge.LogLevelInfo:  1,
		centrifuge.LogLevelWarn:  2,
		centrifuge.LogLevelError: 3,
	}
	return levelOrder[level] >= levelOrder[sl.level]
}

// NewProductionLogHandler creates a log handler suitable for production environments
// Uses JSON output format
func NewProductionLogHandler(logLevel string) centrifuge.LogHandler {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return NewFormattedLogHandler(logger, logLevel)
}

// NewDevelopmentLogHandler creates a log handler suitable for development environments
// Uses text output format for better readability
func NewDevelopmentLogHandler(logLevel string) centrifuge.LogHandler {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return NewFormattedLogHandler(logger, logLevel)
}

// NewFormattedLogHandler creates a log handler from a slog.Logger
func NewFormattedLogHandler(logger *slog.Logger, logLevel string) centrifuge.LogHandler {
	// Map string log level to Centrifuge log level
	var level centrifuge.LogLevel
	switch logLevel {
	case "debug":
		level = centrifuge.LogLevelDebug
	case "info":
		level = centrifuge.LogLevelInfo
	case "warn":
		level = centrifuge.LogLevelWarn
	case "error":
		level = centrifuge.LogLevelError
	default:
		level = centrifuge.LogLevelInfo
	}

	return NewSlogLogger(logger, level)
}
