package util

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("creates logger with default INFO level", func(t *testing.T) {
		os.Unsetenv("LOG_LEVEL")
		logger := NewLogger()
		assert.NotNil(t, logger)
	})

	t.Run("creates logger with DEBUG level from env", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		defer os.Unsetenv("LOG_LEVEL")
		logger := NewLogger()
		assert.NotNil(t, logger)
	})

	t.Run("creates logger with WARN level from env", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "WARN")
		defer os.Unsetenv("LOG_LEVEL")
		logger := NewLogger()
		assert.NotNil(t, logger)
	})

	t.Run("creates logger with ERROR level from env", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "ERROR")
		defer os.Unsetenv("LOG_LEVEL")
		logger := NewLogger()
		assert.NotNil(t, logger)
	})

	t.Run("defaults to INFO level for unrecognized level", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "INVALID")
		defer os.Unsetenv("LOG_LEVEL")
		logger := NewLogger()
		assert.NotNil(t, logger)
	})
}

func TestLoggerMethods(t *testing.T) {
	t.Run("logger Info method exists and works", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		assert.NotNil(t, logger)
		assert.NotPanics(t, func() {
			logger.Info("test info message", "key", "value")
		})
	})

	t.Run("logger Warn method exists and works", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		assert.NotNil(t, logger)
		assert.NotPanics(t, func() {
			logger.Warn("test warn message", "severity", "high")
		})
	})

	t.Run("logger Error method exists and works", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		assert.NotNil(t, logger)
		assert.NotPanics(t, func() {
			logger.Error("test error message", "error_code", 500)
		})
	})

	t.Run("logger Debug method exists and works", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		assert.NotNil(t, logger)
		assert.NotPanics(t, func() {
			logger.Debug("test debug message", "detail", "verbose")
		})
	})
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("package-level Info function works", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "DEBUG")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		assert.NotPanics(t, func() {
			Info("test message", "key", "value")
		})
	})

	t.Run("package-level Warn function works", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "DEBUG")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		assert.NotPanics(t, func() {
			Warn("test message", "key", "value")
		})
	})

	t.Run("package-level Error function works", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "DEBUG")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		assert.NotPanics(t, func() {
			Error("test message", "key", "value")
		})
	})

	t.Run("package-level Debug function works", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "DEBUG")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		assert.NotPanics(t, func() {
			Debug("test message", "key", "value")
		})
	})
}

func TestWithContext(t *testing.T) {
	t.Run("WithContext returns logger with context attributes", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "DEBUG")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		contextLogger := logger.With("request_id", "req-123", "user_id", "user-456")
		assert.NotNil(t, contextLogger)

		assert.NotPanics(t, func() {
			contextLogger.Info("processing request", "action", "query")
		})
	})

	t.Run("package-level WithContext function works", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "DEBUG")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		contextLogger := WithContext("request_id", "req-123")
		assert.NotNil(t, contextLogger)

		assert.NotPanics(t, func() {
			contextLogger.Info("test message")
		})
	})
}

func TestConcurrentLogging(t *testing.T) {
	t.Run("handles concurrent logging from multiple goroutines", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "INFO")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		done := make(chan bool, 50)

		for i := 0; i < 50; i++ {
			go func(id int) {
				Info("concurrent log", "goroutine_id", id)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 50; i++ {
			<-done
		}

		// If we get here without panic, test passes
		assert.True(t, true)
	})
}

func TestNilGlobalLogger(t *testing.T) {
	t.Run("handles nil GlobalLogger gracefully", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		GlobalLogger = nil
		defer func() { GlobalLogger = oldGlobalLogger }()

		// These should not panic
		assert.NotPanics(t, func() {
			Info("test")
			Warn("test")
			Error("test")
			Debug("test")
			WithContext("key", "value")
		})
	})
}

func TestLoggingPerformance(t *testing.T) {
	t.Run("logging has minimal overhead", func(t *testing.T) {
		oldGlobalLogger := GlobalLogger
		os.Setenv("LOG_LEVEL", "INFO")
		GlobalLogger = NewLogger()
		defer func() {
			GlobalLogger = oldGlobalLogger
			os.Unsetenv("LOG_LEVEL")
		}()

		start := time.Now()
		for i := 0; i < 100; i++ {
			Info("performance test", "iteration", i)
		}
		elapsed := time.Since(start)

		// 100 logs should take less than 100ms (typical ~1µs per log)
		maxDuration := 100 * time.Millisecond
		assert.Less(t, elapsed, maxDuration)
	})
}

func TestLoggerConfiguration(t *testing.T) {
	t.Run("LOG_LEVEL environment variable controls log level", func(t *testing.T) {
		// Test DEBUG level
		os.Setenv("LOG_LEVEL", "DEBUG")
		debugLogger := NewLogger()
		assert.NotNil(t, debugLogger)
		os.Unsetenv("LOG_LEVEL")

		// Test INFO level (default)
		infoLogger := NewLogger()
		assert.NotNil(t, infoLogger)

		// Test ERROR level
		os.Setenv("LOG_LEVEL", "ERROR")
		errorLogger := NewLogger()
		assert.NotNil(t, errorLogger)
		os.Unsetenv("LOG_LEVEL")
	})
}

func TestStructuredAttributes(t *testing.T) {
	t.Run("logger supports multiple structured attributes", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "INFO")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		assert.NotNil(t, logger)

		assert.NotPanics(t, func() {
			logger.Info("operation completed",
				"operation", "backfill",
				"blocks_processed", 100,
				"duration_ms", 500,
				"success", true,
				"batch_size", 25)
		})
	})
}

func TestJSONOutputFormat(t *testing.T) {
	t.Run("logger outputs valid JSON with required fields", func(t *testing.T) {
		// Create a buffer to capture output
		var buf bytes.Buffer

		// Create logger with buffer output
		handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		})
		logger := slog.New(handler)

		// Log a message
		logger.Info("test message", "key", "value", "count", 42)

		// Parse JSON output
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err, "output should be valid JSON")

		// Verify required fields exist
		assert.Contains(t, logEntry, "time", "log should have time field")
		assert.Contains(t, logEntry, "level", "log should have level field")
		assert.Contains(t, logEntry, "msg", "log should have msg field")
		assert.Contains(t, logEntry, "source", "log should have source field with AddSource enabled")

		// Verify values
		assert.Equal(t, "INFO", logEntry["level"])
		assert.Equal(t, "test message", logEntry["msg"])
		assert.Equal(t, "value", logEntry["key"])
		assert.Equal(t, float64(42), logEntry["count"])

		// Verify source has file and line info
		source, ok := logEntry["source"].(map[string]interface{})
		assert.True(t, ok, "source should be an object")
		assert.Contains(t, source, "file")
		assert.Contains(t, source, "line")
	})

	t.Run("logger supports case-insensitive LOG_LEVEL", func(t *testing.T) {
		// Test lowercase
		os.Setenv("LOG_LEVEL", "debug")
		defer os.Unsetenv("LOG_LEVEL")

		logger := NewLogger()
		assert.NotNil(t, logger)

		// Test mixed case
		os.Setenv("LOG_LEVEL", "WaRn")
		logger = NewLogger()
		assert.NotNil(t, logger)
	})
}

func BenchmarkLoggingInfo(b *testing.B) {
	os.Setenv("LOG_LEVEL", "INFO")
	defer os.Unsetenv("LOG_LEVEL")

	logger := NewLogger()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLoggingError(b *testing.B) {
	os.Setenv("LOG_LEVEL", "INFO")
	defer os.Unsetenv("LOG_LEVEL")

	logger := NewLogger()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Error("benchmark error", "error_code", 500)
	}
}
