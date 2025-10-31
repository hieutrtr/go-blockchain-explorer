package index

import (
	"fmt"
	"os"
	"strconv"
)

// ReorgConfig holds configuration for the reorg handler
// Addresses Task 6: Add configuration and metrics
type ReorgConfig struct {
	MaxDepth int // Maximum reorg depth to handle (default: 6 blocks)
}

// NewReorgConfig creates a new reorg configuration from environment variables
// Falls back to sensible defaults if env vars are not set
// Implements AC5: Configuration via REORG_MAX_DEPTH environment variable
// Addresses Task 6.1-6.2: Configuration struct and environment loading
func NewReorgConfig() (*ReorgConfig, error) {
	maxDepthStr := os.Getenv("REORG_MAX_DEPTH")
	maxDepth := 6 // Default: 6 blocks (AC5 requirement)

	if maxDepthStr != "" {
		depth, err := strconv.Atoi(maxDepthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REORG_MAX_DEPTH value '%s': must be an integer", maxDepthStr)
		}
		if depth <= 0 {
			return nil, fmt.Errorf("invalid REORG_MAX_DEPTH value %d: must be > 0", depth)
		}
		maxDepth = depth
	}

	config := &ReorgConfig{
		MaxDepth: maxDepth,
	}

	// Validate before returning
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
// Implements configuration validation (Task 6.2)
func (c *ReorgConfig) Validate() error {
	if c.MaxDepth <= 0 {
		return fmt.Errorf("max_depth must be > 0, got %d", c.MaxDepth)
	}
	if c.MaxDepth > 100 {
		// Warn about unusually large max depth (may indicate misconfiguration)
		// Deep reorgs are rare and may indicate network issues
		fmt.Fprintf(os.Stderr, "warning: max_depth %d is unusually large (typical value: 6)\n", c.MaxDepth)
	}
	return nil
}

// DefaultReorgConfig returns sensible defaults for reorg configuration
// Addresses Task 6.2: Default configuration values
func DefaultReorgConfig() *ReorgConfig {
	return &ReorgConfig{
		MaxDepth: 6, // Default: handle up to 6 block deep reorgs
	}
}
