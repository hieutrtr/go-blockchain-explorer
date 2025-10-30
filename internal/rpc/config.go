package rpc

import (
	"fmt"
	"os"
	"time"
)

// Config holds configuration for the RPC client
type Config struct {
	// RPCURL is the Ethereum RPC endpoint URL (from RPC_URL environment variable)
	RPCURL string

	// ConnectionTimeout is the timeout for establishing RPC connections (default: 10s)
	ConnectionTimeout time.Duration

	// RequestTimeout is the timeout for individual RPC requests (default: 30s)
	RequestTimeout time.Duration

	// MaxRetries is the maximum number of retry attempts for transient failures (default: 5)
	MaxRetries int

	// RetryBaseDelay is the base delay for exponential backoff (default: 1s)
	RetryBaseDelay time.Duration
}

// NewConfig creates a new Config with default values
// RPCURL is read from RPC_URL environment variable
func NewConfig() (*Config, error) {
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("RPC_URL environment variable not set")
	}

	return &Config{
		RPCURL:            rpcURL,
		ConnectionTimeout: 10 * time.Second,
		RequestTimeout:    30 * time.Second,
		MaxRetries:        5,
		RetryBaseDelay:    1 * time.Second,
	}, nil
}

// NewConfigWithDefaults creates a Config with a provided URL and default timeout values
// Useful for testing scenarios
func NewConfigWithDefaults(rpcURL string) *Config {
	return &Config{
		RPCURL:            rpcURL,
		ConnectionTimeout: 10 * time.Second,
		RequestTimeout:    30 * time.Second,
		MaxRetries:        5,
		RetryBaseDelay:    1 * time.Second,
	}
}
