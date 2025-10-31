package websocket

import (
	"os"
	"strconv"
	"time"
)

// Config holds WebSocket configuration
type Config struct {
	MaxConnections   int
	PingInterval     time.Duration
	ReadBufferSize   int
	WriteBufferSize  int
	AllowedOrigins   []string
}

// LoadConfig loads WebSocket configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		MaxConnections:  getEnvAsInt("WEBSOCKET_MAX_CONNECTIONS", 1000),
		PingInterval:    getEnvAsDuration("WEBSOCKET_PING_INTERVAL", 30*time.Second),
		ReadBufferSize:  getEnvAsInt("WEBSOCKET_READ_BUFFER_SIZE", 1024),
		WriteBufferSize: getEnvAsInt("WEBSOCKET_WRITE_BUFFER_SIZE", 1024),
		AllowedOrigins:  getEnvAsStringSlice("API_CORS_ORIGINS", []string{"*"}),
	}
}

// getEnvAsInt reads an environment variable as int with default
func getEnvAsInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// getEnvAsDuration reads an environment variable as duration with default
func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultVal
}

// getEnvAsStringSlice reads an environment variable as string slice
func getEnvAsStringSlice(key string, defaultVal []string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{value}
	}
	return defaultVal
}
