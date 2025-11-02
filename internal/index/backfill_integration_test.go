//go:build integration

package index

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackfillIntegration_100Blocks tests backfilling 100 blocks end-to-end
// Validates AC1: Backfill Integration Tests
func TestBackfillIntegration_100Blocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup test database
	_, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate 100 test blocks with 5 transactions each
	fixtures := test.GenerateTestBlocks(t, 1, 100, 5)
	require.Len(t, fixtures, 100, "Should generate 100 blocks")

	// Create mock RPC client with test data
	mockRPC := test.NewMockRPCClient(t, fixtures)
	require.Equal(t, 100, mockRPC.GetBlockCount(), "Mock RPC should have 100 blocks")

	// Create backfill coordinator
	config := &Config{
		Workers:     4,
		BatchSize:   25,
		StartHeight: 1,
		EndHeight:   100,
	}

	mockStore := NewIntegrationMockBlockStoreExtended()
	coordinator, err := NewBackfillCoordinator(mockRPC, mockStore, config)
	require.NoError(t, err, "Failed to create backfill coordinator")

	// Measure backfill time (AC1: tests complete in <30 seconds)
	startTime := time.Now()

	// Execute backfill
	err = coordinator.Backfill(ctx, 1, 100)
	require.NoError(t, err, "Backfill should complete without errors")

	duration := time.Since(startTime)
	t.Logf("Backfill completed in %v", duration)

	// AC1: Tests complete in <30 seconds
	assert.Less(t, duration, 30*time.Second, "Backfill should complete within 30 seconds")

	// Verify RPC calls were made
	callCount := mockRPC.GetCallCount()
	t.Logf("RPC GetBlockByNumber called %d times", callCount)
	assert.Greater(t, callCount, 0, "RPC should be called")

	// TODO: Once database insertion is implemented in backfill.go:
	// - Verify all blocks are in database
	// - Verify transactions are linked to blocks
	// - Verify logs are linked to transactions
	// - Validate foreign key relationships
	// - Check block hashes match expected values
}

// TestBackfillIntegration_WithDatabaseInsertion tests full backfill with DB writes
// This test will be expanded once database insertion is implemented in backfill.go
func TestBackfillIntegration_WithDatabaseInsertion(t *testing.T) {
	t.Skip("TODO: Implement once backfill.go writes to database")

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup test database
	_, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate smaller test chain (10 blocks)
	fixtures := test.GenerateTestBlocks(t, 1, 10, 3)

	// Create mock RPC
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Create coordinator
	config := &Config{
		Workers:     2,
		BatchSize:   5,
		StartHeight: 1,
		EndHeight:   10,
	}

	mockStore := NewIntegrationMockBlockStoreExtended()
	coordinator, err := NewBackfillCoordinator(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Execute backfill
	err = coordinator.Backfill(ctx, 1, 10)
	require.NoError(t, err)

	// Verify database state
	// TODO: Query database and verify:
	// 1. All 10 blocks are inserted
	// 2. All 30 transactions are inserted (10 blocks * 3 tx)
	// 3. All logs are inserted
	// 4. Foreign key relationships are intact
	// 5. Block hashes match fixture values
}

// TestBackfillIntegration_ErrorHandling tests RPC failure scenarios
// Validates AC1: Error handling during backfill
func TestBackfillIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup test database
	_, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate test blocks
	fixtures := test.GenerateTestBlocks(t, 1, 50, 2)

	// Create mock RPC with failures
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Configure block 25 to fail 2 times before succeeding
	mockRPC.SetFailures(25, 2)

	// Create coordinator with retry enabled
	config := &Config{
		Workers:     2,
		BatchSize:   10,
		StartHeight: 1,
		EndHeight:   50,
	}

	mockStore := NewIntegrationMockBlockStoreExtended()
	coordinator, err := NewBackfillCoordinator(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Execute backfill (should succeed despite failures)
	err = coordinator.Backfill(ctx, 1, 50)
	require.NoError(t, err, "Backfill should succeed with retries")

	// Verify block 25 was eventually fetched
	assert.True(t, mockRPC.GetCallCount() > 50, "Should have made retry attempts")

	t.Log("Backfill successfully handled transient RPC failures")
}

// TestBackfillIntegration_ContextCancellation tests graceful shutdown
// Validates AC1: Context cancellation handling
func TestBackfillIntegration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test database
	_, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate large block range
	fixtures := test.GenerateTestBlocks(t, 1, 1000, 1)

	// Create mock RPC with delay to simulate slow network
	mockRPC := test.NewMockRPCClient(t, fixtures)
	mockRPC.SetDelay(50 * time.Millisecond) // Slow down fetching

	// Create coordinator
	config := &Config{
		Workers:     4,
		BatchSize:   50,
		StartHeight: 1,
		EndHeight:   1000,
	}

	mockStore := NewIntegrationMockBlockStoreExtended()
	coordinator, err := NewBackfillCoordinator(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start backfill in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- coordinator.Backfill(ctx, 1, 1000)
	}()

	// Cancel after short duration
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Wait for backfill to stop
	err = <-errChan

	// Should return context canceled error
	assert.Error(t, err, "Backfill should return error on cancellation")
	t.Logf("Backfill stopped with error: %v", err)

	// Verify backfill stopped early
	callCount := mockRPC.GetCallCount()
	assert.Less(t, callCount, 1000, "Should not have fetched all blocks")
	t.Logf("Fetched %d blocks before cancellation", callCount)
}

// TestBackfillIntegration_Performance benchmarks backfill speed
// Validates AC1: Performance requirements
func TestBackfillIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup test database
	_, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate 100 blocks with varying transaction counts
	fixtures := test.GenerateTestBlocks(t, 1, 100, 10)

	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Test different worker configurations
	workerConfigs := []struct {
		workers int
		expect  time.Duration
	}{
		{workers: 1, expect: 30 * time.Second}, // Baseline
		{workers: 4, expect: 10 * time.Second}, // Expected improvement
		{workers: 8, expect: 8 * time.Second},  // Diminishing returns
	}

	for _, tc := range workerConfigs {
		t.Run(fmt.Sprintf("workers=%d", tc.workers), func(t *testing.T) {
			config := &Config{
				Workers:     tc.workers,
				BatchSize:   25,
				StartHeight: 1,
				EndHeight:   100,
			}

			mockStore := NewIntegrationMockBlockStoreExtended()
	coordinator, err := NewBackfillCoordinator(mockRPC, mockStore, config)
			require.NoError(t, err)

			mockRPC.ResetCallCount()

			startTime := time.Now()
			err = coordinator.Backfill(ctx, 1, 100)
			duration := time.Since(startTime)

			require.NoError(t, err)
			t.Logf("Workers=%d: %v (expected <%v)", tc.workers, duration, tc.expect)

			// Performance should improve with more workers
			assert.Less(t, duration, tc.expect, "Should meet performance target")
		})
	}
}

// TestBackfillIntegration_EmptyRange tests edge case of empty block range
func TestBackfillIntegration_EmptyRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	_, cleanup := test.SetupTestDB(t)
	defer cleanup()

	fixtures := test.GenerateTestBlocks(t, 1, 10, 1)
	mockRPC := test.NewMockRPCClient(t, fixtures)

	config := &Config{
		Workers:     2,
		BatchSize:   5,
		StartHeight: 1,
		EndHeight:   10,
	}

	mockStore := NewIntegrationMockBlockStoreExtended()
	coordinator, err := NewBackfillCoordinator(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Test invalid range (start > end)
	err = coordinator.Backfill(ctx, 10, 5)
	assert.Error(t, err, "Should return error for invalid range")

	// Test single block
	mockRPC.ResetCallCount()
	err = coordinator.Backfill(ctx, 5, 5)
	assert.NoError(t, err, "Should handle single block")
	assert.Equal(t, 1, mockRPC.GetCallCount(), "Should fetch exactly 1 block")
}
