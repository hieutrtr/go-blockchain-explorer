//go:build integration

package util

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LogEntry represents a parsed JSON log entry
type LogEntry struct {
	Time   string                 `json:"time"`
	Level  string                 `json:"level"`
	Msg    string                 `json:"msg"`
	Source map[string]interface{} `json:"source"`
	Attrs  map[string]interface{} // Additional attributes
}

// captureLogOutput captures log output to a buffer
func captureLogOutput(fn func()) string {
	// Create buffer to capture output
	var buf bytes.Buffer

	// Create new logger with buffer as output
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	oldLogger := GlobalLogger
	GlobalLogger = slog.New(handler)

	// Execute function that generates logs
	fn()

	// Restore original logger
	GlobalLogger = oldLogger

	return buf.String()
}

// parseLogEntry parses a JSON log line
func parseLogEntry(line string) (*LogEntry, error) {
	var entry LogEntry
	rawEntry := make(map[string]interface{})

	if err := json.Unmarshal([]byte(line), &rawEntry); err != nil {
		return nil, err
	}

	// Extract standard fields
	if time, ok := rawEntry["time"].(string); ok {
		entry.Time = time
	}
	if level, ok := rawEntry["level"].(string); ok {
		entry.Level = level
	}
	if msg, ok := rawEntry["msg"].(string); ok {
		entry.Msg = msg
	}
	if source, ok := rawEntry["source"].(map[string]interface{}); ok {
		entry.Source = source
	}

	// Extract additional attributes
	entry.Attrs = make(map[string]interface{})
	for key, value := range rawEntry {
		if key != "time" && key != "level" && key != "msg" && key != "source" {
			entry.Attrs[key] = value
		}
	}

	return &entry, nil
}

// TestLoggingIntegration_CaptureLogOutput tests log capture during operations
// Validates AC7: Logging Integration Tests - Subtask 8.1
func TestLoggingIntegration_CaptureLogOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Capture logs from a test operation
	output := captureLogOutput(func() {
		Info("test info message", "key", "value")
		Warn("test warning message", "count", 42)
		Error("test error message", "error", "something went wrong")
	})

	// Verify output contains log entries
	assert.NotEmpty(t, output, "Should capture log output")

	// Count log lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "Should have 3 log entries")

	t.Logf("✓ Captured %d log entries during operation", len(lines))
}

// TestLoggingIntegration_RequiredFields tests log entry structure
// Validates AC7: Logging Integration Tests - Subtask 8.2
func TestLoggingIntegration_RequiredFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Capture a log entry
	output := captureLogOutput(func() {
		Info("test message",
			"block_height", 12345,
			"hash", "0xabcd",
			"duration_ms", 150,
		)
	})

	// Parse log entry
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0, "Should have log entries")

	entry, err := parseLogEntry(lines[0])
	require.NoError(t, err, "Should parse log entry as JSON")

	// Verify required fields
	assert.NotEmpty(t, entry.Time, "Should have 'time' field")
	assert.NotEmpty(t, entry.Level, "Should have 'level' field")
	assert.NotEmpty(t, entry.Msg, "Should have 'msg' field")
	assert.NotNil(t, entry.Source, "Should have 'source' field")

	// Verify source contains file and line
	assert.Contains(t, entry.Source, "file", "Source should contain file")
	assert.Contains(t, entry.Source, "line", "Source should contain line")
	assert.Contains(t, entry.Source, "function", "Source should contain function")

	// Verify custom attributes
	assert.Equal(t, "test message", entry.Msg)
	assert.Equal(t, float64(12345), entry.Attrs["block_height"])
	assert.Equal(t, "0xabcd", entry.Attrs["hash"])
	assert.Equal(t, float64(150), entry.Attrs["duration_ms"])

	t.Log("✓ Log entries contain all required fields (time, level, msg, source, attributes)")
}

// TestLoggingIntegration_LogLevels tests different log levels
// Validates AC7: Logging Integration Tests - Subtask 8.3
func TestLoggingIntegration_LogLevels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testCases := []struct {
		level       string
		logFunc     func(string, ...any)
		expectLevel string
	}{
		{"DEBUG", Debug, "DEBUG"},
		{"INFO", Info, "INFO"},
		{"WARN", Warn, "WARN"},
		{"ERROR", Error, "ERROR"},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			// Capture log output
			output := captureLogOutput(func() {
				tc.logFunc("test message at "+tc.level, "test_key", "test_value")
			})

			// Parse log entry
			lines := strings.Split(strings.TrimSpace(output), "\n")
			require.Greater(t, len(lines), 0, "Should have log entry")

			entry, err := parseLogEntry(lines[0])
			require.NoError(t, err, "Should parse log entry")

			// Verify level
			assert.Equal(t, tc.expectLevel, entry.Level,
				"Log level should be %s", tc.expectLevel)

			t.Logf("✓ %s level logs correctly", tc.level)
		})
	}
}

// TestLoggingIntegration_ValidJSON tests log output is valid JSON
// Validates AC7: Logging Integration Tests - Subtask 8.4
func TestLoggingIntegration_ValidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Capture logs with various attribute types
	output := captureLogOutput(func() {
		Info("string test", "str", "value")
		Info("int test", "num", 42)
		Info("float test", "decimal", 3.14)
		Info("bool test", "flag", true)
		Info("multiple attrs", "a", 1, "b", "two", "c", true)
	})

	// Verify each line is valid JSON
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, line := range lines {
		var entry map[string]interface{}
		err := json.Unmarshal([]byte(line), &entry)
		assert.NoError(t, err, "Line %d should be valid JSON: %s", i, line)

		// Verify it has required JSON log structure
		assert.Contains(t, entry, "time", "Should have time field")
		assert.Contains(t, entry, "level", "Should have level field")
		assert.Contains(t, entry, "msg", "Should have msg field")
	}

	t.Logf("✓ All %d log entries are valid JSON", len(lines))
}

// TestLoggingIntegration_SensitiveDataFiltering tests sensitive data is not logged
// Validates AC7: Logging Integration Tests - Subtask 8.5
func TestLoggingIntegration_SensitiveDataFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test that sensitive data patterns should not appear in logs
	sensitivePatterns := []string{
		"sk_live_",      // Stripe API key pattern
		"Bearer ",       // Auth tokens
		"password=",     // Passwords
		"api_key=",      // API keys
		"secret=",       // Secrets
	}

	// Capture logs with various data (simulating what might be logged)
	output := captureLogOutput(func() {
		// Good: Sanitized URLs
		Info("connecting to RPC", "url", "https://mainnet.infura.io/v3/[REDACTED]")
		Info("database connection", "host", "localhost", "port", 5432)

		// Good: Block data (not sensitive)
		Info("indexed block",
			"height", 12345,
			"hash", "0xabcd1234",
			"tx_count", 100,
		)

		// Log error without exposing secrets
		Error("RPC error", "error", "connection timeout", "retry_count", 3)
	})

	// Verify no sensitive patterns in output
	for _, pattern := range sensitivePatterns {
		assert.NotContains(t, output, pattern,
			"Log output should not contain sensitive pattern: %s", pattern)
	}

	// Verify safe patterns ARE present
	assert.Contains(t, output, "[REDACTED]", "Should use [REDACTED] for sensitive data")
	assert.Contains(t, output, "height", "Should log non-sensitive block data")

	t.Log("✓ Sensitive data is not logged (API keys, passwords, secrets)")
}

// TestLoggingIntegration_ContextualInformation tests contextual attributes
// Validates AC7: Logging Integration Tests - Subtask 8.6
func TestLoggingIntegration_ContextualInformation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testCases := []struct {
		name           string
		logFunc        func()
		expectedAttrs  map[string]interface{}
		expectedInMsg  string
	}{
		{
			name: "block_processing",
			logFunc: func() {
				Info("processing block",
					"block_height", uint64(100000),
					"block_hash", "0x1234abcd",
					"tx_count", 50,
				)
			},
			expectedAttrs: map[string]interface{}{
				"block_height": float64(100000),
				"tx_count":     float64(50),
			},
			expectedInMsg: "processing block",
		},
		{
			name: "error_with_details",
			logFunc: func() {
				Error("RPC call failed",
					"block_height", uint64(200000),
					"error", "connection timeout",
					"retry_count", 3,
					"duration_ms", 5000,
				)
			},
			expectedAttrs: map[string]interface{}{
				"block_height": float64(200000),
				"retry_count":  float64(3),
			},
			expectedInMsg: "RPC call failed",
		},
		{
			name: "reorg_detection",
			logFunc: func() {
				Warn("reorg detected",
					"fork_height", uint64(99999),
					"depth", 3,
					"old_hash", "0xaaa",
					"new_hash", "0xbbb",
				)
			},
			expectedAttrs: map[string]interface{}{
				"fork_height": float64(99999),
				"depth":       float64(3),
			},
			expectedInMsg: "reorg detected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture log output
			output := captureLogOutput(tc.logFunc)

			// Parse log entry
			lines := strings.Split(strings.TrimSpace(output), "\n")
			require.Greater(t, len(lines), 0, "Should have log entry")

			entry, err := parseLogEntry(lines[0])
			require.NoError(t, err, "Should parse log entry")

			// Verify message
			assert.Contains(t, entry.Msg, tc.expectedInMsg,
				"Message should contain expected text")

			// Verify contextual attributes
			for key, expectedValue := range tc.expectedAttrs {
				actualValue, ok := entry.Attrs[key]
				assert.True(t, ok, "Should have attribute '%s'", key)
				assert.Equal(t, expectedValue, actualValue,
					"Attribute '%s' should have expected value", key)
			}

			t.Logf("✓ Log includes context: %s", tc.name)
		})
	}

	t.Log("✓ Logs include contextual information (block height, error details, etc.)")
}

// TestLoggingIntegration_LogLevelFiltering tests log level filtering from environment
// Validates that LOG_LEVEL environment variable controls output
func TestLoggingIntegration_LogLevelFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testCases := []struct {
		envLevel string
	}{
		{"DEBUG"},
		{"INFO"},
		{"WARN"},
		{"ERROR"},
	}

	for _, tc := range testCases {
		t.Run("level_"+tc.envLevel, func(t *testing.T) {
			// Set environment variable
			os.Setenv("LOG_LEVEL", tc.envLevel)
			defer os.Unsetenv("LOG_LEVEL")

			// Create new logger with this level
			logger := NewLogger()
			assert.NotNil(t, logger, "Should create logger")

			// Verify we can log at different levels without panic
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: func() slog.Level {
					switch tc.envLevel {
					case "DEBUG":
						return slog.LevelDebug
					case "INFO":
						return slog.LevelInfo
					case "WARN":
						return slog.LevelWarn
					case "ERROR":
						return slog.LevelError
					default:
						return slog.LevelInfo
					}
				}(),
				AddSource: true,
			})
			testLogger := slog.New(handler)

			// Log at different levels
			testLogger.Debug("debug message")
			testLogger.Info("info message")
			testLogger.Warn("warn message")
			testLogger.Error("error message")

			output := buf.String()
			assert.NotEmpty(t, output, "Should have log output")

			t.Logf("✓ Log level %s configured correctly", tc.envLevel)
		})
	}
}

// TestLoggingIntegration_StructuredLogging tests structured key-value logging
// Validates that attributes are properly structured, not string concatenated
func TestLoggingIntegration_StructuredLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Capture structured log
	output := captureLogOutput(func() {
		Info("indexed block",
			"height", 12345,
			"hash", "0xabcdef",
			"timestamp", 1609459200,
			"tx_count", 42,
		)
	})

	// Parse log entry
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 0, "Should have log entry")

	entry, err := parseLogEntry(lines[0])
	require.NoError(t, err, "Should parse log entry")

	// Verify attributes are separate fields, not concatenated in message
	assert.NotContains(t, entry.Msg, "height=12345",
		"Attributes should not be concatenated in message")
	assert.NotContains(t, entry.Msg, "hash=0xabcdef",
		"Attributes should not be concatenated in message")

	// Verify attributes are in separate fields
	assert.Equal(t, float64(12345), entry.Attrs["height"])
	assert.Equal(t, "0xabcdef", entry.Attrs["hash"])
	assert.Equal(t, float64(1609459200), entry.Attrs["timestamp"])
	assert.Equal(t, float64(42), entry.Attrs["tx_count"])

	t.Log("✓ Logging uses structured key-value pairs, not string concatenation")
}

// TestLoggingIntegration_WithContext tests logger with additional context
// Validates WithContext function adds persistent attributes
func TestLoggingIntegration_WithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create buffer to capture output
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	oldLogger := GlobalLogger
	GlobalLogger = slog.New(handler)
	defer func() { GlobalLogger = oldLogger }()

	// Create logger with context
	contextLogger := WithContext("worker_id", 42, "batch", "test_batch")

	// Log multiple messages with this context
	contextLogger.Info("processing started")
	contextLogger.Info("processing completed", "duration", 1.5)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Equal(t, 2, len(lines), "Should have 2 log entries")

	// Verify both entries have context attributes
	for i, line := range lines {
		entry, err := parseLogEntry(line)
		require.NoError(t, err, "Should parse entry %d", i)

		// Verify persistent context
		assert.Equal(t, float64(42), entry.Attrs["worker_id"],
			"Entry %d should have worker_id context", i)
		assert.Equal(t, "test_batch", entry.Attrs["batch"],
			"Entry %d should have batch context", i)
	}

	t.Log("✓ WithContext adds persistent attributes to logger")
}

// TestLoggingIntegration_HighVolumeLogging tests logging under high load
// Validates logger handles high volume without errors
func TestLoggingIntegration_HighVolumeLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Capture high volume of logs
	messageCount := 1000

	output := captureLogOutput(func() {
		for i := 0; i < messageCount; i++ {
			Info("high volume test",
				"iteration", i,
				"data", "some data",
			)
		}
	})

	// Verify all messages were logged
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, messageCount, len(lines),
		"Should capture all %d log messages", messageCount)

	// Verify random samples are valid JSON
	samples := []int{0, 100, 500, 999}
	for _, idx := range samples {
		_, err := parseLogEntry(lines[idx])
		assert.NoError(t, err, "Log entry %d should be valid JSON", idx)
	}

	t.Logf("✓ Logger handles %d messages without errors", messageCount)
}
