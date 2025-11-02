package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hieutt50/go-blockchain-explorer/internal/api"
	"github.com/hieutt50/go-blockchain-explorer/internal/api/websocket"
	"github.com/hieutt50/go-blockchain-explorer/internal/db"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

func main() {
	// Initialize global logger (already done in init(), but explicit for clarity)
	util.Info("starting blockchain explorer API server")

	// Load API configuration
	apiConfig := api.NewConfig()
	util.Info("API server configuration loaded",
		"port", apiConfig.Port,
		"cors_origins", apiConfig.CORSOrigins,
	)

	// Load database configuration
	dbConfig, err := db.NewConfig()
	if err != nil {
		util.Error("failed to load database configuration", "error", err.Error())
		os.Exit(1)
	}

	// Create database connection pool (separate from indexer, max 10 connections for API)
	if dbConfig.MaxConns > 10 {
		dbConfig.MaxConns = 10 // Override to 10 for API server
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, dbConfig)
	if err != nil {
		util.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	util.Info("database connection pool established",
		"max_conns", dbConfig.MaxConns,
	)

	// Note: Database migrations are managed separately to avoid race conditions.
	// Run migrations manually with: make migrate
	// or: make db-setup
	util.Info("skipping database migrations (manage manually to avoid conflicts)")

	// Initialize WebSocket Hub for real-time streaming (Story 2.2)
	wsConfig := websocket.LoadConfig()
	hub := websocket.NewHub(wsConfig)
	util.Info("WebSocket hub initialized",
		"max_connections", wsConfig.MaxConnections,
		"ping_interval", wsConfig.PingInterval,
	)

	// Start WebSocket Hub in background with cancellable context
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel() // Ensure hub is stopped on exit
	go hub.Run(hubCtx)
	util.Info("WebSocket hub started")

	// Create API server with WebSocket hub
	server := api.NewServerWithHub(pool, apiConfig, hub)
	util.Info("API server initialized with WebSocket support")

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         apiConfig.Address(),
		Handler:      server.Router(),
		ReadTimeout:  apiConfig.ReadTimeout,
		WriteTimeout: apiConfig.WriteTimeout,
		IdleTimeout:  apiConfig.IdleTimeout,
	}

	// Start HTTP server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		util.Info("API server listening",
			"address", httpServer.Addr,
		)
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		util.Info("received shutdown signal", "signal", sig.String())

	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			util.Error("server error", "error", err.Error())
		}
	}

	// Graceful shutdown
	util.Info("shutting down API server gracefully",
		"timeout_seconds", apiConfig.ShutdownTimeout.Seconds(),
	)

	// Stop WebSocket Hub first (closes all client connections)
	hubCancel()
	util.Info("WebSocket hub shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), apiConfig.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		util.Error("error during server shutdown", "error", err.Error())
		// Force close after timeout
		if err := httpServer.Close(); err != nil {
			util.Error("error forcing server close", "error", err.Error())
		}
	}

	util.Info("API server shutdown complete")
}
