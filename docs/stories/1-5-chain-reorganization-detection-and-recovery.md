# Story 1.5: Chain Reorganization Detection and Recovery

Status: drafted

## Story

As a **blockchain indexer system**,
I want **automatic detection and recovery from chain reorganizations (reorgs) up to 6 blocks deep**,
so that **I maintain data integrity by marking orphaned blocks and re-indexing the canonical chain from the fork point**.

## Acceptance Criteria

1. **AC1: Reorg Detection**
   - Detects reorg when new block's parent_hash doesn't match database head hash
   - Triggered by live-tail coordinator during sequential block processing
   - Logs reorg detection event with structured context (height, expected parent, actual parent)
   - Returns error to live-tail if reorg depth exceeds maximum (default: 6 blocks)

2. **AC2: Fork Point Discovery**
   - Walks backwards from current database head to find common ancestor block
   - Compares block hashes at each height between database and blockchain
   - Stops when hashes match (fork point found)
   - Limits search depth to configurable maximum (default: 6 blocks)
   - Returns error if fork point not found within depth limit

3. **AC3: Orphaned Block Marking**
   - Marks all blocks from fork point + 1 to current database head as orphaned
   - Updates `orphaned` flag to TRUE without deleting blocks (soft delete pattern)
   - Uses database transaction to ensure atomicity
   - Preserves orphaned blocks and their transactions for audit trail
   - Foreign key cascade behavior maintains referential integrity

4. **AC4: Canonical Chain Re-indexing**
   - After marking orphaned blocks, live-tail resumes normal processing
   - Fetches and indexes canonical blocks from fork point + 1 forward
   - Inserts canonical blocks with orphaned=false
   - Maintains sequential processing (one block at a time)
   - Reorg recovery completes when database head matches network head

5. **AC5: Configuration and Observability**
   - Maximum reorg depth configurable via `REORG_MAX_DEPTH` environment variable (default: 6)
   - Structured logging for: reorg detected, fork point found, blocks marked orphaned, recovery complete
   - Metrics: reorg_detected_total counter, reorg_depth gauge, orphaned_blocks_total counter
   - Reorg depth and impacted block range included in all log events

## Tasks / Subtasks

- [ ] **Task 1: Design reorg handler architecture** (AC: #1, #5)
  - [ ] Subtask 1.1: Design `ReorgHandler` struct with store, max depth, logger
  - [ ] Subtask 1.2: Design reorg detection trigger from live-tail coordinator
  - [ ] Subtask 1.3: Design fork point discovery algorithm (backwards walk)
  - [ ] Subtask 1.4: Design orphaned block marking strategy (database transaction)
  - [ ] Subtask 1.5: Document integration with live-tail and storage layer

- [ ] **Task 2: Implement reorg detection** (AC: #1)
  - [ ] Subtask 2.1: Create `internal/index/reorg.go` with ReorgHandler struct
  - [ ] Subtask 2.2: Implement `HandleReorg(ctx, newBlock)` method as entry point
  - [ ] Subtask 2.3: Log reorg detection event with structured fields (height, hashes)
  - [ ] Subtask 2.4: Extract current database head for comparison
  - [ ] Subtask 2.5: Calculate initial reorg depth estimate
  - [ ] Subtask 2.6: Return error if depth exceeds maximum immediately

- [ ] **Task 3: Implement fork point discovery** (AC: #2)
  - [ ] Subtask 3.1: Implement `findForkPoint(ctx, newBlock)` method
  - [ ] Subtask 3.2: Walk backwards from current head height
  - [ ] Subtask 3.3: Fetch blockchain block hash for each height (via RPC)
  - [ ] Subtask 3.4: Compare with database block hash at same height
  - [ ] Subtask 3.5: Return fork point height when hashes match
  - [ ] Subtask 3.6: Return error if max depth exceeded without finding fork point
  - [ ] Subtask 3.7: Log each step of fork point search for debugging

- [ ] **Task 4: Implement orphaned block marking** (AC: #3)
  - [ ] Subtask 4.1: Implement `markOrphanedBlocks(ctx, forkPoint, currentHead)` method
  - [ ] Subtask 4.2: Calculate range of blocks to mark orphaned (forkPoint+1 to currentHead)
  - [ ] Subtask 4.3: Begin database transaction for atomicity
  - [ ] Subtask 4.4: Execute UPDATE statement to set orphaned=true for block range
  - [ ] Subtask 4.5: Commit transaction and log success with block count
  - [ ] Subtask 4.6: Rollback transaction on error and return error context
  - [ ] Subtask 4.7: Verify foreign key cascade behavior preserves transactions/logs

- [ ] **Task 5: Integrate with live-tail for canonical re-indexing** (AC: #4)
  - [ ] Subtask 5.1: Return success from HandleReorg after marking orphaned blocks
  - [ ] Subtask 5.2: Live-tail resumes normal processing from fork point + 1
  - [ ] Subtask 5.3: Verify live-tail fetches canonical blocks sequentially
  - [ ] Subtask 5.4: Log canonical chain re-indexing progress
  - [ ] Subtask 5.5: Test end-to-end reorg recovery (detection → marking → re-index)

- [ ] **Task 6: Add configuration and metrics** (AC: #5)
  - [ ] Subtask 6.1: Create `internal/index/reorg_config.go` with configuration struct
  - [ ] Subtask 6.2: Load max depth from `REORG_MAX_DEPTH` environment variable (default: 6)
  - [ ] Subtask 6.3: Add Prometheus metrics (reorg_detected_total, reorg_depth, orphaned_blocks_total)
  - [ ] Subtask 6.4: Log reorg events with structured fields (depth, fork point, block range)
  - [ ] Subtask 6.5: Update metrics in HandleReorg method

- [ ] **Task 7: Write comprehensive tests** (AC: #1-#5)
  - [ ] Subtask 7.1: Create `internal/index/reorg_test.go` with mocked store and RPC
  - [ ] Subtask 7.2: Test reorg detection (parent hash mismatch scenario)
  - [ ] Subtask 7.3: Test fork point discovery (3-block reorg)
  - [ ] Subtask 7.4: Test maximum depth (6 blocks)
  - [ ] Subtask 7.5: Test depth exceeded error (7+ blocks)
  - [ ] Subtask 7.6: Test orphaned block marking (verify UPDATE statement)
  - [ ] Subtask 7.7: Test configuration validation and loading
  - [ ] Subtask 7.8: Test end-to-end integration with live-tail (mock scenario)
  - [ ] Subtask 7.9: Achieve >70% test coverage for reorg package

## Dev Notes

### Architecture Context

**Component:** `internal/index/` package (extends indexing layer with reorg handling)

**Key Design Patterns:**
- **Reorg Detection**: Triggered by parent hash mismatch in live-tail
- **Fork Point Discovery**: Backwards walk comparing blockchain vs. database hashes
- **Soft Delete**: Mark orphaned blocks (orphaned=true) rather than hard delete
- **Transactional Marking**: Use database transaction for atomicity
- **Resumable Recovery**: Live-tail resumes normal processing after marking

**Integration Points:**
- **Live-Tail Coordinator** (`internal/index/LiveTailCoordinator`): Triggers reorg handler on parent mismatch
- **RPC Client** (`internal/rpc/Client`): Fetch blockchain block hashes for comparison
- **Storage Layer** (`internal/store/Store`): Query database blocks, mark orphaned via UPDATE
- **Metrics** (Story 1.7): Prometheus metrics for reorg events

**Technology Stack:**
- Go database transactions: pgx.Tx for atomic updates
- Structured logging: log/slog for reorg events
- Prometheus metrics: counters and gauges for observability

### Project Structure Notes

**Files to Create:**
```
internal/index/
├── reorg.go           # ReorgHandler, HandleReorg, findForkPoint, markOrphanedBlocks
├── reorg_config.go    # Configuration struct and environment loading
└── reorg_test.go      # Unit and integration tests with mocks
```

**Configuration:**
```bash
REORG_MAX_DEPTH=6          # Maximum reorg depth to handle (default: 6 blocks)
```

### Reorg Handling Algorithm

**High-Level Flow:**
1. Live-tail detects parent hash mismatch → triggers ReorgHandler
2. ReorgHandler calculates current depth (newBlock.Height - dbHead.Height)
3. If depth > max, return error (manual intervention required)
4. Walk backwards from dbHead to find fork point (matching hashes)
5. Mark blocks from forkPoint+1 to dbHead as orphaned (UPDATE statement)
6. Return success → live-tail resumes, fetches canonical blocks from forkPoint+1

**Fork Point Discovery Pseudocode:**
```
forkPoint = findForkPoint(ctx, newBlock):
    currentHeight = dbHead.Height
    for depth = 0 to maxDepth:
        if currentHeight == 0:
            return 0  // Genesis block is always fork point

        // Fetch blockchain hash at currentHeight
        chainBlock = rpcClient.GetBlockByNumber(ctx, currentHeight)

        // Fetch database hash at currentHeight
        dbBlock = store.GetBlockByHeight(ctx, currentHeight)

        if chainBlock.Hash == dbBlock.Hash:
            return currentHeight  // Fork point found

        currentHeight--

    return error  // Fork point not found within maxDepth
```

**Orphaned Block Marking SQL:**
```sql
BEGIN;
UPDATE blocks SET orphaned = true WHERE height >= $1 AND height <= $2;
COMMIT;
```

### Data Integrity Considerations

**Soft Delete Rationale:**
- Preserves audit trail of orphaned chains
- Allows forensic analysis of reorgs
- Enables recovery if needed (re-mark as canonical if error)

**Foreign Key Cascade Behavior:**
- Transactions and logs remain in database with orphaned=true blocks
- Foreign keys: `transactions.block_height → blocks.height ON DELETE CASCADE`
- Orphaned flag query: `SELECT * FROM blocks WHERE orphaned = false` (canonical chain)

**Reorg Recovery Guarantee:**
- After marking orphaned blocks, database head = forkPoint
- Live-tail resumes from forkPoint + 1
- Canonical blocks inserted with orphaned=false
- Database eventually consistent with blockchain canonical chain

### Error Handling Strategy

**Reorg Depth Exceeded:**
- Depth > maxDepth (default: 6 blocks) triggers error
- Live-tail halts indexing (no automatic recovery)
- Manual intervention required: inspect reorg depth, adjust maxDepth, or manual recovery
- Log ERROR with depth, fork point, and recommended action

**Fork Point Not Found:**
- Searched maxDepth blocks without finding matching hash
- Possible causes: deep reorg, database corruption, wrong network
- Return error to live-tail (halts indexing)
- Log ERROR with search range and database vs. blockchain hashes

**Database Transaction Failure:**
- BEGIN or COMMIT fails during orphaned block marking
- Rollback transaction automatically (pgx behavior)
- Return error to live-tail (halts indexing)
- Log ERROR with transaction context and retry recommendation

**RPC Fetch Failure:**
- RPC client fails to fetch blockchain block hash during fork point discovery
- RPC retry logic handles transient failures automatically
- If permanent failure, return error to live-tail
- Log ERROR with RPC context and height

### Performance Considerations

**Fork Point Search:**
- Worst case: maxDepth RPC calls + maxDepth database queries
- 6-block reorg: ~12 queries total (6 RPC + 6 DB)
- Typical latency: 1-2 seconds (assuming 100ms per RPC call)

**Orphaned Block Marking:**
- Single UPDATE statement (atomic, fast)
- Indexed by height: `idx_blocks_orphaned_height` supports query
- Transaction overhead: negligible (<10ms)

**Canonical Re-indexing:**
- Live-tail processes canonical blocks sequentially
- 6-block reorg: ~12-18 seconds to fully recover (2-3s per block)
- During recovery, database head lags network head temporarily

### Testing Strategy

**Unit Test Coverage Target:** >70% for reorg package

**Test Scenarios:**
1. **Reorg Detection**: Mock parent hash mismatch, verify HandleReorg called
2. **Fork Point Discovery - 3 Blocks**: Mock 3-block reorg, verify fork point found
3. **Fork Point Discovery - 6 Blocks**: Max depth scenario, verify success
4. **Depth Exceeded**: Mock 7-block reorg, verify error returned
5. **Orphaned Block Marking**: Verify UPDATE statement executed with correct range
6. **Configuration**: Test env var loading and defaults
7. **End-to-End Integration**: Mock live-tail → reorg → recovery flow
8. **Database Transaction**: Test commit success and rollback on error
9. **RPC Fetch Failure**: Mock RPC error during fork point discovery

**Mocking Strategy:**
- Mock `store.Store` interface for database operations (GetBlockByHeight, MarkBlocksOrphaned)
- Mock `rpc.Client` interface (RPCBlockFetcher) for blockchain block fetches
- Use in-memory state to simulate database and blockchain states
- Test transaction behavior with mock Store that tracks BEGIN/COMMIT/ROLLBACK

### Learnings from Previous Story (Story 1.4)

**From Story 1-4-live-tail-mechanism-for-new-blocks (Status: review)**

- **Integration Point Created**: ReorgHandler interface stub in live-tail (livetail.go:167-183)
  - **IMPLEMENT Interface**: Story 1.5 implements the actual ReorgHandler interface
  - Live-tail already calls `reorgHandler.HandleReorg(ctx, newBlock)` on parent mismatch
  - Current stub is no-op; Story 1.5 provides real implementation

- **Parent Hash Validation Logic**: Live-tail checks parent hash before insertion (livetail.go:167-171)
  - **REUSE Pattern**: Story 1.5 leverages this detection mechanism
  - Reorg triggered when: `!bytesEqual(newBlock.ParentHash, dbHead.Hash)`
  - Live-tail passes `newBlock` to reorg handler with full context

- **Error Handling Pattern**: Log and continue vs. halt (livetail.go:111-118)
  - Story 1.4: Transient errors → log and continue
  - **Story 1.5**: Reorg errors → return error to halt live-tail (requires manual review)
  - Deep reorgs (>6 blocks) should halt indexing for safety

- **Configuration Pattern**: Environment variables with sensible defaults
  - Story 1.4: `LIVETAIL_POLL_INTERVAL` (default: 2s)
  - **Story 1.5**: `REORG_MAX_DEPTH` (default: 6 blocks) - follow same env var pattern

- **Structured Logging Pattern**: JSON output with slog (livetail.go:192-201)
  - Follow same pattern for reorg events: `logger.Warn("reorg detected", slog.Uint64("height", height), slog.Int("depth", depth))`

- **Testing Setup**: Interface-based mocking for testability
  - Story 1.4: RPCBlockFetcher, BlockStore, ReorgHandler interfaces
  - **Story 1.5**: Implement ReorgHandler interface, mock Store and RPC for tests

- **Metrics Collection**: Stats method returns metrics map (livetail.go:225-239)
  - Story 1.4: blockCount, lagSeconds, networkHead gauges
  - **Story 1.5**: Add reorg-specific metrics (reorgDetectedTotal, reorgDepth, orphanedBlocksTotal)

- **Context Propagation**: All methods accept context.Context (livetail.go:90)
  - Story 1.4: `Start(ctx)`, `processNextBlock(ctx)`
  - **Story 1.5**: `HandleReorg(ctx, newBlock)` - follow same pattern

- **Atomic Operations**: Thread-safe metrics without locks (livetail.go:204-223)
  - Story 1.4: Uses atomic.LoadUint64, atomic.StoreUint64
  - **Story 1.5**: Follow same pattern for reorg metrics (atomic counters/gauges)

- **Files Created in Story 1.4**:
  - `internal/index/livetail.go` - Coordinator with reorg detection
  - `internal/index/livetail_config.go` - Configuration management
  - `internal/index/livetail_test.go` - Test suite with 82% coverage

- **Key Interfaces to Implement**:
  ```go
  // Already defined in internal/index/livetail.go (stub)
  type ReorgHandler interface {
      HandleReorg(ctx context.Context, newBlock *Block) error
  }
  ```

- **Testing Patterns to Follow**:
  - Table-driven tests for configuration validation
  - Mock with interfaces (MockStore, MockRPCClient)
  - Context cancellation tests
  - Integration tests with realistic reorg scenarios

- **Technical Debt from Story 1.4**:
  - ReorgHandler is stub (Story 1.5 resolves this)
  - Prometheus metrics collected but not emitted (Story 1.7 handles emission)

[Source: stories/1-4-live-tail-mechanism-for-new-blocks.md#Dev-Agent-Record]

### References

- [Source: docs/tech-spec-epic-1.md#Story-1.5-Chain-Reorganization-Detection-and-Recovery]
- [Source: docs/tech-spec-epic-1.md#Data-Integrity-Rules]
- [Source: docs/solution-architecture.md#Reorg-Handler]
- [Source: docs/PRD.md#FR003-Chain-Reorganization-Handling]
- [Source: stories/1-4-live-tail-mechanism-for-new-blocks.md#Dev-Agent-Record]
- [Source: docs/epic-stories.md#Story-1.5]

---

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

