package index

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// ReorgHandlerImpl implements the ReorgHandler interface for chain reorganization detection and recovery
// Addresses Tasks 1-5: Architecture, Detection, Fork Point Discovery, Orphaned Block Marking, Integration
type ReorgHandlerImpl struct {
	rpcClient RPCBlockFetcher // For fetching blockchain block hashes during fork point discovery
	store     BlockStoreExtended // Extended interface with MarkBlocksOrphaned method
	config    *ReorgConfig

	// Metrics (AC5: Observability)
	reorgDetectedTotal  uint64 // Counter: total reorgs detected
	reorgDepth          uint64 // Gauge: depth of last reorg
	orphanedBlocksTotal uint64 // Counter: total orphaned blocks marked
}

// BlockStoreExtended extends BlockStore with orphaned block marking capability
// Addresses Task 1.1: Design interface extension for storage layer
type BlockStoreExtended interface {
	BlockStore
	// MarkBlocksOrphaned marks all blocks in the height range [startHeight, endHeight] as orphaned
	// Uses database transaction for atomicity (AC3)
	MarkBlocksOrphaned(ctx context.Context, startHeight, endHeight uint64) error
}

// NewReorgHandler creates a new reorg handler with the provided configuration
// Addresses Task 1.1: ReorgHandler struct with store, max depth, logger
func NewReorgHandler(
	rpcClient RPCBlockFetcher,
	store BlockStoreExtended,
	config *ReorgConfig,
) (*ReorgHandlerImpl, error) {
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

	util.Info("reorg handler initialized",
		"max_depth", config.MaxDepth,
	)

	return &ReorgHandlerImpl{
		rpcClient: rpcClient,
		store:     store,
		config:    config,
	}, nil
}

// HandleReorg is the main entry point for reorg handling triggered by live-tail coordinator
// Implements AC1: Reorg Detection, AC2: Fork Point Discovery, AC3: Orphaned Block Marking
// Addresses Task 2: Implement reorg detection
func (rh *ReorgHandlerImpl) HandleReorg(ctx context.Context, newBlock *Block) error {
	// AC1: Get current database head for comparison (Task 2.4)
	dbHead, err := rh.store.GetLatestBlock(ctx)
	if err != nil {
		util.Error("failed to get database head during reorg",
			"error", err.Error(),
			"new_block_height", newBlock.Height,
		)
		return fmt.Errorf("failed to get database head: %w", err)
	}

	// AC1: Log reorg detection event with structured context (Task 2.3)
	util.Warn("reorg detected - parent hash mismatch",
		"new_block_height", newBlock.Height,
		"new_block_hash", fmt.Sprintf("%x", newBlock.Hash),
		"new_block_parent", fmt.Sprintf("%x", newBlock.ParentHash),
		"db_head_height", dbHead.Height,
		"db_head_hash", fmt.Sprintf("%x", dbHead.Hash),
	)

	// AC1: Calculate initial reorg depth estimate (Task 2.5)
	// IMPORTANT: This is a fast-fail optimization based on height difference only.
	// The actual reorg depth is determined after fork point discovery and may differ.
	// Secondary validation after fork point discovery (line 113) catches edge cases where
	// the initial estimate is incorrect. This check provides early rejection for obviously
	// deep reorgs without expensive backwards walk.
	initialDepth := newBlock.Height - dbHead.Height
	if initialDepth > uint64(rh.config.MaxDepth) {
		// AC1: Return error if depth exceeds maximum immediately (Task 2.6)
		util.Error("reorg depth exceeds maximum - manual intervention required",
			"initial_depth", initialDepth,
			"max_depth", rh.config.MaxDepth,
			"new_block_height", newBlock.Height,
			"db_head_height", dbHead.Height,
		)
		return fmt.Errorf("reorg depth (%d) exceeds maximum (%d) - manual intervention required",
			initialDepth, rh.config.MaxDepth)
	}

	// AC2: Find fork point by walking backwards (Task 3)
	forkPointHeight, err := rh.findForkPoint(ctx, dbHead)
	if err != nil {
		util.Error("failed to find fork point",
			"error", err.Error(),
			"db_head_height", dbHead.Height,
		)
		return fmt.Errorf("failed to find fork point: %w", err)
	}

	// Calculate actual reorg depth
	actualDepth := dbHead.Height - forkPointHeight

	util.Info("fork point found",
		"fork_point_height", forkPointHeight,
		"reorg_depth", actualDepth,
		"orphaned_blocks_start", forkPointHeight+1,
		"orphaned_blocks_end", dbHead.Height,
	)

	// AC5: Update metrics (Task 6.5)
	atomic.AddUint64(&rh.reorgDetectedTotal, 1)
	atomic.StoreUint64(&rh.reorgDepth, actualDepth)

	// AC3: Mark orphaned blocks from fork point + 1 to current head (Task 4)
	if forkPointHeight < dbHead.Height {
		// There are blocks to mark as orphaned
		if err := rh.markOrphanedBlocks(ctx, forkPointHeight+1, dbHead.Height); err != nil {
			util.Error("failed to mark orphaned blocks",
				"error", err.Error(),
				"fork_point", forkPointHeight,
				"db_head", dbHead.Height,
			)
			return fmt.Errorf("failed to mark orphaned blocks: %w", err)
		}

		// Update orphaned blocks counter
		orphanedCount := dbHead.Height - forkPointHeight
		atomic.AddUint64(&rh.orphanedBlocksTotal, orphanedCount)

		util.Info("orphaned blocks marked successfully",
			"fork_point", forkPointHeight,
			"orphaned_count", orphanedCount,
			"start_height", forkPointHeight+1,
			"end_height", dbHead.Height,
		)
	}

	// AC4: Return success - live-tail will resume normal processing (Task 5.1)
	util.Info("reorg handling completed - live-tail will resume from fork point",
		"fork_point", forkPointHeight,
		"next_height_to_process", forkPointHeight+1,
	)

	return nil
}

// findForkPoint walks backwards from current database head to find common ancestor block
// Implements AC2: Fork Point Discovery
// Addresses Task 3: Implement fork point discovery
func (rh *ReorgHandlerImpl) findForkPoint(ctx context.Context, dbHead *Block) (uint64, error) {
	currentHeight := dbHead.Height
	searchDepth := 0

	util.Debug("starting fork point search",
		"start_height", currentHeight,
		"max_depth", rh.config.MaxDepth,
	)

	// Task 3.2: Walk backwards from current head height
	for searchDepth <= rh.config.MaxDepth {
		// Task 3.7: Log each step of fork point search for debugging
		util.Debug("checking fork point candidate",
			"height", currentHeight,
			"search_depth", searchDepth,
		)

		// Handle genesis block (always a fork point)
		if currentHeight == 0 {
			util.Info("reached genesis block - using as fork point",
				"fork_point_height", 0,
			)
			return 0, nil
		}

		// Task 3.3: Fetch blockchain block hash for this height via RPC
		chainBlock, err := rh.rpcClient.GetBlockByNumber(ctx, currentHeight)
		if err != nil {
			// RPC error - cannot continue fork point search
			util.Error("rpc fetch failed during fork point search",
				"error", err.Error(),
				"height", currentHeight,
				"search_depth", searchDepth,
			)
			return 0, fmt.Errorf("failed to fetch blockchain block at height %d: %w", currentHeight, err)
		}

		// Task 3.4: Fetch database block hash at same height
		dbBlock, err := rh.store.GetBlockByHeight(ctx, currentHeight)
		if err != nil {
			// Database error - cannot continue fork point search
			util.Error("database fetch failed during fork point search",
				"error", err.Error(),
				"height", currentHeight,
				"search_depth", searchDepth,
			)
			return 0, fmt.Errorf("failed to fetch database block at height %d: %w", currentHeight, err)
		}

		// Task 3.4: Compare blockchain hash with database hash
		chainHash := chainBlock.Hash().Bytes()
		if bytesEqual(chainHash, dbBlock.Hash) {
			// Task 3.5: Hashes match - fork point found!
			util.Info("fork point found - hashes match",
				"fork_point_height", currentHeight,
				"hash", fmt.Sprintf("%x", chainHash),
				"search_depth", searchDepth,
			)
			return currentHeight, nil
		}

		// Hashes don't match - continue searching backwards
		util.Debug("hashes mismatch - continuing search",
			"height", currentHeight,
			"chain_hash", fmt.Sprintf("%x", chainHash),
			"db_hash", fmt.Sprintf("%x", dbBlock.Hash),
		)

		// Move backwards
		currentHeight--
		searchDepth++
	}

	// Task 3.6: Return error if max depth exceeded without finding fork point
	util.Error("fork point not found within max depth",
		"max_depth", rh.config.MaxDepth,
		"start_height", dbHead.Height,
		"end_height", currentHeight,
	)
	return 0, fmt.Errorf("fork point not found within max depth (%d blocks)", rh.config.MaxDepth)
}

// markOrphanedBlocks marks all blocks in the range [startHeight, endHeight] as orphaned
// Implements AC3: Orphaned Block Marking with database transaction
// Addresses Task 4: Implement orphaned block marking
func (rh *ReorgHandlerImpl) markOrphanedBlocks(ctx context.Context, startHeight, endHeight uint64) error {
	// Task 4.2: Calculate range of blocks to mark orphaned
	blockCount := endHeight - startHeight + 1

	util.Info("marking blocks as orphaned",
		"start_height", startHeight,
		"end_height", endHeight,
		"block_count", blockCount,
	)

	// Task 4.3-4.6: Begin database transaction, execute UPDATE, commit or rollback
	// Note: The BlockStoreExtended.MarkBlocksOrphaned method handles transaction internally
	// This follows the pattern where the storage layer manages transaction boundaries
	if err := rh.store.MarkBlocksOrphaned(ctx, startHeight, endHeight); err != nil {
		// Task 4.6: Rollback occurs automatically in storage layer on error
		util.Error("failed to mark blocks as orphaned",
			"error", err.Error(),
			"start_height", startHeight,
			"end_height", endHeight,
		)
		return fmt.Errorf("failed to mark blocks as orphaned (height %d to %d): %w",
			startHeight, endHeight, err)
	}

	// Task 4.5: Log success with block count
	util.Info("successfully marked blocks as orphaned",
		"start_height", startHeight,
		"end_height", endHeight,
		"block_count", blockCount,
	)

	return nil
}

// Stats returns reorg handler statistics for observability
// Implements AC5: Metrics collection
// Addresses Task 6: Configuration and metrics
func (rh *ReorgHandlerImpl) Stats() map[string]interface{} {
	return map[string]interface{}{
		"reorg_detected_total":  atomic.LoadUint64(&rh.reorgDetectedTotal),
		"reorg_depth":           atomic.LoadUint64(&rh.reorgDepth),
		"orphaned_blocks_total": atomic.LoadUint64(&rh.orphanedBlocksTotal),
		"max_depth":             rh.config.MaxDepth,
	}
}
