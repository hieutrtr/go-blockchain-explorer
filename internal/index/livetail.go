package index

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// WebSocketBroadcaster interface for broadcasting to WebSocket clients
type WebSocketBroadcaster interface {
	BroadcastBlock(block BlockData)
	BroadcastTransaction(tx TransactionData)
}

// BlockData represents minimal block data for broadcasting
type BlockData struct {
	Height    uint64
	Hash      string
	TxCount   int
	Timestamp int64
	Miner     string
	GasUsed   uint64
}

// TransactionData represents minimal transaction data for broadcasting
type TransactionData struct {
	Hash        string
	BlockHeight uint64
}

// LiveTailCoordinator manages sequential live-tail processing of new blocks
type LiveTailCoordinator struct {
	rpcClient    RPCBlockFetcher
	store        BlockStore
	ingester     BlockIngester
	reorgHandler ReorgHandler
	hub          WebSocketBroadcaster // Optional WebSocket hub for real-time broadcasts
	config       *LiveTailConfig
	logger       *slog.Logger

	// parseRPCBlock is a mockable function for testing
	parseRPCBlock func(rpcBlock *types.Block, height uint64) *Block

	// Metrics
	blocksProcessed int64
	currentHeight   int64
	startTime       time.Time
}

// BlockStore interface for database operations (stub for now)
type BlockStore interface {
	GetLatestBlock(ctx context.Context) (*Block, error)
	InsertBlock(ctx context.Context, block *Block) error
	GetBlockByHeight(ctx context.Context, height uint64) (*Block, error)
}

// BlockIngester interface for block parsing (stub for now)
type BlockIngester interface {
	ParseBlock(rpcBlock *types.Block) (*Block, error)
}

// ReorgHandler interface for reorg handling (stub for Story 1.5)
type ReorgHandler interface {
	HandleReorg(ctx context.Context, block *Block) error
}

// Block represents a blockchain block (domain model, stub)
type Block struct {
	Height     uint64
	Hash       []byte
	ParentHash []byte
	Timestamp  uint64
	Miner      []byte // Coinbase address
	GasUsed    uint64
	TxCount    int
}

// NewLiveTailCoordinator creates a new live-tail coordinator
func NewLiveTailCoordinator(
	rpcClient RPCBlockFetcher,
	store BlockStore,
	ingester BlockIngester,
	reorgHandler ReorgHandler,
	hub WebSocketBroadcaster, // Optional: can be nil
	config *LiveTailConfig,
) (*LiveTailCoordinator, error) {
	if rpcClient == nil {
		return nil, fmt.Errorf("rpcClient cannot be nil")
	}
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ltc := &LiveTailCoordinator{
		rpcClient:    rpcClient,
		store:        store,
		ingester:     ingester,
		reorgHandler: reorgHandler,
		hub:          hub, // Optional WebSocket broadcaster
		config:       config,
		logger:       logger,
	}
	// Set default parseRPCBlock function (can be overridden in tests)
	ltc.parseRPCBlock = ltc.defaultParseRPCBlock
	return ltc, nil
}

// Start begins the live-tail polling loop
// Implements AC1 (Sequential), AC2 (Polling), AC3 (Error Handling), AC4 (Reorg), AC5 (Observability)
func (ltc *LiveTailCoordinator) Start(ctx context.Context) error {
	ltc.startTime = time.Now()

	ltc.logger.Info("starting live-tail coordinator",
		slog.Duration("poll_interval", ltc.config.PollInterval),
	)

	ticker := time.NewTicker(ltc.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Process next block
			if err := ltc.processNextBlock(ctx); err != nil {
				// AC3: Log and continue (don't halt)
				ltc.logger.Error("error processing block",
					slog.String("error", err.Error()),
					slog.Int64("current_height", ltc.currentHeight),
				)
				// Continue to next tick
			}

		case <-ctx.Done():
			// AC3: Graceful shutdown
			ltc.logger.Info("live-tail coordinator shutting down",
				slog.Duration("duration", time.Since(ltc.startTime)),
				slog.Int64("blocks_processed", ltc.blocksProcessed),
			)
			return ctx.Err()
		}
	}
}

// processNextBlock fetches and processes the next sequential block
// Implements AC1 (Sequential), AC2 (Next height), AC3 (Error handling), AC4 (Reorg check)
func (ltc *LiveTailCoordinator) processNextBlock(ctx context.Context) error {
	// AC1: Query database head before each fetch
	dbHead, err := ltc.store.GetLatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	nextHeight := dbHead.Height + 1

	// AC1/AC2: Fetch next block from RPC
	rpcBlock, err := ltc.rpcClient.GetBlockByNumber(ctx, nextHeight)
	if err != nil {
		// AC2: Handle "block not found" gracefully (next block not yet produced)
		if isBlockNotFound(err) {
			ltc.logger.Debug("next block not yet produced",
				slog.Uint64("next_height", nextHeight),
			)
			return nil // Not an error, just wait for next tick
		}
		// AC3: RPC error (transient or permanent), will retry on next tick
		return fmt.Errorf("failed to fetch block %d: %w", nextHeight, err)
	}

	if rpcBlock == nil {
		ltc.logger.Debug("block not found from rpc",
			slog.Uint64("height", nextHeight),
		)
		return nil
	}

	// AC1/AC2: Parse RPC block to domain model (stub: just extract fields)
	domainBlock := ltc.parseRPCBlock(rpcBlock, nextHeight)

	// AC4: Check for parent hash mismatch (reorg detection)
	if !bytesEqual(domainBlock.ParentHash, dbHead.Hash) {
		ltc.logger.Warn("parent hash mismatch detected (potential reorg)",
			slog.Uint64("height", nextHeight),
			slog.Uint64("db_head_height", dbHead.Height),
		)

		// AC4: Trigger reorg handler if available
		if ltc.reorgHandler != nil {
			if err := ltc.reorgHandler.HandleReorg(ctx, domainBlock); err != nil {
				ltc.logger.Error("reorg handler failed",
					slog.String("error", err.Error()),
					slog.Uint64("height", nextHeight),
				)
				// Continue anyway (reorg handling is asynchronous)
			}
		}
		return nil // Skip this block, will retry on next tick after reorg resolution
	}

	// AC1/AC2: Insert block into database
	if err := ltc.store.InsertBlock(ctx, domainBlock); err != nil {
		return fmt.Errorf("failed to insert block %d: %w", nextHeight, err)
	}

	// Broadcast block to WebSocket clients (Story 2.2)
	if ltc.hub != nil {
		ltc.hub.BroadcastBlock(BlockData{
			Height:    domainBlock.Height,
			Hash:      fmt.Sprintf("0x%x", domainBlock.Hash),
			TxCount:   domainBlock.TxCount,
			Timestamp: int64(domainBlock.Timestamp),
			Miner:     fmt.Sprintf("0x%x", domainBlock.Miner),
			GasUsed:   domainBlock.GasUsed,
		})
	}

	// AC5: Update metrics
	atomic.AddInt64(&ltc.blocksProcessed, 1)
	atomic.StoreInt64(&ltc.currentHeight, int64(nextHeight))

	// AC5: Log block processed with lag calculation
	lag := time.Since(ltc.startTime) / time.Duration(ltc.blocksProcessed) // Simplified lag estimate
	ltc.logger.Info("block processed",
		slog.Uint64("height", nextHeight),
		slog.Duration("lag_estimate", lag),
		slog.Int64("blocks_processed", ltc.blocksProcessed),
	)

	return nil
}

// defaultParseRPCBlock converts go-ethereum Block to domain model
// Stub implementation - full parsing happens in Story 1.5+ (Ingester)
func (ltc *LiveTailCoordinator) defaultParseRPCBlock(rpcBlock *types.Block, height uint64) *Block {
	return &Block{
		Height:     height,
		Hash:       rpcBlock.Hash().Bytes(),
		ParentHash: rpcBlock.ParentHash().Bytes(),
		Timestamp:  rpcBlock.Time(),
		Miner:      rpcBlock.Coinbase().Bytes(),
		GasUsed:    rpcBlock.GasUsed(),
		TxCount:    len(rpcBlock.Transactions()),
	}
}

// Stats returns live-tail statistics
func (ltc *LiveTailCoordinator) Stats() map[string]interface{} {
	return map[string]interface{}{
		"blocks_processed": atomic.LoadInt64(&ltc.blocksProcessed),
		"current_height":   atomic.LoadInt64(&ltc.currentHeight),
		"duration":         time.Since(ltc.startTime),
		"poll_interval":    ltc.config.PollInterval,
	}
}

// Helper functions

// isBlockNotFound checks if error indicates block not found (not an error condition)
func isBlockNotFound(err error) bool {
	// Check for common "block not found" error patterns
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return errMsg == "not found" ||
		   errMsg == "block not found" ||
		   errMsg == "unknown block"
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
