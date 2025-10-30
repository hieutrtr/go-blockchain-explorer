package index

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for the backfill coordinator
type Config struct {
	Workers   int
	BatchSize int
	StartHeight uint64
	EndHeight   uint64
}

// NewConfig creates a new backfill configuration from environment variables
// Falls back to sensible defaults if env vars are not set
func NewConfig() (*Config, error) {
	workers := getEnvInt("BACKFILL_WORKERS", 8)
	if workers <= 0 {
		return nil, fmt.Errorf("BACKFILL_WORKERS must be > 0, got %d", workers)
	}

	batchSize := getEnvInt("BACKFILL_BATCH_SIZE", 100)
	if batchSize <= 0 {
		return nil, fmt.Errorf("BACKFILL_BATCH_SIZE must be > 0, got %d", batchSize)
	}

	startHeight := getEnvUint64("BACKFILL_START_HEIGHT", 0)
	endHeight := getEnvUint64("BACKFILL_END_HEIGHT", 5000)

	if startHeight >= endHeight {
		return nil, fmt.Errorf("BACKFILL_START_HEIGHT (%d) must be < BACKFILL_END_HEIGHT (%d)",
			startHeight, endHeight)
	}

	return &Config{
		Workers:     workers,
		BatchSize:   batchSize,
		StartHeight: startHeight,
		EndHeight:   endHeight,
	}, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Workers <= 0 {
		return fmt.Errorf("workers must be > 0, got %d", c.Workers)
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be > 0, got %d", c.BatchSize)
	}
	if c.StartHeight >= c.EndHeight {
		return fmt.Errorf("start_height (%d) must be < end_height (%d)",
			c.StartHeight, c.EndHeight)
	}
	return nil
}

// TotalBlocks returns the total number of blocks to backfill
func (c *Config) TotalBlocks() uint64 {
	return c.EndHeight - c.StartHeight + 1
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// getEnvUint64 gets a uint64 environment variable with a default value
func getEnvUint64(key string, defaultVal uint64) uint64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	// Overall timeout for the entire backfill operation
	BackfillTimeout time.Duration
	// Timeout for each block fetch operation
	BlockFetchTimeout time.Duration
}

// DefaultTimeoutConfig returns sensible timeout defaults
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		BackfillTimeout:   30 * time.Minute,
		BlockFetchTimeout: 10 * time.Second,
	}
}
