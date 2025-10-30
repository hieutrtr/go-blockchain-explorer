package rpc

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateBackoff(t *testing.T) {
	baseDelay := 1 * time.Second

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"attempt 0", 0, 1 * time.Second},  // 1s
		{"attempt 1", 1, 2 * time.Second},  // 2s
		{"attempt 2", 2, 4 * time.Second},  // 4s
		{"attempt 3", 3, 8 * time.Second},  // 8s
		{"attempt 4", 4, 16 * time.Second}, // 16s
		{"attempt 5", 5, 32 * time.Second}, // 32s (beyond max retries, but function should work)
		{"negative attempt", -1, 1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoff(tt.attempt, baseDelay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryWithBackoff_SuccessFirstTry(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 5,
		baseDelay:  10 * time.Millisecond, // Short delay for tests
	}

	callCount := 0
	operation := func() error {
		callCount++
		return nil // Success on first try
	}

	ctx := context.Background()
	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "should call operation once")
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 5,
		baseDelay:  10 * time.Millisecond,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			// Fail first 2 attempts with transient error
			return errors.New("connection timeout")
		}
		return nil // Success on 3rd attempt
	}

	ctx := context.Background()
	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "should call operation 3 times")
}

func TestRetryWithBackoff_PermanentError(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 5,
		baseDelay:  10 * time.Millisecond,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("invalid parameter") // Permanent error
	}

	ctx := context.Background()
	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "should call operation only once for permanent error")
	assert.Contains(t, err.Error(), "permanent")
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 3,
		baseDelay:  10 * time.Millisecond,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection timeout") // Always fail with transient error
	}

	ctx := context.Background()
	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.Error(t, err)
	// Should call: 1 initial + 3 retries = 4 total
	assert.Equal(t, 4, callCount, "should call operation max retries + 1")
	assert.Contains(t, err.Error(), "max retries exceeded")
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 5,
		baseDelay:  100 * time.Millisecond, // Longer delay to test cancellation
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection timeout") // Always fail
	}

	// Create context that cancels after short time
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err, "should return context error")
	// Should only call once before context cancels during backoff wait
	assert.LessOrEqual(t, callCount, 2, "should stop retrying when context cancelled")
}

func TestRetryWithBackoff_RateLimitError(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 3,
		baseDelay:  10 * time.Millisecond,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("HTTP 429: too many requests") // Rate limit error
		}
		return nil // Success on 3rd attempt
	}

	ctx := context.Background()
	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "should retry rate limit errors")
}

func TestRetryWithBackoff_ExponentialBackoffTiming(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 3,
		baseDelay:  50 * time.Millisecond,
	}

	callCount := 0
	var timestamps []time.Time

	operation := func() error {
		timestamps = append(timestamps, time.Now())
		callCount++
		if callCount <= 3 {
			return errors.New("connection timeout") // Fail first 3 attempts
		}
		return nil
	}

	ctx := context.Background()
	start := time.Now()
	err := retryWithBackoff(ctx, cfg, operation, logger, "test-operation")

	assert.NoError(t, err)
	assert.Equal(t, 4, callCount)

	// Verify exponential backoff delays
	// Expected delays: 0ms (initial), 50ms, 100ms, 200ms
	// Total time should be approximately 350ms

	totalDuration := time.Since(start)
	expectedMinDuration := 350 * time.Millisecond
	expectedMaxDuration := 500 * time.Millisecond // Allow some overhead

	assert.GreaterOrEqual(t, totalDuration, expectedMinDuration, "should respect backoff delays")
	assert.LessOrEqual(t, totalDuration, expectedMaxDuration, "delays should not be excessive")

	// Verify individual delays between calls
	if len(timestamps) >= 2 {
		delay1 := timestamps[1].Sub(timestamps[0])
		assert.GreaterOrEqual(t, delay1, 40*time.Millisecond, "first retry delay should be ~50ms")
		assert.LessOrEqual(t, delay1, 100*time.Millisecond, "first retry delay should not exceed 100ms")
	}

	if len(timestamps) >= 3 {
		delay2 := timestamps[2].Sub(timestamps[1])
		assert.GreaterOrEqual(t, delay2, 90*time.Millisecond, "second retry delay should be ~100ms")
		assert.LessOrEqual(t, delay2, 150*time.Millisecond, "second retry delay should not exceed 150ms")
	}
}

// Benchmark retry with backoff
func BenchmarkRetryWithBackoff_Success(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 5,
		baseDelay:  1 * time.Millisecond,
	}

	operation := func() error {
		return nil // Always succeed
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retryWithBackoff(ctx, cfg, operation, logger, "benchmark-op")
	}
}

func BenchmarkRetryWithBackoff_WithRetries(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := &retryConfig{
		maxRetries: 3,
		baseDelay:  1 * time.Millisecond,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 2 {
				return errors.New("transient error")
			}
			return nil
		}

		_ = retryWithBackoff(ctx, cfg, operation, logger, "benchmark-op")
	}
}
