// File: pkg/logger/logger.go

package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	// Initialize the default logger
	defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// Info logs an informational message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// WithContext returns a new logger with the given context
func WithContext(ctx map[string]any) *slog.Logger {
	return defaultLogger.With(slog.Group("context", slog.Any("data", ctx)))
}
