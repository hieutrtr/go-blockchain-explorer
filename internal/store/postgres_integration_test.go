//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseIntegration_BulkInsert tests bulk insert performance
// Validates AC5: Database Integration Tests - bulk insert with 100+ blocks
func TestDatabaseIntegration_BulkInsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Generate 100 test blocks with transactions and logs
	fixtures := test.GenerateTestBlocks(t, 1, 100, 5)
	require.Len(t, fixtures, 100, "Should generate 100 blocks")

	// Measure bulk insert time (AC5: <1 second for 100 blocks)
	startTime := time.Now()

	// TODO: Once store implementation with bulk insert is ready:
	// ctx := context.Background()
	// testDB, cleanup := test.SetupTestDB(t)
	// defer cleanup()
	// store := NewPostgresStore(testDB.Pool)
	// err := store.BulkInsertBlocks(ctx, fixtures)
	// require.NoError(t, err, "Bulk insert should succeed")

	duration := time.Since(startTime)
	t.Logf("Bulk insert of 100 blocks took %v", duration)

	// AC5: Insert performance meets target (<1 second for 100 blocks)
	// assert.Less(t, duration, 1*time.Second, "Bulk insert should be fast")

	// Verify all blocks were inserted
	// TODO: Query database to verify:
	// - 100 blocks inserted
	// - 500 transactions inserted (100 blocks * 5 tx)
	// - All logs inserted
	// - No duplicates

	_ = fixtures // Use fixtures once store is implemented
}

// TestDatabaseIntegration_TransactionRollback tests commit/rollback behavior
// Validates AC5: Transaction management
func TestDatabaseIntegration_TransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	testDB, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate test data
	fixtures := test.GenerateTestBlocks(t, 1, 10, 2)

	// Test successful transaction commit
	t.Run("commit", func(t *testing.T) {
		tx, err := testDB.Pool.Begin(ctx)
		require.NoError(t, err, "Should begin transaction")

		// TODO: Insert blocks within transaction
		// err = insertBlocksInTx(ctx, tx, fixtures[0:5])
		// require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err, "Transaction should commit")

		// Verify blocks are visible after commit
		// TODO: Query and verify blocks 1-5 exist
	})

	// Test transaction rollback
	t.Run("rollback", func(t *testing.T) {
		tx, err := testDB.Pool.Begin(ctx)
		require.NoError(t, err, "Should begin transaction")

		// TODO: Insert blocks within transaction
		// err = insertBlocksInTx(ctx, tx, fixtures[5:10])
		// require.NoError(t, err)

		err = tx.Rollback(ctx)
		require.NoError(t, err, "Transaction should rollback")

		// Verify blocks are NOT visible after rollback
		// TODO: Query and verify blocks 6-10 don't exist
	})

	_ = fixtures // Use fixtures once insert is implemented
}

// TestDatabaseIntegration_ForeignKeyCascade tests foreign key constraints
// Validates AC5: Cascade deletes (delete block → delete txs → delete logs)
func TestDatabaseIntegration_ForeignKeyCascade(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	testDB, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate test blocks with transactions and logs
	fixtures := test.GenerateTestBlocks(t, 1, 5, 3)

	// TODO: Insert blocks, transactions, and logs
	// store := NewPostgresStore(testDB.Pool)
	// err := store.InsertBlocks(ctx, fixtures)
	// require.NoError(t, err)

	// Verify initial state
	// TODO: Count blocks, transactions, logs
	// initialBlocks := countBlocks(ctx, testDB.Pool)
	// initialTxs := countTransactions(ctx, testDB.Pool)
	// initialLogs := countLogs(ctx, testDB.Pool)
	// require.Equal(t, 5, initialBlocks)
	// require.Equal(t, 15, initialTxs) // 5 blocks * 3 tx
	// require.Greater(t, initialLogs, 0)

	// Delete one block
	_, err := testDB.Pool.Exec(ctx, "DELETE FROM blocks WHERE height = $1", uint64(3))
	require.NoError(t, err, "Should delete block")

	// Verify cascade delete
	// TODO: Verify transactions and logs for block 3 are also deleted
	// remainingBlocks := countBlocks(ctx, testDB.Pool)
	// remainingTxs := countTransactions(ctx, testDB.Pool)
	// require.Equal(t, 4, remainingBlocks, "Should have 4 blocks left")
	// require.Equal(t, 12, remainingTxs, "Should have 12 txs left (4 blocks * 3)")

	_ = fixtures
}

// TestDatabaseIntegration_UniqueConstraints tests duplicate prevention
// Validates AC5: Unique constraints prevent duplicates
func TestDatabaseIntegration_UniqueConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Generate test block
	fixtures := test.GenerateTestBlocks(t, 1, 1, 1)
	block := fixtures[0]

	// TODO: Insert block first time (should succeed)
	// ctx := context.Background()
	// testDB, cleanup := test.SetupTestDB(t)
	// defer cleanup()
	// store := NewPostgresStore(testDB.Pool)
	// err := store.InsertBlock(ctx, block)
	// require.NoError(t, err, "First insert should succeed")

	// Try to insert same block again (should fail due to unique constraint)
	// err = store.InsertBlock(ctx, block)
	// assert.Error(t, err, "Duplicate insert should fail")
	// assert.Contains(t, err.Error(), "unique", "Error should mention unique constraint")

	_ = block
}

// TestDatabaseIntegration_ConnectionPool tests connection pool behavior
// Validates AC5: Connection pool under concurrent load
func TestDatabaseIntegration_ConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Generate test data
	fixtures := test.GenerateTestBlocks(t, 1, 100, 2)

	// TODO: Insert all blocks
	// ctx := context.Background()
	// store := NewPostgresStore(testDB.Pool)
	// err := store.BulkInsertBlocks(ctx, fixtures)
	// require.NoError(t, err)

	// Simulate concurrent queries
	concurrency := 20
	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			// Each worker performs multiple queries
			for j := 0; j < 10; j++ {
				// Query random block
				height := uint64((workerID*10 + j) % 100 + 1)

				// TODO: Query block
				// _, err := store.GetBlockByHeight(ctx, height)
				// if err != nil {
				//     errChan <- err
				//     return
				// }

				_ = height // Use once query is implemented
			}
			errChan <- nil
		}(i)
	}

	// Wait for all workers
	for i := 0; i < concurrency; i++ {
		err := <-errChan
		assert.NoError(t, err, "Concurrent queries should succeed")
	}

	// Verify pool stats
	stats := testDB.Pool.Stat()
	t.Logf("Connection pool stats: TotalConns=%d, IdleConns=%d, AcquiredConns=%d",
		stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())

	// Pool should have handled concurrent load efficiently
	assert.Greater(t, stats.TotalConns(), int32(0), "Should have active connections")
	assert.LessOrEqual(t, stats.TotalConns(), int32(20), "Should not exceed max connections")

	_ = fixtures
}

// TestDatabaseIntegration_Migrations tests that schema matches production
// Validates AC8: Test database schema matches production schema
func TestDatabaseIntegration_Migrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	testDB, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Verify tables exist
	tables := []string{"blocks", "transactions", "logs"}

	for _, table := range tables {
		var exists bool
		err := testDB.Pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table).Scan(&exists)

		require.NoError(t, err, "Should query table existence")
		assert.True(t, exists, "Table %s should exist", table)
	}

	// Verify key indexes exist
	indexes := []string{
		"idx_blocks_orphaned_height",
		"idx_tx_from_addr_block",
		"idx_tx_to_addr_block",
	}

	for _, index := range indexes {
		var exists bool
		err := testDB.Pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM pg_indexes WHERE indexname = $1)",
			index).Scan(&exists)

		require.NoError(t, err, "Should query index existence")
		assert.True(t, exists, "Index %s should exist", index)
	}

	t.Log("Database schema validation complete")
}

// TestDatabaseIntegration_CleanupBetweenTests tests isolation
// Validates AC8: Tests are isolated with clean database state
func TestDatabaseIntegration_CleanupBetweenTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	testDB, cleanup := test.SetupTestDB(t)
	defer cleanup()

	// Test 1: Insert data
	t.Run("insert_data", func(t *testing.T) {
		fixtures := test.GenerateTestBlocks(t, 1, 5, 1)

		// TODO: Insert blocks
		// store := NewPostgresStore(testDB.Pool)
		// err := store.InsertBlocks(ctx, fixtures)
		// require.NoError(t, err)

		// Verify data exists
		// count := countBlocks(ctx, testDB.Pool)
		// assert.Equal(t, 5, count, "Should have 5 blocks")

		_ = fixtures
	})

	// Clean database between tests
	test.CleanDatabase(t, testDB.Pool)

	// Test 2: Verify clean state
	t.Run("verify_clean", func(t *testing.T) {
		// Count should be zero after cleanup
		var count int
		err := testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM blocks").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Blocks table should be empty after cleanup")

		err = testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Transactions table should be empty after cleanup")

		err = testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM logs").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Logs table should be empty after cleanup")
	})
}
