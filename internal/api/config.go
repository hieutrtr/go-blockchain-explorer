package api

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for the API server
type Config struct {
	// Port is the HTTP server port (from API_PORT environment variable, default: 8080)
	Port int

	// CORSOrigins is the allowed CORS origins (from API_CORS_ORIGINS environment variable, default: *)
	CORSOrigins string

	// ReadTimeout is the maximum duration for reading the entire request (default: 30s)
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response (default: 30s)
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the next request (default: 120s)
	IdleTimeout time.Duration

	// ShutdownTimeout is the maximum duration for graceful shutdown (default: 30s)
	ShutdownTimeout time.Duration
}

// NewConfig creates a new Config from environment variables
// Optional environment variables: API_PORT (default: 8080), API_CORS_ORIGINS (default: *)
func NewConfig() *Config {
	// Parse port with default
	port := 8080
	if portStr := os.Getenv("API_PORT"); portStr != "" {
		if parsedPort, err := strconv.Atoi(portStr); err == nil && parsedPort > 0 && parsedPort <= 65535 {
			port = parsedPort
		}
	}

	// Parse CORS origins with default
	corsOrigins := os.Getenv("API_CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}

	return &Config{
		Port:            port,
		CORSOrigins:     corsOrigins,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Address returns the listen address for the HTTP server
func (c *Config) Address() string {
	return fmt.Sprintf(":%d", c.Port)
}
