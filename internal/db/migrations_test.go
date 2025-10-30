package db

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunMigrations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup: Requires PostgreSQL running with test database
	// Set these environment variables to run this test:
	// DB_HOST=localhost DB_PORT=5432 DB_NAME=test_migrations DB_USER=postgres DB_PASSWORD=postgres
	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Get absolute path to migrations directory
	migrationsPath := "../../migrations"

	// Test running migrations
	err = RunMigrations(config, migrationsPath, logger)
	require.NoError(t, err, "initial migration should succeed")

	// Test idempotency - running migrations again should be safe
	err = RunMigrations(config, migrationsPath, logger)
	assert.NoError(t, err, "running migrations again should be safe (ErrNoChange)")

	// Verify migration version
	version, dirty, err := GetMigrationVersion(config, migrationsPath)
	require.NoError(t, err)
	assert.Greater(t, version, uint(0), "version should be greater than 0 after migrations")
	assert.False(t, dirty, "migration should not be dirty")

	// Test rollback
	err = RollbackMigrations(config, migrationsPath, logger)
	assert.NoError(t, err, "rollback should succeed")

	// Verify version decreased
	newVersion, dirty, err := GetMigrationVersion(config, migrationsPath)
	require.NoError(t, err)
	assert.Less(t, newVersion, version, "version should decrease after rollback")
	assert.False(t, dirty, "migration should not be dirty after rollback")
}

func TestRunMigrations_WithConnection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	// Run migrations
	migrationsPath := "../../migrations"
	err = RunMigrations(config, migrationsPath, logger)
	require.NoError(t, err)

	// Create connection pool to verify schema
	pool, err := NewPool(ctx, config, logger)
	require.NoError(t, err)
	defer pool.Close()

	// Verify tables exist
	tables := []string{"blocks", "transactions", "logs"}
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = $1
			)`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "table %s should exist after migrations", table)
	}

	// Verify indexes exist (migration 2)
	indexes := []string{
		"idx_blocks_orphaned_height",
		"idx_blocks_timestamp",
		"idx_tx_block_height",
		"idx_tx_from_addr_block",
		"idx_tx_to_addr_block",
		"idx_tx_block_index",
		"idx_logs_tx_hash",
		"idx_logs_address_topic0",
		"idx_logs_address",
	}

	for _, index := range indexes {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT FROM pg_indexes
				WHERE indexname = $1
			)`, index).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "index %s should exist after migrations", index)
	}
}

func TestRunMigrations_NilConfig(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err := RunMigrations(nil, "../../migrations", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestRunMigrations_NilLogger(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)
	err := RunMigrations(config, "../../migrations", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logger cannot be nil")
}

func TestRunMigrations_EmptyMigrationsPath(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err := RunMigrations(config, "", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrationsPath cannot be empty")
}

func TestRollbackMigrations_NilConfig(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err := RollbackMigrations(nil, "../../migrations", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestGetMigrationVersion_NilConfig(t *testing.T) {
	_, _, err := GetMigrationVersion(nil, "../../migrations")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestGetMigrationVersion_EmptyPath(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)
	_, _, err := GetMigrationVersion(config, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrationsPath cannot be empty")
}

func TestRunMigrations_InvalidPath(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Use a path that doesn't exist
	err := RunMigrations(config, "/nonexistent/path", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create migrate instance")
}

func TestRollbackMigrations_InvalidPath(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Use a path that doesn't exist
	err := RollbackMigrations(config, "/nonexistent/path", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create migrate instance")
}

func TestGetMigrationVersion_InvalidPath(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)

	// Use a path that doesn't exist
	_, _, err := GetMigrationVersion(config, "/nonexistent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create migrate instance")
}

func TestRollbackMigrations_EmptyPath(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := RollbackMigrations(config, "", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrationsPath cannot be empty")
}
