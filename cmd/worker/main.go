package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/db"
	"github.com/hieutt50/go-blockchain-explorer/internal/index"
	"github.com/hieutt50/go-blockchain-explorer/internal/rpc"
	"github.com/hieutt50/go-blockchain-explorer/internal/store"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

func main() {
	// Initialize metrics and logging
	if err := util.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize utilities: %v\n", err)
		os.Exit(1)
	}

	util.Info("starting blockchain explorer worker",
		"version", "1.0.0",
	)

	// Create main context for the application
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// =============================================================================
	// Configuration Loading
	// =============================================================================

	// Load database configuration
	dbConfig, err := db.NewConfig()
	if err != nil {
		util.Error("failed to load database configuration", "error", err.Error())
		os.Exit(1)
	}
	util.Info("database configuration loaded",
		"host", dbConfig.Host,
		"port", dbConfig.Port,
		"database", dbConfig.Name,
		"max_conns", dbConfig.MaxConns,
	)

	// Load RPC configuration
	rpcConfig, err := rpc.NewConfig()
	if err != nil {
		util.Error("failed to load RPC configuration", "error", err.Error())
		os.Exit(1)
	}
	util.Info("RPC configuration loaded",
		"rpc_url", maskAPIKey(rpcConfig.RPCURL),
	)

	// Load indexer configurations
	backfillConfig, err := index.NewConfig()
	if err != nil {
		util.Error("failed to load backfill configuration", "error", err.Error())
		os.Exit(1)
	}
	util.Info("backfill configuration loaded",
		"workers", backfillConfig.Workers,
		"batch_size", backfillConfig.BatchSize,
		"start_height", backfillConfig.StartHeight,
		"end_height", backfillConfig.EndHeight,
	)

	livetailConfig, err := index.NewLiveTailConfig()
	if err != nil {
		util.Error("failed to load livetail configuration", "error", err.Error())
		os.Exit(1)
	}
	util.Info("livetail configuration loaded",
		"poll_interval", livetailConfig.PollInterval,
	)

	reorgConfig, err := index.NewReorgConfig()
	if err != nil {
		util.Error("failed to load reorg configuration", "error", err.Error())
		os.Exit(1)
	}
	util.Info("reorg configuration loaded",
		"max_depth", reorgConfig.MaxDepth,
	)

	// =============================================================================
	// Database Setup
	// =============================================================================

	// Connect to database
	pool, err := db.NewPool(ctx, dbConfig)
	if err != nil {
		util.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer pool.Close()
	util.Info("database connection pool established",
		"max_conns", dbConfig.MaxConns,
	)

	// Note: Database migrations are handled by the API server on startup.
	// The worker skips migrations to avoid race conditions and conflicts.
	util.Info("skipping database migrations (handled by API server)")

	// =============================================================================
	// Component Creation
	// =============================================================================

	// Create RPC client
	rpcClient, err := rpc.NewClient(rpcConfig)
	if err != nil {
		util.Error("failed to create RPC client", "error", err.Error())
		os.Exit(1)
	}
	util.Info("RPC client created")

	// Create store adapter for indexer operations
	storeAdapter := store.NewIndexerAdapter(pool)
	util.Info("store adapter created")

	// =============================================================================
	// Start Metrics Server
	// =============================================================================

	// Start metrics server in background
	go func() {
		util.Info("starting metrics server")
		if err := util.StartMetricsServer(); err != nil {
			util.Error("metrics server failed", "error", err.Error())
			cancel() // Cancel main context on metrics server failure
		}
	}()

	// =============================================================================
	// Backfill Phase (if needed)
	// =============================================================================

	// Check if backfill is needed
	needsBackfill := true
	latestBlock, err := storeAdapter.GetLatestBlock(ctx)
	if err == nil && latestBlock != nil {
		util.Info("database already contains blocks",
			"latest_height", latestBlock.Height,
		)
		// Only backfill if target end height is greater than current latest
		if latestBlock.Height >= backfillConfig.EndHeight {
			needsBackfill = false
			util.Info("backfill not needed - database is up to date",
				"latest_height", latestBlock.Height,
				"target_height", backfillConfig.EndHeight,
			)
		} else {
			// Adjust start height to continue from latest
			backfillConfig.StartHeight = latestBlock.Height + 1
			util.Info("backfill needed - continuing from latest block",
				"start_height", backfillConfig.StartHeight,
				"end_height", backfillConfig.EndHeight,
			)
		}
	} else {
		util.Info("database is empty - full backfill required",
			"start_height", backfillConfig.StartHeight,
			"end_height", backfillConfig.EndHeight,
		)
	}

	if needsBackfill && backfillConfig.EndHeight > 0 {
		util.Info("starting backfill phase")
		backfillStart := time.Now()

		// Create backfill coordinator with store for database insertion
		backfillCoordinator, err := index.NewBackfillCoordinator(rpcClient, storeAdapter, backfillConfig)
		if err != nil {
			util.Error("failed to create backfill coordinator", "error", err.Error())
			os.Exit(1)
		}

		// Run backfill - fetches blocks in parallel and stores them in database
		err = backfillCoordinator.Backfill(ctx, backfillConfig.StartHeight, backfillConfig.EndHeight)
		if err != nil {
			util.Error("backfill failed", "error", err.Error())
			// Don't exit - continue to live-tail which will catch up
			util.Warn("continuing to live-tail despite backfill failure")
		} else {
			backfillDuration := time.Since(backfillStart)
			blocksBackfilled := backfillConfig.EndHeight - backfillConfig.StartHeight + 1
			util.Info("backfill phase completed",
				"duration_seconds", backfillDuration.Seconds(),
				"blocks_backfilled", blocksBackfilled,
				"blocks_per_second", float64(blocksBackfilled)/backfillDuration.Seconds(),
			)
		}
	}

	// =============================================================================
	// Create Reorg Handler
	// =============================================================================

	reorgHandler, err := index.NewReorgHandler(rpcClient, storeAdapter, reorgConfig)
	if err != nil {
		util.Error("failed to create reorg handler", "error", err.Error())
		os.Exit(1)
	}
	util.Info("reorg handler created")

	// =============================================================================
	// Live-Tail Phase
	// =============================================================================

	util.Info("starting live-tail phase")

	// Create live-tail coordinator
	liveTailCoordinator, err := index.NewLiveTailCoordinator(
		rpcClient,
		storeAdapter,
		nil, // no ingester needed - using default parser
		reorgHandler,
		nil, // no WebSocket hub in worker
		livetailConfig,
	)
	if err != nil {
		util.Error("failed to create live-tail coordinator", "error", err.Error())
		os.Exit(1)
	}

	// Start live-tail in background goroutine
	liveTailCtx, liveTailCancel := context.WithCancel(ctx)
	defer liveTailCancel()

	liveTailErrors := make(chan error, 1)
	go func() {
		util.Info("live-tail coordinator started")
		liveTailErrors <- liveTailCoordinator.Start(liveTailCtx)
	}()

	// =============================================================================
	// Graceful Shutdown Handling
	// =============================================================================

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	util.Info("worker fully started - indexing blockchain",
		"latest_block", func() uint64 {
			if latestBlock != nil {
				return latestBlock.Height
			}
			return 0
		}(),
	)

	// Wait for shutdown signal or live-tail error
	select {
	case sig := <-sigChan:
		util.Info("received shutdown signal", "signal", sig.String())

	case err := <-liveTailErrors:
		if err != nil {
			util.Error("live-tail coordinator failed", "error", err.Error())
		}
	}

	// Graceful shutdown
	util.Info("initiating graceful shutdown",
		"timeout_seconds", 30,
	)

	// Cancel live-tail context
	liveTailCancel()

	// Wait for live-tail to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	select {
	case <-liveTailErrors:
		util.Info("live-tail coordinator stopped cleanly")
	case <-shutdownCtx.Done():
		util.Warn("live-tail coordinator shutdown timed out")
	}

	// Close database pool
	pool.Close()
	util.Info("database connection pool closed")

	util.Info("worker shutdown complete")
}

// maskAPIKey masks the API key in RPC URLs for logging
func maskAPIKey(rpcURL string) string {
	// Simple masking - replace everything after last / with ***
	for i := len(rpcURL) - 1; i >= 0; i-- {
		if rpcURL[i] == '/' {
			return rpcURL[:i+1] + "***"
		}
	}
	return rpcURL
}
