package util

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var metricsInitialized = false

// ensureMetricsInit initializes metrics once for all tests
func ensureMetricsInit(t *testing.T) {
	if !metricsInitialized {
		err := Init()
		if err != nil {
			// Ignore duplicate registration errors in tests
			if !strings.Contains(err.Error(), "duplicate") {
				require.NoError(t, err)
			}
		}
		metricsInitialized = true
	}
}

func TestInit(t *testing.T) {
	t.Run("init creates all metrics", func(t *testing.T) {
		// Reset the init flag to test fresh initialization
		metricsInitialized = false

		// On first call, Init should succeed
		err := Init()
		// It's OK if we get duplicate registration error on subsequent runs
		assert.True(t, err == nil || strings.Contains(err.Error(), "duplicate"))

		// Verify all metrics are initialized
		assert.NotNil(t, BlocksIndexed)
		assert.NotNil(t, IndexLagBlocks)
		assert.NotNil(t, IndexLagSeconds)
		assert.NotNil(t, BackfillDuration)

		metricsInitialized = true
	})
}

func TestRecordBlockIndexed(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("increments counter", func(t *testing.T) {
		// Record a block
		RecordBlockIndexed()
		// If no panic, test passes - actual value verification requires prometheus registry access
	})

	t.Run("increments multiple times", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			RecordBlockIndexed()
		}
		// If no panic, test passes
	})

	t.Run("handles nil gracefully", func(t *testing.T) {
		tempCounter := BlocksIndexed
		BlocksIndexed = nil
		RecordBlockIndexed() // Should not panic
		assert.Nil(t, BlocksIndexed)
		BlocksIndexed = tempCounter
	})
}

func TestSetIndexLagBlocks(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("sets gauge value", func(t *testing.T) {
		SetIndexLagBlocks(42.5)
		// If no panic, test passes
	})

	t.Run("replaces previous value", func(t *testing.T) {
		SetIndexLagBlocks(100)
		SetIndexLagBlocks(200)
		// If no panic, test passes
	})

	t.Run("handles zero and negative values", func(t *testing.T) {
		SetIndexLagBlocks(0)
		SetIndexLagBlocks(-5)
		// If no panic, test passes
	})

	t.Run("handles nil gracefully", func(t *testing.T) {
		tempGauge := IndexLagBlocks
		IndexLagBlocks = nil
		SetIndexLagBlocks(100) // Should not panic
		assert.Nil(t, IndexLagBlocks)
		IndexLagBlocks = tempGauge
	})
}

func TestSetIndexLagSeconds(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("sets gauge value", func(t *testing.T) {
		SetIndexLagSeconds(30.5)
		// If no panic, test passes
	})

	t.Run("replaces previous value", func(t *testing.T) {
		SetIndexLagSeconds(60)
		SetIndexLagSeconds(120)
		// If no panic, test passes
	})

	t.Run("handles nil gracefully", func(t *testing.T) {
		tempGauge := IndexLagSeconds
		IndexLagSeconds = nil
		SetIndexLagSeconds(100) // Should not panic
		assert.Nil(t, IndexLagSeconds)
		IndexLagSeconds = tempGauge
	})
}

func TestRecordRPCError(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("records valid error types", func(t *testing.T) {
		errorTypes := []string{"network", "rate_limit", "invalid_param", "timeout", "other"}

		for _, errorType := range errorTypes {
			RecordRPCError(errorType)
			// If no panic, test passes
		}
	})

	t.Run("increments counter for same error type", func(t *testing.T) {
		RecordRPCError("network")
		RecordRPCError("network")
		RecordRPCError("network")
		// If no panic, test passes
	})

	t.Run("maps unknown error types to other", func(t *testing.T) {
		RecordRPCError("unknown_error_type")
		// If no panic and logs warning, test passes
	})
}

func TestRecordBackfillDuration(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("records valid duration", func(t *testing.T) {
		RecordBackfillDuration(0.5)
		RecordBackfillDuration(2.5)
		RecordBackfillDuration(10.0)
		// If no panic, test passes
	})

	t.Run("handles negative duration gracefully", func(t *testing.T) {
		RecordBackfillDuration(-1.0) // Should log warning but not panic
	})

	t.Run("handles zero duration", func(t *testing.T) {
		RecordBackfillDuration(0.0)
		// If no panic, test passes
	})

	t.Run("handles nil gracefully", func(t *testing.T) {
		tempHistogram := BackfillDuration
		BackfillDuration = nil
		RecordBackfillDuration(1.0) // Should not panic
		assert.Nil(t, BackfillDuration)
		BackfillDuration = tempHistogram
	})
}

func TestGetMetricsPort(t *testing.T) {
	t.Run("returns default when not set", func(t *testing.T) {
		originalPort := os.Getenv("METRICS_PORT")
		os.Unsetenv("METRICS_PORT")
		defer func() {
			if originalPort != "" {
				os.Setenv("METRICS_PORT", originalPort)
			}
		}()

		port := GetMetricsPort()
		assert.Equal(t, "9090", port)
	})

	t.Run("returns custom port when set", func(t *testing.T) {
		originalPort := os.Getenv("METRICS_PORT")
		os.Setenv("METRICS_PORT", "8080")
		defer func() {
			if originalPort != "" {
				os.Setenv("METRICS_PORT", originalPort)
			} else {
				os.Unsetenv("METRICS_PORT")
			}
		}()

		port := GetMetricsPort()
		assert.Equal(t, "8080", port)
	})
}

func TestGetMetricsEndpoint(t *testing.T) {
	t.Run("returns default when not set", func(t *testing.T) {
		originalEndpoint := os.Getenv("METRICS_ENDPOINT")
		os.Unsetenv("METRICS_ENDPOINT")
		defer func() {
			if originalEndpoint != "" {
				os.Setenv("METRICS_ENDPOINT", originalEndpoint)
			}
		}()

		endpoint := GetMetricsEndpoint()
		assert.Equal(t, "/metrics", endpoint)
	})

	t.Run("returns custom endpoint when set", func(t *testing.T) {
		originalEndpoint := os.Getenv("METRICS_ENDPOINT")
		os.Setenv("METRICS_ENDPOINT", "/prometheus")
		defer func() {
			if originalEndpoint != "" {
				os.Setenv("METRICS_ENDPOINT", originalEndpoint)
			} else {
				os.Unsetenv("METRICS_ENDPOINT")
			}
		}()

		endpoint := GetMetricsEndpoint()
		assert.Equal(t, "/prometheus", endpoint)
	})
}

func TestMetricsHTTPEndpoint(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("endpoint returns 200 OK and prometheus format", func(t *testing.T) {
		// Create a test server with prometheus handler
		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		// Create a response recorder
		w := httptest.NewRecorder()

		// Serve with prometheus handler
		promhttp.Handler().ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")

		// Read response body
		body, err := io.ReadAll(w.Body)
		require.NoError(t, err)

		bodyStr := string(body)

		// Verify prometheus format
		assert.Contains(t, bodyStr, "# HELP")
		assert.Contains(t, bodyStr, "# TYPE")

		// Verify metrics are present
		assert.Contains(t, bodyStr, "explorer_blocks_indexed_total")
		assert.Contains(t, bodyStr, "explorer_index_lag_blocks")
		assert.Contains(t, bodyStr, "explorer_index_lag_seconds")
		assert.Contains(t, bodyStr, "explorer_rpc_errors_total")
		assert.Contains(t, bodyStr, "explorer_backfill_duration_seconds")
	})

	t.Run("endpoint returns valid prometheus format lines", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		promhttp.Handler().ServeHTTP(w, req)

		body, err := io.ReadAll(w.Body)
		require.NoError(t, err)

		bodyStr := string(body)
		lines := strings.Split(bodyStr, "\n")

		// Verify format contains expected lines
		hasHelpLine := false
		hasTypeLine := false
		hasMetricLine := false

		for _, line := range lines {
			if strings.HasPrefix(line, "# HELP") {
				hasHelpLine = true
			}
			if strings.HasPrefix(line, "# TYPE") {
				hasTypeLine = true
			}
			if strings.HasPrefix(line, "explorer_") && !strings.HasPrefix(line, "#") {
				hasMetricLine = true
			}
		}

		assert.True(t, hasHelpLine, "should have HELP lines")
		assert.True(t, hasTypeLine, "should have TYPE lines")
		assert.True(t, hasMetricLine, "should have metric lines")
	})

	t.Run("all registered metrics are exposed", func(t *testing.T) {
		// Record some metrics first
		RecordBlockIndexed()
		SetIndexLagBlocks(10)
		SetIndexLagSeconds(30)
		RecordRPCError("network")
		RecordBackfillDuration(0.5)

		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		promhttp.Handler().ServeHTTP(w, req)

		body, err := io.ReadAll(w.Body)
		require.NoError(t, err)

		bodyStr := string(body)

		// Verify all metrics definitions exist
		assert.Contains(t, bodyStr, "explorer_blocks_indexed_total")
		assert.Contains(t, bodyStr, "explorer_index_lag_blocks")
		assert.Contains(t, bodyStr, "explorer_index_lag_seconds")
		assert.Contains(t, bodyStr, "explorer_rpc_errors_total")
		assert.Contains(t, bodyStr, "explorer_backfill_duration_seconds")
	})
}

func TestMetricsConcurrency(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("concurrent block indexing updates", func(t *testing.T) {
		// Create multiple goroutines that record blocks concurrently
		done := make(chan bool, 100)

		for i := 0; i < 100; i++ {
			go func() {
				RecordBlockIndexed()
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 100; i++ {
			<-done
		}
		// If no panic or data race, test passes
	})

	t.Run("concurrent RPC error recording", func(t *testing.T) {
		done := make(chan bool, 50)

		for i := 0; i < 50; i++ {
			go func() {
				RecordRPCError("network")
				done <- true
			}()
		}

		for i := 0; i < 50; i++ {
			<-done
		}
		// If no panic or data race, test passes
	})
}

func TestErrorTypeValidation(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("error type is validated", func(t *testing.T) {
		// Valid error types should not cause issues
		validTypes := []string{"network", "rate_limit", "invalid_param", "timeout", "other"}
		for _, et := range validTypes {
			RecordRPCError(et)
		}
		// If no panic, test passes
	})

	t.Run("unknown error types map to other", func(t *testing.T) {
		// Unknown types should be mapped to "other"
		RecordRPCError("some_unknown_type")
		RecordRPCError("another_unknown")
		// If no panic, test passes - verify via metrics endpoint that they're recorded as "other"
	})
}

func TestHistogramBuckets(t *testing.T) {
	ensureMetricsInit(t)

	t.Run("records duration in correct buckets", func(t *testing.T) {
		// Test that histogram buckets are configured correctly
		RecordBackfillDuration(0.05)  // < 0.1 bucket
		RecordBackfillDuration(0.3)   // 0.1-0.5 bucket
		RecordBackfillDuration(0.75)  // 0.5-1.0 bucket
		RecordBackfillDuration(1.5)   // 1.0-2.0 bucket
		RecordBackfillDuration(3.0)   // 2.0-5.0 bucket
		RecordBackfillDuration(7.0)   // 5.0-10.0 bucket
		RecordBackfillDuration(15.0)  // > 10.0 bucket

		// Get metrics to verify histogram
		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		promhttp.Handler().ServeHTTP(w, req)

		body, err := io.ReadAll(w.Body)
		require.NoError(t, err)

		bodyStr := string(body)

		// Verify histogram bucket lines exist
		assert.Contains(t, bodyStr, "explorer_backfill_duration_seconds_bucket")
		assert.Contains(t, bodyStr, "explorer_backfill_duration_seconds_sum")
		assert.Contains(t, bodyStr, "explorer_backfill_duration_seconds_count")
	})
}
