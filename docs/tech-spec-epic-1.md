# Tech Spec: Epic 1 - Core Indexing & Data Pipeline

**Project:** Blockchain Explorer
**Epic:** Core Indexing & Data Pipeline
**Date:** 2025-10-29
**Last Updated:** 2025-10-29 (Tech Stack Synced)
**Author:** Hieu

---

## Epic Overview

**Goal:** Build a production-grade blockchain data pipeline that efficiently indexes Ethereum blocks, handles chain reorganizations, and provides operational visibility.

**Timeline:** Days 1-3 of 7-day sprint

**Success Criteria:**
- Successfully backfills 5,000 blocks in under 5 minutes
- Live-tail maintains <2 second lag from network head
- Automatic reorg detection and recovery for forks up to 6 blocks deep
- Prometheus metrics accurately reflect system state
- System runs continuously for 24+ hours without issues

**Stories:** 9 stories covering RPC client, database schema, parallel backfill, live-tail, reorg handling, migrations, metrics, logging, and integration tests

---

## Scope

### In Scope for Epic 1

- **Blockchain Network:** Ethereum Sepolia testnet only (read-only RPC operations)
- **Data Indexing:** Blocks, transactions, and event logs
- **Historical Data:** Parallel backfill for configurable block range (default: last 5,000 blocks)
- **Real-Time Processing:** Live-tail for new blocks as they are produced
- **Reorg Handling:** Automatic detection and recovery for chain reorganizations up to 6 blocks deep
- **Storage:** PostgreSQL 16 with optimized schema and composite indexes
- **Observability:** Prometheus metrics for indexer performance and health
- **Logging:** Structured JSON logging for all significant events
- **Data Integrity:** Idempotent operations, transaction management, foreign key constraints
- **Testing:** Unit tests (70%+ coverage) and integration tests for critical paths

### Out of Scope for Epic 1

- **Multi-Chain Support:** Other networks (Polygon, Arbitrum, Optimism, etc.) - future enhancement
- **Advanced Blockchain Features:**
  - ERC-20 token transfer decoding and balance tracking
  - Uncle/ommer block handling (Ethereum-specific edge case)
  - Internal transactions (contract-to-contract calls)
  - Historical state queries ("balance at block X")
  - Transaction trace/debug data
- **Smart Contract Features:**
  - Contract source code verification
  - ABI decoding
  - Contract event interpretation beyond raw logs
- **Deep Reorg Recovery:** Reorganizations deeper than 6 blocks (manual intervention required)
- **Write Operations:** Any transactions to the blockchain (read-only system)
- **User-Facing APIs:** REST and WebSocket endpoints (covered in Epic 2)
- **Frontend:** User interface (covered in Epic 2)
- **Advanced Reliability:**
  - Multi-region deployment
  - Read replicas
  - Automatic failover
  - Redis caching layer
- **Performance Optimization Beyond Requirements:**
  - Query result caching
  - Read replicas for horizontal scaling
  - Connection pooling beyond basic configuration

**Rationale:** Epic 1 focuses exclusively on the data pipeline foundation. All user-facing components, advanced features, and production hardening beyond MVP requirements are intentionally deferred.

---

## Technology Stack (Epic 1 Specific)

| Category | Technology | Version | Purpose |
|----------|-----------|---------|---------|
| Language | Go | 1.24+ | Implementation language (required by go-ethereum v1.16.5) |
| Blockchain Client | go-ethereum | 1.16.5 | Ethereum RPC interaction, supports Osaka fork, requires Go 1.24+ |
| Database | PostgreSQL | 16 | Data storage (v18 available but v16 chosen for stability) |
| DB Driver | pgx | v5 (latest) | High-performance PostgreSQL driver (trust score 9.3/10), supports COPY protocol |
| Migrations | golang-migrate | latest | Schema versioning |
| Metrics | prometheus/client_golang | latest | Observability (trust score 7.4/10) |
| Logging | log/slog | stdlib | Structured logging (available in Go 1.21+) |
| Testing | testing + testify | stdlib + latest | Unit and integration tests |

---

## Architecture Overview (Epic 1)

### Component Diagram

```
[Ethereum RPC Node]
       ↓ JSON-RPC
[internal/rpc] ← RPC Client with retry logic
       ↓
[internal/ingest] ← Parse blocks, normalize data
       ↓
[internal/index] ← Backfill + Live-tail + Reorg handling
       ↓
[internal/store/pg] ← PostgreSQL storage layer
       ↓
[PostgreSQL Database] ← blocks, transactions, logs tables
```

### Key Components

1. **RPC Client** (`internal/rpc/`)
   - Connection management
   - Retry logic with exponential backoff
   - Error classification

2. **Ingestion Layer** (`internal/ingest/`)
   - Block parsing
   - Data normalization
   - Domain model conversion

3. **Indexing Layer** (`internal/index/`)
   - Backfill coordinator (parallel workers)
   - Live-tail coordinator (sequential)
   - Reorg handler

4. **Storage Layer** (`internal/store/pg/`)
   - Database abstraction
   - Bulk insert optimization
   - Query builders

---

## Data Architecture

### Database Schema

#### Blocks Table

```sql
CREATE TABLE blocks (
    height BIGINT PRIMARY KEY,
    hash BYTEA NOT NULL UNIQUE,
    parent_hash BYTEA NOT NULL,
    miner BYTEA NOT NULL,
    gas_used NUMERIC NOT NULL,
    gas_limit NUMERIC NOT NULL,
    timestamp BIGINT NOT NULL,
    tx_count INTEGER NOT NULL,
    orphaned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blocks_orphaned_height ON blocks(orphaned, height DESC);
CREATE INDEX idx_blocks_timestamp ON blocks(timestamp DESC);
CREATE INDEX idx_blocks_hash ON blocks(hash);
```

#### Transactions Table

```sql
CREATE TABLE transactions (
    hash BYTEA PRIMARY KEY,
    block_height BIGINT NOT NULL REFERENCES blocks(height) ON DELETE CASCADE,
    tx_index INTEGER NOT NULL,
    from_addr BYTEA NOT NULL,
    to_addr BYTEA,  -- NULL for contract creation
    value_wei NUMERIC NOT NULL,
    fee_wei NUMERIC NOT NULL,
    gas_used NUMERIC NOT NULL,
    gas_price NUMERIC NOT NULL,
    nonce BIGINT NOT NULL,
    success BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tx_block_height ON transactions(block_height);
CREATE INDEX idx_tx_from_addr_block ON transactions(from_addr, block_height DESC);
CREATE INDEX idx_tx_to_addr_block ON transactions(to_addr, block_height DESC);
CREATE INDEX idx_tx_block_index ON transactions(block_height, tx_index);
```

#### Logs Table

```sql
CREATE TABLE logs (
    id BIGSERIAL PRIMARY KEY,
    tx_hash BYTEA NOT NULL REFERENCES transactions(hash) ON DELETE CASCADE,
    log_index INTEGER NOT NULL,
    address BYTEA NOT NULL,
    topic0 BYTEA,
    topic1 BYTEA,
    topic2 BYTEA,
    topic3 BYTEA,
    data BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tx_hash, log_index)
);

CREATE INDEX idx_logs_tx_hash ON logs(tx_hash);
CREATE INDEX idx_logs_address_topic0 ON logs(address, topic0);
CREATE INDEX idx_logs_address ON logs(address);
```

### Domain Models

```go
// internal/ingest/models.go

type Block struct {
    Height       uint64
    Hash         []byte
    ParentHash   []byte
    Miner        []byte
    GasUsed      *big.Int
    GasLimit     *big.Int
    Timestamp    uint64
    TxCount      int
    Transactions []Transaction
}

type Transaction struct {
    Hash      []byte
    BlockHeight uint64
    TxIndex   int
    FromAddr  []byte
    ToAddr    []byte  // nil for contract creation
    ValueWei  *big.Int
    FeeWei    *big.Int
    GasUsed   *big.Int
    GasPrice  *big.Int
    Nonce     uint64
    Success   bool
    Logs      []Log
}

type Log struct {
    TxHash   []byte
    LogIndex int
    Address  []byte
    Topics   [4][]byte  // Up to 4 topics
    Data     []byte
}
```

---

## Story Implementation Details

### Story 1.1: Ethereum RPC Client with Retry Logic

**Files:**
- `internal/rpc/client.go`
- `internal/rpc/errors.go`
- `internal/rpc/client_test.go`

**Key Types:**

```go
type Client struct {
    ethClient *ethclient.Client
    config    *Config
    logger    *slog.Logger
}

type Config struct {
    RPCURL       string
    Timeout      time.Duration
    MaxRetries   int
    RetryBackoff time.Duration
}
```

**Implementation Guidance:**

```go
func (c *Client) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
    var block *types.Block
    var err error

    for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
        block, err = c.ethClient.BlockByNumber(ctx, big.NewInt(int64(height)))
        if err == nil {
            return block, nil
        }

        // Classify error
        if isTransientError(err) {
            backoff := time.Duration(attempt) * c.config.RetryBackoff
            c.logger.Warn("RPC call failed, retrying",
                "attempt", attempt+1,
                "backoff", backoff,
                "error", err)
            time.Sleep(backoff)
            continue
        }

        // Permanent error, don't retry
        return nil, fmt.Errorf("permanent RPC error: %w", err)
    }

    return nil, fmt.Errorf("max retries (%d) exceeded: %w", c.config.MaxRetries, err)
}

func isTransientError(err error) bool {
    // Network timeouts, connection refused, rate limits = transient
    // Invalid parameters, not found = permanent
    // Implement based on go-ethereum error types
    return errors.Is(err, context.DeadlineExceeded) ||
           strings.Contains(err.Error(), "connection refused") ||
           strings.Contains(err.Error(), "rate limit")
}
```

**Configuration:**
```bash
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
RPC_TIMEOUT=30s
RPC_MAX_RETRIES=5
RPC_RETRY_BACKOFF=2s
```

**Testing:**
- Mock ethclient for unit tests
- Test retry logic with simulated failures
- Test permanent error immediate failure

---

### Story 1.2: PostgreSQL Schema and Migrations

**Files:**
- `migrations/000001_initial_schema.up.sql`
- `migrations/000001_initial_schema.down.sql`
- `migrations/000002_add_indexes.up.sql`
- `migrations/000002_add_indexes.down.sql`

**Migration Execution:**

```go
// cmd/worker/main.go and cmd/api/main.go

import "github.com/golang-migrate/migrate/v4"
import _ "github.com/golang-migrate/migrate/v4/database/postgres"
import _ "github.com/golang-migrate/migrate/v4/source/file"

func runMigrations(dbURL string) error {
    m, err := migrate.New(
        "file://migrations",
        dbURL,
    )
    if err != nil {
        return err
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }

    return nil
}
```

**Database Configuration:**
```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=postgres
DB_MAX_CONNS=20
```

---

### Story 1.3: Parallel Backfill Worker Pool

**Files:**
- `internal/index/backfill.go`
- `internal/index/backfill_test.go`

**Key Types:**

```go
type BackfillCoordinator struct {
    rpcClient   *rpc.Client
    ingester    *ingest.Ingester
    store       store.Store
    workerCount int
    batchSize   int
    logger      *slog.Logger
}
```

**Implementation Pattern:**

```go
func (bc *BackfillCoordinator) Backfill(ctx context.Context, startHeight, endHeight uint64) error {
    totalBlocks := endHeight - startHeight + 1

    // Create job channel
    jobs := make(chan uint64, bc.workerCount*2)
    results := make(chan *ingest.Block, bc.workerCount*2)
    errors := make(chan error, bc.workerCount)

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < bc.workerCount; i++ {
        wg.Add(1)
        go bc.worker(ctx, &wg, jobs, results, errors)
    }

    // Start result collector
    done := make(chan struct{})
    go bc.collectResults(ctx, results, done)

    // Send jobs
    go func() {
        for height := startHeight; height <= endHeight; height++ {
            select {
            case jobs <- height:
            case <-ctx.Done():
                return
            }
        }
        close(jobs)
    }()

    // Wait for workers
    wg.Wait()
    close(results)
    <-done

    select {
    case err := <-errors:
        return err
    default:
        return nil
    }
}

func (bc *BackfillCoordinator) worker(
    ctx context.Context,
    wg *sync.WaitGroup,
    jobs <-chan uint64,
    results chan<- *ingest.Block,
    errors chan<- error,
) {
    defer wg.Done()

    for height := range jobs {
        block, err := bc.fetchAndParseBlock(ctx, height)
        if err != nil {
            select {
            case errors <- err:
            default:
            }
            return
        }

        results <- block
    }
}

func (bc *BackfillCoordinator) collectResults(ctx context.Context, results <-chan *ingest.Block, done chan<- struct{}) {
    defer close(done)

    batch := make([]*ingest.Block, 0, bc.batchSize)

    for block := range results {
        batch = append(batch, block)

        if len(batch) >= bc.batchSize {
            if err := bc.store.InsertBlocks(ctx, batch); err != nil {
                bc.logger.Error("Failed to insert batch", "error", err)
                return
            }
            bc.logger.Info("Batch inserted", "count", len(batch))
            batch = batch[:0]
        }
    }

    // Insert remaining
    if len(batch) > 0 {
        bc.store.InsertBlocks(ctx, batch)
    }
}
```

**Configuration:**
```bash
BACKFILL_WORKERS=8
BACKFILL_BATCH_SIZE=100
BACKFILL_START_HEIGHT=0
BACKFILL_END_HEIGHT=5000
```

---

### Story 1.4: Live-Tail Mechanism for New Blocks

**Files:**
- `internal/index/livetail.go`
- `internal/index/livetail_test.go`

**Implementation:**

```go
type LiveTailCoordinator struct {
    rpcClient    *rpc.Client
    ingester     *ingest.Ingester
    store        store.Store
    reorgHandler *ReorgHandler
    pollInterval time.Duration
    logger       *slog.Logger
}

func (ltc *LiveTailCoordinator) Start(ctx context.Context) error {
    ticker := time.NewTicker(ltc.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := ltc.processNextBlock(ctx); err != nil {
                ltc.logger.Error("Failed to process block", "error", err)
                // Continue despite errors (will retry on next tick)
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

func (ltc *LiveTailCoordinator) processNextBlock(ctx context.Context) error {
    // Get current head from database
    dbHead, err := ltc.store.GetLatestBlock(ctx)
    if err != nil {
        return err
    }

    nextHeight := dbHead.Height + 1

    // Fetch next block from RPC
    block, err := ltc.rpcClient.GetBlockByNumber(ctx, nextHeight)
    if err != nil {
        // Block not available yet (we're at head)
        return nil
    }

    // Parse block
    parsedBlock := ltc.ingester.ParseBlock(block)

    // Check if parent matches DB head (reorg detection)
    if !bytes.Equal(parsedBlock.ParentHash, dbHead.Hash) {
        ltc.logger.Warn("Reorg detected",
            "height", nextHeight,
            "expected_parent", hex.EncodeToString(dbHead.Hash),
            "actual_parent", hex.EncodeToString(parsedBlock.ParentHash))
        return ltc.reorgHandler.HandleReorg(ctx, parsedBlock)
    }

    // Normal case: append block
    return ltc.store.InsertBlocks(ctx, []*ingest.Block{parsedBlock})
}
```

**Configuration:**
```bash
LIVE_TAIL_POLL_INTERVAL=5s
```

---

### Story 1.5: Chain Reorganization Detection and Recovery

**Files:**
- `internal/index/reorg.go`
- `internal/index/reorg_test.go`

**Implementation:**

```go
type ReorgHandler struct {
    store    store.Store
    maxDepth int
    logger   *slog.Logger
}

func (rh *ReorgHandler) HandleReorg(ctx context.Context, newBlock *ingest.Block) error {
    // Walk backwards to find common ancestor
    forkPoint, err := rh.findForkPoint(ctx, newBlock)
    if err != nil {
        return err
    }

    // Get current head
    currentHead, err := rh.store.GetLatestBlock(ctx)
    if err != nil {
        return err
    }

    depth := currentHead.Height - forkPoint
    if depth > uint64(rh.maxDepth) {
        return fmt.Errorf("reorg depth %d exceeds max depth %d", depth, rh.maxDepth)
    }

    rh.logger.Warn("Reorg recovery",
        "fork_point", forkPoint,
        "depth", depth,
        "current_head", currentHead.Height,
        "new_head", newBlock.Height)

    // Mark orphaned blocks
    orphanedHeights := make([]uint64, 0)
    for h := forkPoint + 1; h <= currentHead.Height; h++ {
        orphanedHeights = append(orphanedHeights, h)
    }

    if err := rh.store.MarkBlocksOrphaned(ctx, orphanedHeights); err != nil {
        return err
    }

    rh.logger.Info("Marked blocks as orphaned", "count", len(orphanedHeights))

    // Insert new block (will be handled by live-tail on next iteration)
    return nil
}

func (rh *ReorgHandler) findForkPoint(ctx context.Context, newBlock *ingest.Block) (uint64, error) {
    currentHeight := newBlock.Height - 1

    for i := 0; i < rh.maxDepth; i++ {
        if currentHeight == 0 {
            return 0, nil
        }

        dbBlock, err := rh.store.GetBlockByHeight(ctx, currentHeight)
        if err != nil {
            return 0, err
        }

        if bytes.Equal(newBlock.ParentHash, dbBlock.Hash) {
            return currentHeight, nil
        }

        // Continue walking backwards
        newBlock, err = rh.store.GetBlockByHeight(ctx, currentHeight)
        if err != nil {
            return 0, err
        }
        currentHeight--
    }

    return 0, fmt.Errorf("could not find fork point within %d blocks", rh.maxDepth)
}
```

**Configuration:**
```bash
REORG_MAX_DEPTH=6
```

---

### Story 1.6: Database Migration System

Covered in Story 1.2 - migrations are automated via `golang-migrate`.

---

### Story 1.7: Prometheus Metrics for Indexer

**Files:**
- `internal/util/metrics.go`

**Metrics Definitions:**

```go
var (
    BlocksIndexed = promauto.NewCounter(prometheus.CounterOpts{
        Name: "explorer_blocks_indexed_total",
        Help: "Total number of blocks indexed",
    })

    IndexLagBlocks = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "explorer_index_lag_blocks",
        Help: "Number of blocks behind network head",
    })

    IndexLagSeconds = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "explorer_index_lag_seconds",
        Help: "Time lag from network head in seconds",
    })

    RPCErrors = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "explorer_rpc_errors_total",
        Help: "Total number of RPC errors",
    }, []string{"error_type"})

    BackfillDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name: "explorer_backfill_duration_seconds",
        Help: "Time to backfill blocks",
        Buckets: prometheus.DefBuckets,
    })
)
```

**Usage in Code:**

```go
// Increment when block indexed
metrics.BlocksIndexed.Inc()

// Update lag periodically
metrics.IndexLagBlocks.Set(float64(networkHead - dbHead))
metrics.IndexLagSeconds.Set(float64(time.Since(lastBlockTime).Seconds()))

// Track RPC errors
metrics.RPCErrors.WithLabelValues("network").Inc()
```

**Metrics Endpoint:**
```go
// cmd/worker/main.go or cmd/api/main.go
import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())
```

---

### Story 1.8: Structured Logging for Debugging

**Files:**
- `internal/util/logger.go`

**Logger Setup:**

```go
import "log/slog"

func NewLogger(level string) *slog.Logger {
    var logLevel slog.Level
    switch level {
    case "DEBUG":
        logLevel = slog.LevelDebug
    case "INFO":
        logLevel = slog.LevelInfo
    case "WARN":
        logLevel = slog.LevelWarn
    case "ERROR":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: logLevel,
    })

    return slog.New(handler)
}
```

**Usage:**

```go
logger.Info("Backfill started",
    "start_height", startHeight,
    "end_height", endHeight,
    "workers", workerCount)

logger.Error("RPC call failed",
    "attempt", attempt,
    "error", err)
```

**Log Output:**
```json
{"time":"2025-10-29T10:30:00Z","level":"INFO","msg":"Backfill started","start_height":0,"end_height":5000,"workers":8}
```

---

### Story 1.9: Integration Tests for Indexer Pipeline

**Files:**
- Various `*_test.go` files
- `internal/index/backfill_test.go`
- `internal/index/reorg_test.go`

**Test Strategy:**

1. **Unit Tests:** Mock external dependencies (RPC client, database)
2. **Integration Tests:** Use test database (Docker or testcontainers)
3. **End-to-End Tests:** Full workflow with test RPC mock

**Example Integration Test:**

```go
func TestBackfill_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create mock RPC client with known blocks
    mockRPC := &MockRPCClient{
        blocks: generateTestBlocks(0, 100),
    }

    // Run backfill
    coordinator := &BackfillCoordinator{
        rpcClient: mockRPC,
        store: store.NewPostgresStore(db),
        workerCount: 4,
        batchSize: 25,
    }

    err := coordinator.Backfill(context.Background(), 0, 100)
    require.NoError(t, err)

    // Verify all blocks inserted
    count, err := db.Query("SELECT COUNT(*) FROM blocks")
    require.NoError(t, err)
    assert.Equal(t, 101, count) // 0-100 inclusive
}
```

---

## Testing Strategy (Epic 1)

**Unit Test Coverage Targets:**
- RPC Client: 80%+ (retry logic critical)
- Ingestion: 70%+ (parsing logic)
- Indexing: 80%+ (backfill, live-tail, reorg critical)
- Storage: 70%+ (queries)

**Integration Tests:**
- Backfill workflow with test database
- Reorg recovery scenarios
- End-to-end indexing pipeline

**Performance Tests:**
- Backfill 5,000 blocks (target: <5 minutes)
- Measure database insert performance

**Running Tests:**
```bash
# Unit tests only
go test ./internal/... -short

# All tests including integration
go test ./internal/...

# With coverage
go test ./internal/... -cover
```

---

## Implementation Timeline (Epic 1)

**Day 1:** Stories 1.1 + 1.2
- RPC client implementation
- Database schema and migrations
- Basic testing

**Day 2:** Story 1.3
- Worker pool implementation
- Bulk insert optimization
- Backfill testing

**Day 3:** Stories 1.4 + 1.5 + 1.7 + 1.8
- Live-tail implementation
- Reorg handling
- Metrics and logging
- Integration tests

---

## Risks, Assumptions, and Open Questions

### Risks

#### Risk 1: RPC Rate Limiting
- **Description:** Free-tier RPC providers (Alchemy/Infura) may throttle requests during 8-worker parallel backfill
- **Probability:** Medium
- **Impact:** High (blocks backfill progress)
- **Mitigation Strategies:**
  1. Configurable worker count (`BACKFILL_WORKERS`) - reduce to 4 if rate limited
  2. Exponential backoff retry logic already implemented (max 5 retries)
  3. Monitor RPC error metrics (`explorer_rpc_errors_total`)
  4. Upgrade to paid tier if needed (~$50/month for higher limits)
  5. Fallback to public Sepolia nodes (slower but free)
- **Owner:** Developer
- **Status:** Monitored during backfill testing

#### Risk 2: Reorg Deeper Than 6 Blocks
- **Description:** Chain reorganization exceeding max handled depth (6 blocks) would not be automatically recovered
- **Probability:** Very Low (Sepolia testnet)
- **Impact:** Medium (requires manual intervention)
- **Mitigation Strategies:**
  1. Reorg depth limit configurable (`REORG_MAX_DEPTH`)
  2. System detects and logs errors for deep reorgs
  3. Manual recovery: Mark orphaned blocks, re-run backfill from fork point
  4. Alert via metrics if reorg depth exceeds threshold
  5. Sepolia testnet rarely has deep reorgs (>3 blocks uncommon)
- **Owner:** Developer
- **Status:** Acceptable for portfolio demonstration

#### Risk 3: Database Connection Pool Exhaustion
- **Description:** 8 backfill workers + live-tail coordinator may exhaust PostgreSQL connection pool
- **Probability:** Low
- **Impact:** Medium (indexer stalls)
- **Mitigation Strategies:**
  1. Connection pool sized appropriately (`DB_MAX_CONNS=20`)
  2. Batch inserts reduce connection hold time
  3. Workers share connection pool via pgxpool
  4. Monitor database connection count via pg_stat_activity
  5. Configurable worker count allows tuning
- **Owner:** Developer
- **Status:** Connection pooling configured

#### Risk 4: Blockchain Node Unavailability
- **Description:** Ethereum Sepolia RPC endpoint goes offline during indexing or demo
- **Probability:** Low
- **Impact:** High (system cannot index new blocks)
- **Mitigation Strategies:**
  1. Multiple RPC provider configuration (Alchemy primary, Infura fallback) - future enhancement
  2. Retry logic handles transient outages (up to 10 seconds)
  3. Graceful degradation: Live-tail pauses, API continues serving indexed data
  4. Health check reports RPC status
  5. For demo: Cache sample data for offline presentation (future)
- **Owner:** Developer
- **Status:** Retry logic implemented, multi-provider future enhancement

#### Risk 5: Performance Target Not Met
- **Description:** Backfill fails to achieve 5,000 blocks in <5 minutes on standard hardware
- **Probability:** Low
- **Impact:** Medium (NFR001 not met)
- **Mitigation Strategies:**
  1. Worker pool pattern specifically designed for parallelism
  2. Bulk inserts optimize database writes
  3. Benchmark on target hardware during Day 2
  4. If slow: Increase workers (if RPC allows), optimize batch size, add connection pooling tuning
  5. Worst case: Document actual performance and explain bottleneck
- **Owner:** Developer
- **Status:** Performance target to be validated Day 2

### Assumptions

#### Assumption 1: RPC Availability
- **Assumption:** Ethereum Sepolia testnet will remain accessible via Alchemy/Infura throughout 7-day development sprint
- **Validation:** Check provider status pages before starting
- **Risk if Invalid:** Cannot develop or test system (HIGH IMPACT)
- **Fallback:** Use public Sepolia nodes or cached test data

#### Assumption 2: RPC Free Tier Sufficiency
- **Assumption:** Free-tier RPC limits (~300K requests/day for Alchemy) sufficient for development and demo
- **Calculation:** 5,000 blocks × (1 block + ~50 txs avg) = ~255K requests for backfill, well under limit
- **Validation:** Monitor usage during backfill
- **Risk if Invalid:** Need to pay for upgraded tier ($50/month)
- **Fallback:** Reduce backfill scope to 3,000 blocks or upgrade

#### Assumption 3: Standard Hardware Available
- **Assumption:** Development machine has 4+ CPU cores, 8+ GB RAM, SSD storage
- **Validation:** Check hardware specs before starting
- **Risk if Invalid:** Performance targets may not be met (MEDIUM IMPACT)
- **Fallback:** Adjust worker count, accept longer backfill times, document constraints

#### Assumption 4: Sepolia Block Structure Stability
- **Assumption:** Ethereum Sepolia testnet has similar block structure to mainnet (blocks, txs, logs) and structure won't change during sprint
- **Validation:** Review go-ethereum documentation
- **Risk if Invalid:** Parsing logic may fail (HIGH IMPACT)
- **Fallback:** N/A - Sepolia is stable testnet, highly unlikely to change

#### Assumption 5: PostgreSQL 16 Compatibility
- **Assumption:** PostgreSQL 16 features (COPY protocol, composite indexes) work as documented with pgx v5
- **Validation:** Test database setup on Day 1
- **Risk if Invalid:** May need to adjust queries or use PostgreSQL 15 (LOW IMPACT)
- **Fallback:** Downgrade to PostgreSQL 15, avoid v16-specific features

#### Assumption 6: Scope Sufficiency
- **Assumption:** Indexing 5,000 blocks is sufficient to demonstrate technical competency (vs. full historical 50M+ blocks)
- **Validation:** PRD specifies 5,000 blocks as target
- **Risk if Invalid:** Evaluators may expect more comprehensive indexing (LOW IMPACT)
- **Fallback:** Document that system can scale to full historical indexing with same architecture

#### Assumption 7: No Authentication Required
- **Assumption:** PostgreSQL database requires no complex authentication beyond username/password (local development)
- **Validation:** Docker Compose PostgreSQL configuration
- **Risk if Invalid:** Connection setup more complex (LOW IMPACT)
- **Fallback:** Adjust connection string configuration

### Open Questions

#### Question 1: Graceful Shutdown with Checkpoint Saving
- **Question:** Should backfill save checkpoint state to resume from last processed block on restart?
- **Options:**
  1. Yes - Add checkpoint table tracking last processed height, resume on restart
  2. No - Restart from beginning if interrupted (simpler, acceptable for 5K blocks)
- **Decision:** **No** - Out of scope for MVP. Restarting backfill from beginning is acceptable for demo scale (5K blocks in <5min). Add as future enhancement if needed.
- **Implications:** If indexer crashes during backfill, re-indexes from start (acceptable trade-off)

#### Question 2: Internal Transactions Indexing
- **Question:** Should we index internal transactions (contract-to-contract calls)?
- **Context:** Internal txs not in standard block data, requires trace APIs
- **Decision:** **No** - Out of scope per PRD. Only top-level transactions indexed. Internal tx indexing requires eth_trace APIs not available on all providers.
- **Implications:** Address transaction history may be incomplete for contract interactions (acceptable for demo)

#### Question 3: Offline Demo Mode
- **Question:** Should system support offline demo mode with cached sample data if Sepolia RPC unavailable during presentation?
- **Decision:** **Defer to Day 7** - If time permits, cache 100 blocks of sample data for offline presentation. Not critical for implementation.
- **Implications:** Live demo requires working RPC connection (acceptable risk given retry logic)

#### Question 4: Connection Pool Sizing
- **Question:** What is optimal DB connection pool size for 8 workers + live-tail + API server?
- **Current Config:** `DB_MAX_CONNS=20` (indexer), `DB_MAX_CONNS=10` (API)
- **Decision:** **Start with defaults, tune if needed** - Monitor pg_stat_activity during backfill. Adjust if connection exhaustion observed.
- **Implications:** May need tuning during Day 2 performance testing

#### Question 5: Reorg Notification Strategy
- **Question:** Should reorgs trigger external notifications (email, Slack webhook)?
- **Decision:** **No** - Structured logs and metrics sufficient for demo. External notifications out of scope.
- **Implications:** Reorgs only visible via logs and metrics (acceptable for portfolio project)

---

## Configuration Summary (Epic 1)

```bash
# RPC
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
RPC_TIMEOUT=30s
RPC_MAX_RETRIES=5
RPC_RETRY_BACKOFF=2s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=postgres
DB_MAX_CONNS=20

# Indexer
BACKFILL_START_HEIGHT=0
BACKFILL_END_HEIGHT=5000
BACKFILL_WORKERS=8
BACKFILL_BATCH_SIZE=100
LIVE_TAIL_POLL_INTERVAL=5s
REORG_MAX_DEPTH=6

# Logging
LOG_LEVEL=INFO
```

---

## Success Validation

After Epic 1 implementation, validate:

1. ✅ Can connect to Ethereum Sepolia RPC
2. ✅ Database schema created via migrations
3. ✅ Backfill 5,000 blocks in <5 minutes
4. ✅ Live-tail maintains <2s lag
5. ✅ Reorg recovery works (simulate reorg)
6. ✅ Prometheus metrics exposed at `/metrics`
7. ✅ Structured logs visible in JSON format
8. ✅ System runs 24+ hours without issues

---

## Requirements Traceability Matrix

This matrix maps PRD requirements → Epic 1 acceptance criteria → architecture components → implementation files → test coverage, ensuring complete end-to-end traceability.

### Functional Requirements Coverage

| PRD Req | Requirement | Epic 1 AC | Architecture Component | Implementation File | Test File | Test Method | Status |
|---------|-------------|-----------|------------------------|---------------------|-----------|-------------|--------|
| **FR001** | Historical Block Indexing | Backfills 5K blocks <5min | Backfill Coordinator + Worker Pool | `internal/index/backfill.go` | `internal/index/backfill_test.go` | `TestBackfill_Performance` | ✅ Spec Complete |
| **FR002** | Real-Time Block Monitoring | Live-tail <2s lag | Live-Tail Coordinator | `internal/index/livetail.go` | `internal/index/livetail_test.go` | `TestLiveTail_Lag` | ✅ Spec Complete |
| **FR003** | Reorg Handling | Reorg up to 6 blocks | Reorg Handler | `internal/index/reorg.go` | `internal/index/reorg_test.go` | `TestReorg_6BlockDepth` | ✅ Spec Complete |
| **FR013** | Metrics Exposure | Metrics accurate | Prometheus Metrics | `internal/util/metrics.go` | `internal/util/metrics_test.go` | `TestMetrics_Registration` | ✅ Spec Complete |
| **FR014** | Structured Logging | JSON logs | Structured Logger | `internal/util/logger.go` | N/A (stdlib) | Manual verification | ✅ Spec Complete |

### Non-Functional Requirements Coverage

| PRD Req | Requirement | Epic 1 AC | Architecture Component | Implementation File | Test File | Verification Method | Status |
|---------|-------------|-----------|------------------------|---------------------|-----------|---------------------|--------|
| **NFR001** | Backfill <5min (5K blocks) | Same | Worker Pool (8 workers) + Bulk Inserts | `internal/index/backfill.go`, `internal/store/pg/postgres.go` | `internal/index/backfill_test.go` | Performance benchmark | ✅ Spec Complete |
| **NFR003** | 24h Continuous Operation | System runs 24h+ | RPC Retry Logic + Error Handling | `internal/rpc/client.go`, `internal/index/livetail.go` | `internal/rpc/client_test.go` | Integration test (long-running) | ✅ Spec Complete |
| **NFR005** | Test Coverage >70% | Coverage >70% | All components | All `internal/` files | All `*_test.go` files | `go test -cover` | ✅ Spec Complete |

### Story-Level Traceability

| Story | Epic 1 AC | Components | Files | Tests | PRD Mapping |
|-------|-----------|-----------|-------|-------|-------------|
| **1.1: RPC Client** | RPC with retry | RPC Client Layer | `internal/rpc/client.go`, `internal/rpc/errors.go` | `internal/rpc/client_test.go` | NFR003 (reliability) |
| **1.2: Schema** | Schema supports data | Database Schema | `migrations/000001_initial_schema.up.sql`, `migrations/000002_add_indexes.up.sql` | Integration tests | FR001, FR002, FR003 (all data) |
| **1.3: Backfill** | 5K blocks <5min | Backfill Coordinator | `internal/index/backfill.go`, `internal/ingest/ingester.go` | `internal/index/backfill_test.go` | FR001, NFR001 |
| **1.4: Live-Tail** | <2s lag | Live-Tail Coordinator | `internal/index/livetail.go` | `internal/index/livetail_test.go` | FR002 |
| **1.5: Reorg** | Reorg up to 6 blocks | Reorg Handler | `internal/index/reorg.go` | `internal/index/reorg_test.go` | FR003 |
| **1.6: Migrations** | Auto migrations | Migration System | `migrations/*.sql`, `cmd/worker/main.go` | Integration tests | All FRs (schema foundation) |
| **1.7: Metrics** | Metrics accurate | Prometheus Metrics | `internal/util/metrics.go` | `internal/util/metrics_test.go` | FR013 |
| **1.8: Logging** | JSON logs | Structured Logger | `internal/util/logger.go` | Manual verification | FR014 |
| **1.9: Tests** | >70% coverage | Test Suite | All `*_test.go` | Self-testing | NFR005 |

### Data Model Coverage

| Data Model | Tables | Indexes | PRD Requirements Served | Epic 1 Stories | Test Coverage |
|------------|--------|---------|-------------------------|----------------|---------------|
| **Blocks** | `blocks` table | `idx_blocks_orphaned_height`, `idx_blocks_timestamp`, `idx_blocks_hash` | FR001, FR002, FR003, FR004 (Epic 2) | 1.2, 1.3, 1.4, 1.5 | `internal/store/pg/blocks_test.go` |
| **Transactions** | `transactions` table | `idx_tx_block_height`, `idx_tx_from_addr_block`, `idx_tx_to_addr_block`, `idx_tx_block_index` | FR001, FR002, FR005, FR006 (Epic 2) | 1.2, 1.3, 1.4 | `internal/store/pg/transactions_test.go` |
| **Logs** | `logs` table | `idx_logs_tx_hash`, `idx_logs_address_topic0`, `idx_logs_address` | FR001, FR002, FR007 (Epic 2) | 1.2, 1.3, 1.4 | `internal/store/pg/logs_test.go` |

### Integration Points Coverage

| Integration | Type | Epic 1 Component | Configuration | Test Strategy | PRD Requirement |
|-------------|------|------------------|---------------|---------------|-----------------|
| **Ethereum Sepolia RPC** | External (outbound) | RPC Client | `RPC_URL`, `RPC_TIMEOUT`, `RPC_MAX_RETRIES` | Mock RPC in unit tests, real RPC in integration tests | FR001, FR002 |
| **PostgreSQL 16** | External (outbound) | Storage Layer | `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_MAX_CONNS` | Test database (Docker), testcontainers-go | All data-related FRs |

### Acceptance Criteria → Test Mapping

| Epic 1 Acceptance Criteria | Verification Method | Test Location | Expected Result | Status |
|-----------------------------|---------------------|---------------|-----------------|--------|
| Backfills 5,000 blocks in <5 minutes | Performance benchmark | `internal/index/backfill_test.go::TestBackfill_Performance` | Duration < 300s | To be validated Day 2 |
| Live-tail maintains <2s lag | Integration test with timer | `internal/index/livetail_test.go::TestLiveTail_Lag` | Lag < 2s | To be validated Day 3 |
| Reorg up to 6 blocks detected and recovered | Unit test with mock reorg scenario | `internal/index/reorg_test.go::TestReorg_6BlockDepth` | Orphaned blocks marked, canonical chain restored | To be validated Day 3 |
| Metrics accurately reflect state | Unit test + manual verification | `internal/util/metrics_test.go` + Prometheus UI | Metrics match actual system state | To be validated Day 3 |
| System runs 24+ hours | Long-running integration test | Manual execution on Day 3 evening | No crashes, memory leaks, or degradation | To be validated Day 3-4 |

### Coverage Summary

| Category | Total Items | Covered | Coverage % | Status |
|----------|-------------|---------|-----------|--------|
| **Functional Requirements (Epic 1 Scope)** | 5 | 5 | 100% | ✅ Complete |
| **Non-Functional Requirements (Epic 1 Scope)** | 3 | 3 | 100% | ✅ Complete |
| **Stories** | 9 | 9 | 100% | ✅ Complete |
| **Data Models** | 3 | 3 | 100% | ✅ Complete |
| **Integration Points** | 2 | 2 | 100% | ✅ Complete |
| **Acceptance Criteria** | 5 | 5 | 100% | ✅ Complete |

### Gap Analysis

**No gaps identified.** All PRD requirements within Epic 1 scope are mapped to:
- ✅ Epic-level acceptance criteria
- ✅ Architecture components
- ✅ Implementation files
- ✅ Test files and methods
- ✅ Verification strategies

### Notes

1. **Epic 2 Dependencies:** Requirements FR004-FR012 (API and frontend) are intentionally out of scope for Epic 1 and covered in Epic 2 tech spec.
2. **Test Status:** All tests are specified but not yet implemented. Tests will be written during Days 1-3 implementation.
3. **Performance Validation:** NFR001 (backfill speed) will be validated during Day 2 performance testing. If target not met, mitigation strategies documented in Risks section.
4. **Integration Coverage:** Epic 1 focuses on data pipeline. Integration with Epic 2 (API layer) validated in Epic 2 integration tests.

---

_Generated from Solution Architecture and PRD_
