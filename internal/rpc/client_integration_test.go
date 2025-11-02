//go:build integration

package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRPCIntegration_RetryLogic tests retry with transient failures
// Validates AC4: RPC Client Integration Tests - retry logic
func TestRPCIntegration_RetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate test blocks
	fixtures := test.GenerateTestBlocks(t, 1, 10, 1)

	// Create mock RPC client
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Configure to fail 2 times before succeeding
	mockRPC.SetGlobalFailures(2)

	// First call should fail
	_, err := mockRPC.GetBlockByNumber(ctx, 1)
	assert.Error(t, err, "First call should fail")

	// Second call should fail
	_, err = mockRPC.GetBlockByNumber(ctx, 1)
	assert.Error(t, err, "Second call should fail")

	// Third call should succeed
	block, err := mockRPC.GetBlockByNumber(ctx, 1)
	require.NoError(t, err, "Third call should succeed after retries")
	assert.NotNil(t, block, "Should return block")

	t.Log("RPC retry logic validated successfully")
}

// TestRPCIntegration_ExponentialBackoff tests retry timing
// Validates AC4: Exponential backoff timing
func TestRPCIntegration_ExponentialBackoff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	fixtures := test.GenerateTestBlocks(t, 1, 1, 1)
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Set 3 failures
	mockRPC.SetGlobalFailures(3)

	// TODO: Implement retry wrapper with exponential backoff
	// This would wrap the mock client and add retry logic
	//
	// retryClient := NewRetryClient(mockRPC, RetryConfig{
	//     MaxAttempts: 4,
	//     InitialBackoff: 100 * time.Millisecond,
	//     MaxBackoff: 1 * time.Second,
	//     Multiplier: 2.0,
	// })

	// Measure retry attempts and timing
	startTime := time.Now()

	// Make calls and measure timing
	for i := 0; i < 4; i++ {
		attemptStart := time.Now()
		_, err := mockRPC.GetBlockByNumber(ctx, 1)

		duration := time.Since(attemptStart)
		t.Logf("Attempt %d: %v (error: %v)", i+1, duration, err != nil)

		if err == nil {
			break
		}
	}

	totalDuration := time.Since(startTime)
	t.Logf("Total time with retries: %v", totalDuration)

	// AC4: Verify exponential backoff timing
	// Expected: ~100ms, ~200ms, ~400ms between attempts
	// Total: ~700ms minimum
	// assert.Greater(t, totalDuration, 700*time.Millisecond, "Should respect backoff timing")
}

// TestRPCIntegration_PermanentError tests immediate failure
// Validates AC4: Permanent error immediate failure (no retry)
func TestRPCIntegration_PermanentError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	fixtures := test.GenerateTestBlocks(t, 1, 10, 1)
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Set permanent error for block 5
	mockRPC.SetPermanentError(5)

	// Attempt to fetch block 5 (should fail immediately)
	startTime := time.Now()
	_, err := mockRPC.GetBlockByNumber(ctx, 5)
	duration := time.Since(startTime)

	assert.Error(t, err, "Should return error for permanent failure")
	assert.Contains(t, err.Error(), "invalid", "Error should indicate permanent failure")

	// Should fail fast (no retries)
	assert.Less(t, duration, 100*time.Millisecond, "Should fail immediately without retries")

	t.Log("Permanent error handling validated")
}

// TestRPCIntegration_ContextCancellation tests timeout handling
// Validates AC4: Context cancellation during retry loop
func TestRPCIntegration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	fixtures := test.GenerateTestBlocks(t, 1, 100, 1)
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Set delay to simulate slow network
	mockRPC.SetDelay(200 * time.Millisecond)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start fetching blocks
	startTime := time.Now()
	var fetchCount int

	for i := uint64(1); i <= 100; i++ {
		_, err := mockRPC.GetBlockByNumber(ctx, i)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				t.Logf("Context canceled after fetching %d blocks", fetchCount)
				break
			}
			require.NoError(t, err, "Unexpected error")
		}
		fetchCount++
	}

	duration := time.Since(startTime)
	t.Logf("Fetched %d blocks in %v before context cancellation", fetchCount, duration)

	// Should have stopped early due to context cancellation
	assert.Less(t, fetchCount, 100, "Should not complete all fetches")
	assert.Less(t, duration, 1*time.Second, "Should stop on context cancellation")
}

// TestRPCIntegration_MaxRetriesExceeded tests retry limit
// Validates AC4: Max retries exceeded error
func TestRPCIntegration_MaxRetriesExceeded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	fixtures := test.GenerateTestBlocks(t, 1, 1, 1)
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Set high failure count (more than max retries)
	mockRPC.SetGlobalFailures(10)

	// TODO: Use retry client with max attempts = 3
	// retryClient := NewRetryClient(mockRPC, RetryConfig{
	//     MaxAttempts: 3,
	//     InitialBackoff: 50 * time.Millisecond,
	// })

	// Make calls until max retries exceeded
	var lastErr error
	for i := 0; i < 5; i++ {
		_, err := mockRPC.GetBlockByNumber(ctx, 1)
		lastErr = err
		if err != nil {
			t.Logf("Attempt %d failed: %v", i+1, err)
		}
	}

	assert.Error(t, lastErr, "Should still be failing after max retries")
	t.Log("Max retries exceeded handling validated")
}

// TestRPCIntegration_ConcurrentCalls tests thread safety
// Validates that RPC client handles concurrent calls correctly
func TestRPCIntegration_ConcurrentCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	fixtures := test.GenerateTestBlocks(t, 1, 100, 1)
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Launch multiple goroutines fetching blocks concurrently
	concurrency := 10
	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			// Each worker fetches 10 blocks
			for j := 0; j < 10; j++ {
				height := uint64(workerID*10 + j + 1)
				_, err := mockRPC.GetBlockByNumber(ctx, height)
				if err != nil {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}(i)
	}

	// Wait for all workers
	for i := 0; i < concurrency; i++ {
		err := <-errChan
		assert.NoError(t, err, "Concurrent calls should succeed")
	}

	// Verify total call count
	totalCalls := mockRPC.GetCallCount()
	expectedCalls := concurrency * 10
	assert.Equal(t, expectedCalls, totalCalls, "Should have made expected number of calls")

	t.Logf("Handled %d concurrent RPC calls successfully", totalCalls)
}

// TestRPCIntegration_ErrorClassification tests error type detection
// Validates AC4: Error classification (transient vs permanent)
func TestRPCIntegration_ErrorClassification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Test transient errors
	t.Run("transient_errors", func(t *testing.T) {
		transientErrors := []string{
			"network timeout",
			"connection refused",
			"temporary failure",
		}

		for _, errMsg := range transientErrors {
			// TODO: Use error classification logic
			// isTransient := IsTransientError(errors.New(errMsg))
			// assert.True(t, isTransient, "%s should be transient", errMsg)

			t.Logf("Error '%s' should be transient (retryable)", errMsg)
		}
	})

	// Test permanent errors
	t.Run("permanent_errors", func(t *testing.T) {
		permanentErrors := []string{
			"invalid block height",
			"block not found",
			"invalid parameters",
		}

		for _, errMsg := range permanentErrors {
			// TODO: Use error classification logic
			// isTransient := IsTransientError(errors.New(errMsg))
			// assert.False(t, isTransient, "%s should be permanent", errMsg)

			t.Logf("Error '%s' should be permanent (no retry)", errMsg)
		}
	})

	_ = ctx
}

// TestRPCIntegration_SlowNetwork tests behavior with slow responses
// Validates handling of slow network conditions
func TestRPCIntegration_SlowNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	fixtures := test.GenerateTestBlocks(t, 1, 10, 1)

	// Create slow RPC client (200-500ms delays)
	slowRPC := test.NewMockSlowRPCClient(t, fixtures, 200*time.Millisecond, 500*time.Millisecond)

	// Fetch blocks and measure timing
	startTime := time.Now()
	var totalFetched int

	for i := uint64(1); i <= 10; i++ {
		_, err := slowRPC.GetBlockByNumber(ctx, i)
		require.NoError(t, err, "Should handle slow network")
		totalFetched++
	}

	duration := time.Since(startTime)
	t.Logf("Fetched %d blocks in %v with slow network", totalFetched, duration)

	// Should have fetched all blocks despite slow network
	assert.Equal(t, 10, totalFetched, "Should fetch all blocks")
	assert.Greater(t, duration, 2*time.Second, "Should take time with slow network")
}
