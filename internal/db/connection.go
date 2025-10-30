package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// Pool wraps pgxpool.Pool to provide connection pooling
type Pool struct {
	*pgxpool.Pool
	config *Config
}

// NewPool creates a new database connection pool
// It establishes connections to PostgreSQL and verifies connectivity with a ping
func NewPool(ctx context.Context, config *Config) (*Pool, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Build connection pool configuration
	poolConfig, err := pgxpool.ParseConfig(config.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection config: %w", err)
	}

	// Configure connection pool settings
	poolConfig.MaxConns = int32(config.MaxConns)
	poolConfig.MaxConnIdleTime = config.IdleTimeout
	poolConfig.MaxConnLifetime = config.ConnLifetime
	poolConfig.HealthCheckPeriod = 1 * config.ConnTimeout

	// Create connection pool with timeout
	ctx, cancel := context.WithTimeout(ctx, config.ConnTimeout)
	defer cancel()

	util.Info("connecting to database",
		"config", config.SafeString())

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection with ping
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	util.Info("database connection established",
		"max_conns", config.MaxConns,
		"idle_timeout", config.IdleTimeout.String(),
		"conn_lifetime", config.ConnLifetime.String())

	return &Pool{
		Pool:   pool,
		config: config,
	}, nil
}

// Close closes the database connection pool
func (p *Pool) Close() {
	if p.Pool != nil {
		util.Info("closing database connection pool")
		p.Pool.Close()
	}
}

// HealthCheck performs a health check on the database connection
// Returns nil if the database is healthy, error otherwise
func (p *Pool) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, p.config.ConnTimeout)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// Stats returns connection pool statistics
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Stat()
}
