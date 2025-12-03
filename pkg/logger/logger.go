package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

var (
	globalLogger *slog.Logger
	once         sync.Once
)

// LogLevel can be set via environment variable LOG_LEVEL (debug, info, warn, error)
func init() {
	Init()
}

// Init initializes the global logger.
func Init() {
	once.Do(func() {
		level := slog.LevelInfo
		envLevel := os.Getenv("LOG_LEVEL")
		switch envLevel {
		case "debug", "DEBUG":
			level = slog.LevelDebug
		case "warn", "WARN":
			level = slog.LevelWarn
		case "error", "ERROR":
			level = slog.LevelError
		}

		opts := &slog.HandlerOptions{
			Level: level,
			// Add source info in debug mode
			AddSource: level == slog.LevelDebug,
		}

		// Use JSON handler for structured logging
		handler := slog.NewJSONHandler(os.Stdout, opts)
		globalLogger = slog.New(handler)
		slog.SetDefault(globalLogger)
	})
}

// Get returns the global logger.
func Get() *slog.Logger {
	if globalLogger == nil {
		Init()
	}
	return globalLogger
}

// Info logs at Info level.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Debug logs at Debug level.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Warn logs at Warn level.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs at Error level.
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// With returns a logger with attributes.
func With(args ...any) *slog.Logger {
	return Get().With(args...)
}

// WithContext returns a logger (currently just the global one, but could extract from context).
func WithContext(ctx context.Context) *slog.Logger {
	// In the future, extract correlation ID from context here
	return Get()
}
