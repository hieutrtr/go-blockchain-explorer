package util

import (
	"log/slog"
	"os"
	"strings"
)

// GlobalLogger is the application-wide logger instance
var GlobalLogger *slog.Logger

// init initializes the global logger (internal package initialization)
func init() {
	GlobalLogger = NewLogger()
}

// NewLogger creates a new structured JSON logger with level configured from environment
// LOG_LEVEL environment variable: DEBUG, INFO, WARN, ERROR (default: INFO)
func NewLogger() *slog.Logger {
	// Get log level from environment variable
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if levelStr == "" {
		levelStr = "INFO"
	}

	// Parse log level
	var level slog.Level
	switch levelStr {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo // Default to INFO if unrecognized
	}

	// Create JSON handler with specified level
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // Include source file and line number
	})

	// Create and return logger instance
	return slog.New(handler)
}

// Logger methods wrapping GlobalLogger for convenience
// These provide structured logging with key-value pairs

// Info logs an info-level message with attributes
func Info(msg string, attrs ...any) {
	if GlobalLogger != nil {
		GlobalLogger.Info(msg, attrs...)
	}
}

// Warn logs a warning-level message with attributes
func Warn(msg string, attrs ...any) {
	if GlobalLogger != nil {
		GlobalLogger.Warn(msg, attrs...)
	}
}

// Error logs an error-level message with attributes
func Error(msg string, attrs ...any) {
	if GlobalLogger != nil {
		GlobalLogger.Error(msg, attrs...)
	}
}

// Debug logs a debug-level message with attributes
func Debug(msg string, attrs ...any) {
	if GlobalLogger != nil {
		GlobalLogger.Debug(msg, attrs...)
	}
}

// WithContext returns a logger with additional context attributes
// Useful for adding request IDs, trace IDs, or other contextual information
func WithContext(attrs ...any) *slog.Logger {
	if GlobalLogger != nil {
		return GlobalLogger.With(attrs...)
	}
	return GlobalLogger
}
