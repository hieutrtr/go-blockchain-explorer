package db

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for the database connection
type Config struct {
	// Host is the database server hostname (from DB_HOST environment variable, default: localhost)
	Host string

	// Port is the database server port (from DB_PORT environment variable, default: 5432)
	Port int

	// Name is the database name (from DB_NAME environment variable, required)
	Name string

	// User is the database user (from DB_USER environment variable, required)
	User string

	// Password is the database password (from DB_PASSWORD environment variable, required)
	Password string

	// MaxConns is the maximum number of connections in the pool (from DB_MAX_CONNS environment variable, default: 20)
	MaxConns int

	// ConnTimeout is the timeout for establishing database connections (default: 5s)
	ConnTimeout time.Duration

	// IdleTimeout is the maximum time a connection can be idle (default: 5m)
	IdleTimeout time.Duration

	// ConnLifetime is the maximum lifetime of a connection (default: 30m)
	ConnLifetime time.Duration
}

// NewConfig creates a new Config from environment variables
// Required environment variables: DB_NAME, DB_USER, DB_PASSWORD
// Optional environment variables: DB_HOST (default: localhost), DB_PORT (default: 5432), DB_MAX_CONNS (default: 20)
func NewConfig() (*Config, error) {
	// Required fields
	name := os.Getenv("DB_NAME")
	if name == "" {
		return nil, fmt.Errorf("DB_NAME environment variable not set")
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		return nil, fmt.Errorf("DB_USER environment variable not set")
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("DB_PASSWORD environment variable not set")
	}

	// Optional fields with defaults
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := 5432
	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		parsedPort, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT value: %w", err)
		}
		if parsedPort < 1 || parsedPort > 65535 {
			return nil, fmt.Errorf("DB_PORT must be between 1 and 65535, got %d", parsedPort)
		}
		port = parsedPort
	}

	maxConns := 20
	if maxConnsStr := os.Getenv("DB_MAX_CONNS"); maxConnsStr != "" {
		parsedMaxConns, err := strconv.Atoi(maxConnsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MAX_CONNS value: %w", err)
		}
		if parsedMaxConns < 1 {
			return nil, fmt.Errorf("DB_MAX_CONNS must be at least 1, got %d", parsedMaxConns)
		}
		maxConns = parsedMaxConns
	}

	return &Config{
		Host:         host,
		Port:         port,
		Name:         name,
		User:         user,
		Password:     password,
		MaxConns:     maxConns,
		ConnTimeout:  5 * time.Second,
		IdleTimeout:  5 * time.Minute,
		ConnLifetime: 30 * time.Minute,
	}, nil
}

// NewConfigWithDefaults creates a Config with provided values and default timeout settings
// Useful for testing scenarios
func NewConfigWithDefaults(host string, port int, name, user, password string, maxConns int) *Config {
	return &Config{
		Host:         host,
		Port:         port,
		Name:         name,
		User:         user,
		Password:     password,
		MaxConns:     maxConns,
		ConnTimeout:  5 * time.Second,
		IdleTimeout:  5 * time.Minute,
		ConnLifetime: 30 * time.Minute,
	}
}

// ConnectionString builds a PostgreSQL connection string
// Password is included but should never be logged
func (c *Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
	)
}

// SafeString returns a string representation with the password masked
// Safe for logging
func (c *Config) SafeString() string {
	return fmt.Sprintf(
		"postgres://%s:***@%s:%d/%s (maxConns=%d)",
		c.User,
		c.Host,
		c.Port,
		c.Name,
		c.MaxConns,
	)
}
