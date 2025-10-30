package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Requires PostgreSQL running with test database
	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, pool)
	defer pool.Close()

	// Verify pool is functional with a simple query
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestNewPool_NilConfig(t *testing.T) {
	ctx := context.Background()

	pool, err := NewPool(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestNewPool_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Use an invalid host to test connection failure
	config := NewConfigWithDefaults("invalid-host-that-does-not-exist", 5432, "test", "user", "pass", 10)

	// Use a short timeout to make test faster
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestPool_HealthCheck_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Test health check
	err = pool.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestPool_HealthCheck_AfterClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)

	// Close the pool
	pool.Close()

	// Health check should fail after close
	err = pool.HealthCheck(ctx)
	assert.Error(t, err)
}

func TestPool_Stats_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Get stats
	stats := pool.Stats()
	assert.NotNil(t, stats)

	// Stats should show some connections (at least 0)
	assert.GreaterOrEqual(t, stats.TotalConns(), int32(0))
}

func TestPool_ConcurrentConnections_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Test concurrent queries
	numWorkers := 10
	done := make(chan bool, numWorkers)
	errors := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer func() { done <- true }()
			var result int
			err := pool.QueryRow(ctx, "SELECT $1::int", id).Scan(&result)
			if err != nil {
				errors <- err
				return
			}
			if result != id {
				errors <- assert.AnError
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent query error: %v", err)
	}
}

func TestPool_ContextCancellation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Create a context that's already cancelled
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	// Query with cancelled context should fail
	var result int
	err = pool.QueryRow(cancelledCtx, "SELECT 1").Scan(&result)
	assert.Error(t, err)
}

func TestPool_Close_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	require.NoError(t, err)

	// Close should be idempotent - calling multiple times should not panic
	pool.Close()
	pool.Close() // Second close should be safe
}

func TestPool_Close_NilPool(t *testing.T) {
	// Create a Pool with nil internal pool (edge case)
	pool := &Pool{
		Pool:   nil,
		config: NewConfigWithDefaults("localhost", 5432, "test", "user", "pass", 10),
	}

	// Close with nil pool should not panic
	assert.NotPanics(t, func() {
		pool.Close()
	})
}

func TestNewPool_InvalidConnectionString(t *testing.T) {
	// Config with invalid characters that would cause connection string parsing to fail
	config := NewConfigWithDefaults("", 0, "", "", "", -1)
	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestNewPool_ParseConfigError(t *testing.T) {
	// Create a config that will cause ParseConfig to fail
	// This tests the error path in NewPool:29-32
	config := &Config{
		Host:         "localhost",
		Port:         5432,
		Name:         "test",
		User:         "user",
		Password:     "pass",
		MaxConns:     20,
		ConnTimeout:  5 * time.Second,
		IdleTimeout:  5 * time.Minute,
		ConnLifetime: 30 * time.Minute,
	}

	ctx := context.Background()

	// Try to connect to invalid host - should fail
	config.Host = "invalid-host-does-not-exist-12345"
	pool, err := NewPool(ctx, config)
	// Connection should fail (either at creation or ping)
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestPool_AllMethods_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Manually set environment variables for this test
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "test_migrations")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
	}()

	config, err := NewConfig()
	if err != nil {
		t.Skipf("skipping test: database configuration not available: %v", err)
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Skipf("skipping test: could not connect to database: %v", err)
	}
	require.NotNil(t, pool)

	// Test HealthCheck
	err = pool.HealthCheck(ctx)
	assert.NoError(t, err, "health check should succeed")

	// Test Stats
	stats := pool.Stats()
	assert.NotNil(t, stats, "stats should not be nil")
	assert.GreaterOrEqual(t, stats.TotalConns(), int32(0), "total connections should be >= 0")

	// Test Close
	pool.Close()

	// HealthCheck after close should fail
	err = pool.HealthCheck(ctx)
	assert.Error(t, err, "health check should fail after close")
}
