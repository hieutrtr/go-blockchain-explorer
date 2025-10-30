package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

func main() {
	// Initialize metrics package
	if err := util.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize metrics: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting blockchain explorer worker")

	// Start metrics server in a goroutine
	go func() {
		if err := util.StartMetricsServer(); err != nil {
			logger.Error("metrics server failed", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	logger.Info("worker started, waiting for signals")

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("received signal",
		"signal", sig.String(),
	)

	// TODO: Add graceful shutdown for other components (RPC client, database, coordinators)
	// For now, we're just demonstrating metrics server integration

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("shutting down gracefully",
		"timeout_seconds", 30,
	)

	// Wait for context to complete or timeout
	<-ctx.Done()

	logger.Info("worker shutdown complete")
}
