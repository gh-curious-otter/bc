// Package log provides structured logging for bc using log/slog.
package log

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	// mu protects logger and verbose
	mu sync.RWMutex

	// logger is the global logger instance
	logger *slog.Logger

	// verbose controls whether debug-level logs are shown
	verbose bool
)

func init() {
	// Default to info level, text output to stderr
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SetVerbose enables or disables verbose (debug) logging.
func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
	level := slog.LevelInfo
	if v {
		level = slog.LevelDebug
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// SetOutput sets the output writer for the logger.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	logger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
}

// Debug logs a debug message. Only shown when verbose is enabled.
func Debug(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Error(msg, args...)
}

// With returns a logger with the given attributes.
func With(args ...any) *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger.With(args...)
}

// Logger returns the underlying slog.Logger.
func Logger() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}
