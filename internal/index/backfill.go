package index

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// RPCBlockFetcher interface for fetching blocks (allows testing with mocks)
type RPCBlockFetcher interface {
	GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error)
}

// BackfillCoordinator manages parallel block backfilling with worker pool pattern
type BackfillCoordinator struct {
	rpcClient RPCBlockFetcher
	config    *Config

	// Metrics
	blocksFetched    int64
	blocksInserted   int64
	batchesProcessed int64
	startTime        time.Time
}

// BlockResult represents a fetched block with metadata
type BlockResult struct {
	Height   uint64
	Block    *types.Block
	WorkerID int
	Error    error
}

// NewBackfillCoordinator creates a new backfill coordinator
func NewBackfillCoordinator(rpcClient RPCBlockFetcher, config *Config) (*BackfillCoordinator, error) {
	if rpcClient == nil {
		return nil, fmt.Errorf("rpcClient cannot be nil")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &BackfillCoordinator{
		rpcClient: rpcClient,
		config:    config,
	}, nil
}

// Backfill executes the backfill operation with parallel workers
// Implements AC1 (Worker Pool Architecture), AC2 (Performance), AC3 (Error Handling)
func (bc *BackfillCoordinator) Backfill(ctx context.Context, startHeight, endHeight uint64) error {
	// Validate height range
	if startHeight > endHeight {
		return fmt.Errorf("startHeight (%d) must be <= endHeight (%d)", startHeight, endHeight)
	}

	bc.startTime = time.Now()
	totalBlocks := endHeight - startHeight + 1

	util.Info("starting backfill",
		"start_height", startHeight,
		"end_height", endHeight,
		"total_blocks", totalBlocks,
		"workers", bc.config.Workers,
		"batch_size", bc.config.BatchSize,
	)

	// Create channels for communication
	jobQueue := make(chan uint64, bc.config.Workers*2) // Buffered to avoid blocking
	resultChan := make(chan *BlockResult, bc.config.Workers*2)
	errorChan := make(chan *WorkerError, 1) // Buffered to non-blocking send

	// WaitGroup for coordination
	var workerWg sync.WaitGroup
	var collectorWg sync.WaitGroup

	// Track if backfill was halted due to error
	var haltOnce sync.Once
	haltBackfill := false

	// Start worker goroutines
	workerWg.Add(bc.config.Workers)
	for i := 0; i < bc.config.Workers; i++ {
		go bc.worker(ctx, i, &workerWg, jobQueue, resultChan, errorChan, &haltBackfill, &haltOnce)
	}

	// Start result collector goroutine
	collectorWg.Add(1)
	collectedBlocks := make([]*types.Block, 0, bc.config.BatchSize)
	collectedHeights := make([]uint64, 0, bc.config.BatchSize)
	go func() {
		defer collectorWg.Done()
		// For now, just consume results (actual DB insertion in future)
		for result := range resultChan {
			if result.Error == nil {
				collectedBlocks = append(collectedBlocks, result.Block)
				collectedHeights = append(collectedHeights, result.Height)

				if len(collectedBlocks) >= bc.config.BatchSize {
					util.Debug("batch complete",
						"batch_size", len(collectedBlocks),
						"batch_count", int(bc.batchesProcessed)+1,
					)
					bc.batchesProcessed++
					bc.blocksInserted += int64(len(collectedBlocks))
					collectedBlocks = collectedBlocks[:0]
					collectedHeights = collectedHeights[:0]
				}
			}
		}

		// Flush remaining blocks
		if len(collectedBlocks) > 0 {
			util.Debug("flushing remaining blocks",
				"batch_size", len(collectedBlocks),
			)
			bc.batchesProcessed++
			bc.blocksInserted += int64(len(collectedBlocks))
		}
	}()

	// Send jobs to worker queue
	go func() {
		defer close(jobQueue)
		for height := startHeight; height <= endHeight; height++ {
			select {
			case <-ctx.Done():
				util.Info("context cancelled, stopping job distribution")
				return
			default:
			}

			// Check if error occurred in workers
			select {
			case workerErr := <-errorChan:
				// Error received from worker - halt backfill
				util.Error("permanent error from worker, halting backfill",
					"worker_id", workerErr.WorkerID,
					"height", workerErr.Height,
					"error", workerErr.Error.Error(),
				)
				haltOnce.Do(func() {
					haltBackfill = true
				})
				return
			default:
			}

			// Send job to queue (non-blocking; if queue full, this might block slightly)
			jobQueue <- height
			bc.blocksFetched++
		}
	}()

	// Wait for all workers to finish
	workerWg.Wait()
	close(resultChan)
	collectorWg.Wait()

	// Check if there was an error
	select {
	case workerErr := <-errorChan:
		duration := time.Since(bc.startTime)
		util.Error("backfill failed",
			"duration", duration.String(),
			"blocks_fetched", bc.blocksFetched,
			"blocks_inserted", bc.blocksInserted,
			"worker_id", workerErr.WorkerID,
			"height", workerErr.Height,
			"error", workerErr.Error.Error(),
		)
		return fmt.Errorf("backfill failed at height %d (worker %d): %w",
			workerErr.Height, workerErr.WorkerID, workerErr.Error)
	default:
	}

	// Log completion summary
	duration := time.Since(bc.startTime)
	throughputPerSec := float64(bc.blocksFetched) / duration.Seconds()

	util.Info("backfill completed successfully",
		"duration", duration.String(),
		"blocks_fetched", bc.blocksFetched,
		"blocks_inserted", bc.blocksInserted,
		"batches_processed", bc.batchesProcessed,
		"throughput_blocks_per_second", fmt.Sprintf("%.2f", throughputPerSec),
	)

	return nil
}

// WorkerError represents an error from a worker with context
type WorkerError struct {
	WorkerID int
	Height   uint64
	Error    error
}

// worker is a goroutine that fetches blocks from RPC and sends them to result channel
// Implements AC1, AC2, AC3
func (bc *BackfillCoordinator) worker(
	ctx context.Context,
	workerID int,
	wg *sync.WaitGroup,
	jobQueue <-chan uint64,
	resultChan chan<- *BlockResult,
	errorChan chan<- *WorkerError,
	haltFlag *bool,
	haltOnce *sync.Once,
) {
	defer wg.Done()

	util.Debug("worker started",
		"worker_id", workerID,
	)

	for height := range jobQueue {
		// Check if halt flag is set
		if *haltFlag {
			util.Debug("worker detected halt flag, stopping",
				"worker_id", workerID,
			)
			continue // consume remaining items in queue
		}

		// Fetch block with timeout
		workerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		block, err := bc.rpcClient.GetBlockByNumber(workerCtx, height)
		cancel()

		if err != nil {
			// Permanent error - halt backfill
			util.Error("worker encountered error",
				"worker_id", workerID,
				"height", height,
				"error", err.Error(),
			)

			workerErr := &WorkerError{
				WorkerID: workerID,
				Height:   height,
				Error:    err,
			}

			// Send error (non-blocking; ignore if channel already has error)
			select {
			case errorChan <- workerErr:
			default:
			}

			haltOnce.Do(func() {
				*haltFlag = true
			})
			continue
		}

		// Send result to collector
		result := &BlockResult{
			Height:   height,
			Block:    block,
			WorkerID: workerID,
			Error:    nil,
		}

		select {
		case resultChan <- result:
		case <-ctx.Done():
			util.Debug("context cancelled, worker exiting",
				"worker_id", workerID,
			)
			return
		}
	}

	util.Debug("worker finished",
		"worker_id", workerID,
	)
}

// Stats returns backfill statistics
func (bc *BackfillCoordinator) Stats() map[string]interface{} {
	return map[string]interface{}{
		"blocks_fetched":     bc.blocksFetched,
		"blocks_inserted":    bc.blocksInserted,
		"batches_processed":  bc.batchesProcessed,
		"duration":           time.Since(bc.startTime),
		"workers":            bc.config.Workers,
		"batch_size":         bc.config.BatchSize,
	}
}

// BackfillWithConfig executes backfill using stored config
func (bc *BackfillCoordinator) BackfillWithConfig(ctx context.Context) error {
	return bc.Backfill(ctx, bc.config.StartHeight, bc.config.EndHeight)
}
