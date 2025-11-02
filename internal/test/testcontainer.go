//go:build integration

package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDatabase holds a test database connection and cleanup function
type TestDatabase struct {
	Pool      *pgxpool.Pool
	Container *postgres.PostgresContainer
	ConnStr   string
}

// SetupTestDB starts a PostgreSQL test container, applies migrations, and returns a connection pool
// Uses testcontainers-go for isolated PostgreSQL instances
// Returns cleanup function that should be called with defer
func SetupTestDB(t *testing.T) (*TestDatabase, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container with specific version and settings
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection string
	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err, "Failed to get connection string")

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "Failed to create connection pool")

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	// Apply migrations
	applyMigrations(t, pool)

	testDB := &TestDatabase{
		Pool:      pool,
		Container: container,
		ConnStr:   connStr,
	}

	// Cleanup function
	cleanup := func() {
		if pool != nil {
			pool.Close()
		}
		if container != nil {
			if err := container.Terminate(ctx); err != nil {
				t.Logf("Failed to terminate container: %v", err)
			}
		}
	}

	return testDB, cleanup
}

// applyMigrations applies all migration files to the test database
func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	// Get project root (assuming we're in internal/test)
	projectRoot, err := getProjectRoot()
	require.NoError(t, err, "Failed to find project root")

	migrationsDir := filepath.Join(projectRoot, "migrations")

	// Read and apply migration files in order
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*_*.up.sql"))
	require.NoError(t, err, "Failed to list migration files")

	for _, file := range files {
		t.Logf("Applying migration: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		require.NoError(t, err, "Failed to read migration file: %s", file)

		_, err = pool.Exec(ctx, string(content))
		require.NoError(t, err, "Failed to apply migration: %s", file)
	}

	t.Logf("Successfully applied %d migrations", len(files))
}

// getProjectRoot finds the project root directory by looking for go.mod
func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up directories looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found (no go.mod)")
		}
		dir = parent
	}
}

// CleanDatabase truncates all tables to reset state between tests
// Useful for tests that need a fresh database without recreating the container
func CleanDatabase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	// Truncate tables in reverse dependency order (logs, transactions, blocks)
	tables := []string{"logs", "transactions", "blocks"}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "Failed to truncate table: %s", table)
	}

	t.Logf("Cleaned database (truncated %d tables)", len(tables))
}

// GetTestDBConfig returns a test database configuration
type TestDBConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// GetTestDBConfig extracts test database configuration from connection string
func (db *TestDatabase) GetConfig(t *testing.T) TestDBConfig {
	t.Helper()

	host, err := db.Container.Host(context.Background())
	require.NoError(t, err)

	port, err := db.Container.MappedPort(context.Background(), "5432")
	require.NoError(t, err)

	return TestDBConfig{
		Host:     host,
		Port:     port.Int(),
		Database: "test_db",
		User:     "test_user",
		Password: "test_password",
	}
}
