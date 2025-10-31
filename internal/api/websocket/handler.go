package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

var (
	// Rate limiting: track connections per IP
	rateLimiter   = make(map[string]*ipRateLimit)
	rateLimiterMu sync.Mutex
)

type ipRateLimit struct {
	count     int
	lastReset time.Time
}

// checkOrigin returns a function that validates WebSocket origin based on config
func checkOrigin(config *Config) func(*http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// If no allowed origins specified or contains "*", allow all
		if len(config.AllowedOrigins) == 0 || contains(config.AllowedOrigins, "*") {
			return true
		}

		// Check if origin is in allowed list
		return contains(config.AllowedOrigins, origin)
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// HandleWebSocket handles WebSocket upgrade requests
func HandleWebSocket(hub *Hub, config *Config) http.HandlerFunc {
	// Create upgrader with config-based buffer sizes and proper CORS check
	upgrader := websocket.Upgrader{
		ReadBufferSize:  config.ReadBufferSize,
		WriteBufferSize: config.WriteBufferSize,
		CheckOrigin:     checkOrigin(config),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		clientIP := r.RemoteAddr

		// Check rate limit
		if !checkRateLimit(clientIP) {
			http.Error(w, "Too many connections", http.StatusTooManyRequests)
			util.Warn("WebSocket rate limit exceeded", "ip", clientIP)
			IncrementErrorMetrics("rate_limit_exceeded")
			return
		}

		// Check max connections
		stats := hub.Stats()
		if stats.ActiveConnections >= config.MaxConnections {
			http.Error(w, "Max connections reached", http.StatusServiceUnavailable)
			util.Warn("WebSocket max connections reached",
				"active", stats.ActiveConnections,
				"max", config.MaxConnections,
			)
			IncrementErrorMetrics("max_connections_reached")
			return
		}

		// Upgrade connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			util.Error("WebSocket upgrade failed", "error", err)
			IncrementErrorMetrics("upgrade_failed")
			return
		}

		// Create client
		clientID := uuid.New().String()
		client := NewClient(clientID, conn, hub)

		// Register client with hub
		hub.register <- client

		// Start client goroutines
		go client.writePump()
		go client.readPump()

		util.Info("WebSocket connection established",
			"client_id", clientID,
			"remote_addr", r.RemoteAddr,
		)
	}
}

// checkRateLimit checks if IP is within rate limit (10 connections per minute)
func checkRateLimit(ip string) bool {
	rateLimiterMu.Lock()
	defer rateLimiterMu.Unlock()

	now := time.Now()

	limit, exists := rateLimiter[ip]
	if !exists {
		rateLimiter[ip] = &ipRateLimit{
			count:     1,
			lastReset: now,
		}
		return true
	}

	// Reset counter if minute has passed
	if now.Sub(limit.lastReset) > time.Minute {
		limit.count = 1
		limit.lastReset = now
		return true
	}

	// Check limit
	if limit.count >= 10 {
		return false
	}

	limit.count++
	return true
}
