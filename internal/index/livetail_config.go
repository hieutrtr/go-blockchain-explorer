package index

import (
	"fmt"
	"os"
	"time"
)

// LiveTailConfig holds configuration for the live-tail coordinator
type LiveTailConfig struct {
	PollInterval time.Duration
}

// NewLiveTailConfig creates a new live-tail configuration from environment variables
// Falls back to sensible defaults if env vars are not set
func NewLiveTailConfig() (*LiveTailConfig, error) {
	pollIntervalStr := os.Getenv("LIVETAIL_POLL_INTERVAL")
	pollInterval := 2 * time.Second // Default: 2 seconds

	if pollIntervalStr != "" {
		duration, err := time.ParseDuration(pollIntervalStr)
		if err == nil && duration > 0 {
			pollInterval = duration
		}
		// If parsing fails, use default (no error)
	}

	return &LiveTailConfig{
		PollInterval: pollInterval,
	}, nil
}

// Validate checks if the configuration is valid
func (c *LiveTailConfig) Validate() error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be > 0, got %v", c.PollInterval)
	}
	if c.PollInterval > 60*time.Second {
		// Warn but don't error - user might want a longer interval
		fmt.Fprintf(os.Stderr, "warning: poll_interval %v is unusually long\n", c.PollInterval)
	}
	return nil
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *LiveTailConfig {
	return &LiveTailConfig{
		PollInterval: 2 * time.Second,
	}
}
