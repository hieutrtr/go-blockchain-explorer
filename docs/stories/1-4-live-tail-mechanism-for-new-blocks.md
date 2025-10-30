# Story 1.4: Live-Tail Mechanism for New Blocks

Status: ready-for-dev

## Story

As a **blockchain indexer system**,
I want **a live-tail coordinator that sequentially processes new blocks as they are produced on the Ethereum network**,
so that **I can maintain near real-time synchronization with the blockchain head with <2 second lag**.

## Acceptance Criteria

1. **AC1: Sequential Block Processing**
   - Fetches next block sequentially from database head + 1
   - Processes blocks one at a time (no parallelization for live-tail)
   - Queries database for current head height before each fetch
   - Maintains strict height ordering (no gaps or duplicates)

2. **AC2: Polling and Timing**
   - Polls for new blocks at configurable interval (default: 2 seconds)
   - Uses time.Ticker for precise polling cadence
   - Lag between network head and database head < 2 seconds (averaged over 1 minute)
   - Handles "block not found" gracefully (network hasn't produced next block yet)

3. **AC3: Error Handling and Resilience**
   - Continues processing despite transient RPC errors (no halt)
   - Logs errors with context but doesn't exit
   - RPC client retry logic handles transient failures automatically
   - Permanent errors (e.g., network unreachable) logged but polling continues

4. **AC4: Integration with Reorg Handler**
   - Checks for parent hash mismatch before inserting block
   - Triggers reorg handler if parent hash doesn't match database head
   - Delegates reorg handling to ReorgHandler (Story 1.5 - not yet implemented, stub OK)
   - Continues normal processing after reorg resolution

5. **AC5: Configuration and Observability**
   - Poll interval configurable via `LIVETAIL_POLL_INTERVAL` environment variable
   - Structured logging for: block processed, parent mismatch detected, errors
   - Metrics: blocks_processed_total counter, current_head_height gauge, lag_seconds gauge
   - Context cancellation support for graceful shutdown

## Tasks / Subtasks

- [ ] **Task 1: Design live-tail coordinator architecture** (AC: #1, #5)
  - [ ] Subtask 1.1: Design `LiveTailCoordinator` struct with RPC client, ingester, store, poll interval
  - [ ] Subtask 1.2: Design sequential processing loop with time.Ticker
  - [ ] Subtask 1.3: Design database head query pattern before each block fetch
  - [ ] Subtask 1.4: Design context cancellation handling for graceful shutdown
  - [ ] Subtask 1.5: Document design patterns and integration points

- [ ] **Task 2: Implement sequential block processing** (AC: #1, #2)
  - [ ] Subtask 2.1: Create `internal/index/livetail.go` with LiveTailCoordinator struct
  - [ ] Subtask 2.2: Implement `Start(ctx)` method with ticker loop
  - [ ] Subtask 2.3: Implement `processNextBlock(ctx)` method with head query and block fetch
  - [ ] Subtask 2.4: Handle "block not found" case (next block not yet produced)
  - [ ] Subtask 2.5: Integrate with ingestion layer (parse block → domain model)
  - [ ] Subtask 2.6: Integrate with storage layer (insert block via store interface)

- [ ] **Task 3: Implement error handling and resilience** (AC: #3)
  - [ ] Subtask 3.1: Log errors with structured context (block height, error type)
  - [ ] Subtask 3.2: Continue processing after errors (don't halt coordinator)
  - [ ] Subtask 3.3: Rely on RPC client retry logic for transient failures
  - [ ] Subtask 3.4: Handle context cancellation gracefully (stop ticker, clean up)

- [ ] **Task 4: Integrate with reorg handler** (AC: #4)
  - [ ] Subtask 4.1: Query database for parent block hash
  - [ ] Subtask 4.2: Compare fetched block's parent hash with database head hash
  - [ ] Subtask 4.3: If mismatch detected, log warning and trigger reorg handler
  - [ ] Subtask 4.4: Create ReorgHandler interface stub (Story 1.5 not yet implemented)
  - [ ] Subtask 4.5: Pass reorg handling context to stub (no-op for now)

- [ ] **Task 5: Add configuration and metrics** (AC: #5)
  - [ ] Subtask 5.1: Create `internal/index/livetail_config.go` with configuration struct
  - [ ] Subtask 5.2: Load poll interval from `LIVETAIL_POLL_INTERVAL` environment variable (default: 2s)
  - [ ] Subtask 5.3: Add Prometheus metrics (blocks_processed_total, current_head_height, lag_seconds)
  - [ ] Subtask 5.4: Log block processing events with structured fields (height, hash, lag)
  - [ ] Subtask 5.5: Calculate and log lag between network head and database head

- [ ] **Task 6: Write comprehensive tests** (AC: #1-#5)
  - [ ] Subtask 6.1: Create `internal/index/livetail_test.go` with mocked RPC and database
  - [ ] Subtask 6.2: Test sequential processing (process 5 blocks in order)
  - [ ] Subtask 6.3: Test polling with ticker (verify 2-second cadence)
  - [ ] Subtask 6.4: Test "block not found" case (next block not yet produced)
  - [ ] Subtask 6.5: Test context cancellation (graceful shutdown)
  - [ ] Subtask 6.6: Test configuration validation and loading
  - [ ] Subtask 6.7: Test parent hash mismatch detection (reorg scenario)
  - [ ] Subtask 6.8: Achieve >70% test coverage for livetail package

## Dev Notes

### Architecture Context

**Component:** `internal/index/` package (extends indexing layer with live-tail)

**Key Design Patterns:**
- **Ticker Loop:** Uses time.Ticker for precise polling interval
- **Sequential Processing:** Processes blocks one at a time (no parallelization)
- **Head-Tracking:** Queries database for current head before each fetch
- **Error Resilience:** Logs and continues (doesn't halt on errors)
- **Reorg Detection:** Checks parent hash mismatch before insertion

**Integration Points:**
- **RPC Client** (`internal/rpc/Client`): Fetch blocks via `GetBlockByNumber()`
- **Ingester** (future): Parse RPC blocks into domain models (currently stub OK)
- **Storage Layer** (`internal/store/Store`): Query head, insert blocks
- **Reorg Handler** (future Story 1.5): Detect and handle parent hash mismatch

**Technology Stack:**
- Go concurrency primitives: time.Ticker, context.Context
- go-ethereum v1.16.5: Block types and RPC data structures
- Structured logging: log/slog (JSON output)
- Prometheus metrics: counters and gauges

### Project Structure Notes

**Files to Create:**
```
internal/index/
├── livetail.go           # LiveTailCoordinator, ticker loop, processNextBlock
├── livetail_config.go    # Configuration struct and environment loading
└── livetail_test.go      # Unit and integration tests with mocks
```

**Configuration:**
```bash
LIVETAIL_POLL_INTERVAL=2s          # Polling interval (default: 2 seconds)
```

### Performance Considerations

**Lag Target:**
- Target: <2 second lag from network head (averaged over 1 minute)
- Poll interval: 2 seconds → theoretical max lag ~2-4 seconds
- Actual lag depends on: RPC latency, database insert time, block production rate

**Sequential vs. Parallel:**
- Live-tail uses **sequential processing** (one block at a time)
- Backfill uses **parallel workers** (8 workers by default)
- Rationale: Live-tail prioritizes low lag and simplicity over throughput

**Database Query Optimization:**
- Query head height once per block (single query: `SELECT MAX(height) FROM blocks WHERE orphaned = false`)
- Parent hash verification (single query: `SELECT hash FROM blocks WHERE height = ?`)

### Error Handling Strategy

**Error Classification:**
- **Transient Errors:** Handled by RPC client retry logic (transparent to live-tail)
- **"Block Not Found":** Expected when next block not yet produced (log at DEBUG level, continue)
- **Permanent Errors:** Log at ERROR level, continue polling (don't halt)

**Reorg Detection:**
- Parent hash mismatch indicates potential reorg
- Trigger ReorgHandler (Story 1.5 - stub for now)
- After reorg resolution, continue normal processing

**Example Error Flow:**
1. Live-tail fetches block 1001
2. RPC returns error (transient network issue)
3. RPC client retries automatically (3 attempts with backoff)
4. If retry succeeds → block processed normally
5. If retry fails → log error, wait for next ticker event, retry block 1001

### Testing Strategy

**Unit Test Coverage Target:** >70% for livetail package

**Test Scenarios:**
1. **Sequential Processing:** Process 5 blocks in order, verify head tracking
2. **Ticker Cadence:** Verify ticker fires at 2-second intervals
3. **Block Not Found:** Mock RPC returns nil (next block not produced yet)
4. **Context Cancellation:** Cancel context mid-loop, verify cleanup
5. **Configuration:** Test env var loading and defaults
6. **Reorg Detection:** Mock parent hash mismatch, verify handler called
7. **Error Resilience:** Mock RPC errors, verify coordinator continues

**Mocking Strategy:**
- Mock `rpc.Client` interface (RPCBlockFetcher from Story 1.3)
- Mock `store.Store` interface for head queries and inserts
- Mock `ingest.Ingester` interface (or use stub for now)
- Mock `ReorgHandler` interface (Story 1.5 not yet implemented)

### Learnings from Previous Story (Story 1.3)

**From Story 1-3-parallel-backfill-worker-pool (Status: done)**

- **New Service Created**: `BackfillCoordinator` with worker pool pattern available at `internal/index/backfill.go`
  - **REUSE Pattern**: RPCBlockFetcher interface for testability (backfill.go:16-18)
  - Live-tail can use the same interface for mocking in tests

- **Architectural Decision**: Buffered channels (size = 2 * workers) prevent blocking
  - Live-tail uses sequential processing (no buffered channels needed for workers)
  - Single goroutine with ticker loop (simpler than worker pool)

- **Configuration Pattern**: Environment variables with sensible defaults
  - Story 1.3: `BACKFILL_WORKERS`, `BACKFILL_BATCH_SIZE`, `BACKFILL_START_HEIGHT`, `BACKFILL_END_HEIGHT`
  - Story 1.4: `LIVETAIL_POLL_INTERVAL` (follow same env var pattern)

- **Structured Logging Pattern**: JSON output to stdout with slog (backfill.go:72-103)
  - Follow same pattern for live-tail: `logger.Info("block processed", slog.Uint64("height", height), slog.Duration("lag", lag))`

- **Testing Setup**: RPCBlockFetcher interface enables mocking without external dependencies
  - **REUSE Interface**: Define RPCBlockFetcher in `internal/index/` (already exists from Story 1.3)
  - Live-tail tests can use MockRPCClient pattern from backfill_test.go

- **Metrics Collection**: Stats method returns metrics map (backfill.go:295-301)
  - Story 1.4: Follow same pattern for lag tracking and block count

- **Context Handling**: All methods accept context.Context for cancellation
  - Story 1.3: `Backfill(ctx, startHeight, endHeight)` - proper context propagation
  - Story 1.4: `Start(ctx)` - use context for ticker loop cancellation

- **Technical Debt**: Prometheus metrics collected but not emitted (Story 1.7 handles emission)
  - Story 1.4: Same approach - collect metrics, emit in Story 1.7

- **Files Created in Story 1.3**:
  - `internal/index/backfill.go` - Coordinator implementation
  - `internal/index/backfill_config.go` - Configuration management
  - `internal/index/backfill_test.go` - Comprehensive test suite (78% coverage)

- **Key Interfaces to Reuse**:
  ```go
  // Already defined in internal/index/backfill.go
  type RPCBlockFetcher interface {
      GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error)
  }
  ```

- **Testing Patterns to Follow**:
  - Table-driven tests for configuration validation (backfill_test.go:230-291)
  - Mock with interface (MockRPCClient implements RPCBlockFetcher)
  - Context cancellation tests (backfill_test.go:313-329)
  - Performance/throughput tests (backfill_test.go:523-550)

[Source: stories/1-3-parallel-backfill-worker-pool.md#Dev-Agent-Record]

### References

- [Source: docs/tech-spec-epic-1.md#Story-1.4-Live-Tail-Mechanism-for-New-Blocks]
- [Source: docs/tech-spec-epic-1.md#Component-Responsibilities]
- [Source: docs/solution-architecture.md#Indexing-Layer]
- [Source: docs/PRD.md#FR002-Real-Time-Block-Monitoring]
- [Source: stories/1-3-parallel-backfill-worker-pool.md#Dev-Agent-Record]

---

## Dev Agent Record

### Context Reference

- [Story Context XML](1-4-live-tail-mechanism-for-new-blocks.context.xml)

### Agent Model Used

Claude Sonnet 4.5 (claude-sonnet-4-5-20250929)

### Debug Log References

### Completion Notes List

### File List

---

## Change Log

- 2025-10-30: Initial story created from epic breakdown, sprint status, and previous learnings from Story 1.3
