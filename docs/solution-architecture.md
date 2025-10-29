# Blockchain Explorer - Solution Architecture Document

**Project:** Blockchain Explorer
**Date:** 2025-10-29
**Last Updated:** 2025-10-29 (Tech Stack Updated)
**Author:** Hieu

## Executive Summary

This document defines the solution architecture for the Blockchain Explorer, a production-grade Ethereum blockchain indexer and query platform built in Go. The system implements a complete data pipeline from blockchain nodes to end-users within a 7-day development sprint.

**Architecture Style:** Modular monolith with separate processes (indexer worker + API server)
**Repository Strategy:** Monorepo
**Primary Language:** Go 1.24+ (required by go-ethereum v1.16.5)
**Target Platform:** Docker Compose (local development/demo)

**Core Components:**
1. **Indexer Worker** - Parallel backfill, live-tail, reorg handling
2. **API Server** - REST + WebSocket endpoints
3. **PostgreSQL Database** - Optimized storage with composite indexes
4. **Minimal SPA Frontend** - Live blocks ticker and search interface

**Key Design Principles:**
- **Layer Separation**: Clear boundaries between RPC, ingestion, indexing, storage, and API layers
- **Scalability**: Worker pool pattern for parallel processing, stateless API design
- **Reliability**: Idempotent operations, automatic reorg recovery, retry logic
- **Observability**: Prometheus metrics, structured JSON logging, health checks
- **Simplicity**: Production-ready patterns without unnecessary complexity

This architecture balances demonstration value (showcasing advanced patterns) with implementation feasibility (achievable in 7 days).

## 1. Technology Stack & Decisions

### 1.1 Technology & Library Decision Table

| Category | Technology | Version | Rationale |
|----------|-----------|---------|-----------|
| **Language** | Go | 1.24+ | Strong concurrency primitives, excellent for systems programming, fast compilation, single binary deployment. **Required by latest go-ethereum (v1.16.5)**. Latest stable: Go 1.25.3 (Oct 2025) or Go 1.24.9 for compatibility. |
| **Database** | PostgreSQL | 16 | ACID guarantees, excellent indexing, wide deployment knowledge, strong JSON support. **Version 16 chosen for production stability** (v18 available but newer). Supported until Nov 2029. |
| **DB Driver** | pgx | v5 (latest) | High-performance native Go driver with **trust score 9.3/10**, connection pooling with pgxpool, native PostgreSQL types, supports COPY protocol for bulk inserts. Use `github.com/jackc/pgx/v5`. |
| **HTTP Router** | chi | v5 (latest) | Lightweight, idiomatic, excellent middleware support, standard library compatible. **Trust score 6.8/10**, actively maintained. Use `github.com/go-chi/chi/v5`. |
| **Blockchain Client** | go-ethereum | 1.16.5 | Official Ethereum Go implementation, **supports Osaka (Fusaka) fork**, default 60M gas limit, **requires Go 1.24+**, comprehensive RPC support. Latest stable release Oct 2025. |
| **Metrics** | prometheus/client_golang | latest | Industry standard for metrics with **trust score 7.4/10**, excellent Go support, official Prometheus client library from `/prometheus/client_golang`. |
| **Logging** | log/slog | stdlib | Structured logging native to Go 1.21+, zero dependencies, performant. Available in Go 1.22+ and later. |
| **Testing** | testing + testify | stdlib + latest | Standard library testing with testify assertions for better readability. Use latest testify version. |
| **Migrations** | golang-migrate | latest | Reliable migration tool, supports PostgreSQL, version tracking, up/down migrations. Use latest stable version. |
| **WebSocket** | gorilla/websocket | latest | Robust WebSocket implementation, production-proven, good error handling. Use latest stable version. |
| **Containerization** | Docker + Docker Compose | 24.0+ / 2.21+ | Standard container runtime, reproducible environments, easy setup |
| **Frontend** | Vanilla HTML/JS | N/A | No build step, minimal complexity, focuses attention on backend capabilities |

**Key Technology Decisions:**

1. **Go 1.24+** (updated from 1.22+): Chosen for excellent concurrency support (goroutines, channels), systems programming capabilities, and fast compilation. **Go 1.24+ is required by go-ethereum v1.16.5**. The worker pool pattern for parallel backfill is natural in Go. log/slog (structured logging) is native in Go 1.21+. Both Go 1.24.9 and Go 1.25.3 are currently supported stable versions (as of October 2025).

2. **PostgreSQL 16**: Selected over time-series databases (ClickHouse, TimescaleDB) for simplicity and broad applicability. **PostgreSQL 18 is available (released Sept 2025) with improved I/O subsystem, but v16 chosen for production stability** with support until November 2029. Blockchain data is append-only and query patterns (block lookup, address history) map well to relational indexing. Composite indexes handle common queries efficiently.

3. **pgx v5**: High-performance pure Go PostgreSQL driver with **trust score 9.3/10** from context7. Provides excellent connection pooling via pgxpool, native PostgreSQL type support, and COPY protocol for efficient bulk inserts during backfill operations.

4. **chi v5 Router**: Lightweight HTTP router (trust score 6.8/10) that's more performant and idiomatic than heavier frameworks like Gin or Echo, while providing necessary middleware support. Stays close to standard library patterns.

5. **go-ethereum v1.16.5** (updated from v1.13.5): Official Ethereum implementation provides reliable RPC client, block parsing, and transaction decoding. **Latest stable version (Oct 2025) supports Osaka (Fusaka) fork** and sets default gas limit to 60M (recommended). Well-documented and actively maintained. **Requires Go 1.24+**.

6. **Separate Processes (indexer + API)**: Allows independent scaling and fault isolation. Indexer can restart without affecting API availability. Communicates via PostgreSQL (no inter-process communication needed).

7. **Vanilla HTML/JS Frontend**: Eliminates build complexity (webpack, npm), keeps focus on backend. Uses native WebSocket API for real-time updates. Sufficient for portfolio demonstration.

### 1.2 Recommended go.mod Configuration

**Initial go.mod with latest stable versions (October 2025):**

```go
module github.com/yourusername/go-blockchain-explorer

go 1.24

require (
    github.com/ethereum/go-ethereum v1.16.5
    github.com/go-chi/chi/v5 v5.1.0
    github.com/jackc/pgx/v5 v5.7.2
    github.com/prometheus/client_golang v1.21.0
    github.com/gorilla/websocket v1.5.3
    github.com/golang-migrate/migrate/v4 v4.18.1
    github.com/stretchr/testify v1.10.0
)
```

**Notes:**
- Go 1.24 is the minimum required version for go-ethereum v1.16.5
- All dependencies use semantic versioning
- Run `go get -u` to update to latest patch versions
- Run `go mod tidy` after adding new imports

## 2. Architecture Overview

### 2.1 System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        External Services                        │
│  ┌──────────────────────┐        ┌─────────────────────┐       │
│  │ Ethereum Sepolia RPC │        │ Prometheus (optional)│       │
│  │ (Alchemy/Infura)     │        │                      │       │
│  └──────────┬───────────┘        └──────────▲──────────┘       │
└─────────────┼────────────────────────────────┼──────────────────┘
              │                                │
              │ JSON-RPC                       │ /metrics
              │ (blocks, txs, logs)            │
              │                                │
┌─────────────▼────────────────────────────────┼──────────────────┐
│                     Indexer Worker           │                   │
│  ┌──────────────────────────────────────────┼────────────┐     │
│  │           internal/rpc (RPC Client)      │            │     │
│  │  • Connection pool, retry logic          │            │     │
│  │  • Error classification                  │            │     │
│  └──────────────┬───────────────────────────┘            │     │
│                 │                                         │     │
│                 ▼                                         │     │
│  ┌──────────────────────────────────────────────────┐   │     │
│  │       internal/ingest (Data Ingestion)           │   │     │
│  │  • Fetch blocks, parse blockchain data           │   │     │
│  │  • Normalize into internal domain models          │   │     │
│  └──────────────┬───────────────────────────────────┘   │     │
│                 │                                         │     │
│                 ▼                                         │     │
│  ┌──────────────────────────────────────────────────┐   │     │
│  │         internal/index (Indexing Logic)          │   │     │
│  │  • Parallel backfill (worker pool)               │   │     │
│  │  • Sequential live-tail                          │   │     │
│  │  • Reorg detection & recovery                    │   │     │
│  └──────────────┬───────────────────────────────────┘   │     │
│                 │                                         │     │
│                 ▼                                         │     │
│  ┌──────────────────────────────────────────────────┐   │     │
│  │      internal/store/pg (Storage Layer)           │   │     │
│  │  • Bulk inserts, upserts                         │   │     │
│  │  • Transaction management                        │   │     │
│  └──────────────┬───────────────────────────────────┘   │     │
└────────────────────────────────────────────────────────────────┘
                  │
                  │ SQL (INSERT, UPDATE)
                  │
┌─────────────────▼────────────────────────────────────────────────┐
│                       PostgreSQL 16                               │
│  ┌────────────┐  ┌──────────────┐  ┌───────────┐               │
│  │  blocks    │  │ transactions │  │   logs    │               │
│  └────────────┘  └──────────────┘  └───────────┘               │
│     + Composite indexes for fast queries                         │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           │ SQL (SELECT)
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│                        API Server                                 │
│  ┌──────────────────────────────────────────────────────┐       │
│  │         internal/api (HTTP Handlers)                  │       │
│  │  • REST endpoints (chi router)                        │       │
│  │  • WebSocket streaming                                │       │
│  │  • Request validation, pagination                     │       │
│  │  • Metrics middleware                                 │       │
│  └──────────────┬───────────────────────────────────────┘       │
│                 │                                                 │
│                 ▼                                                 │
│  ┌──────────────────────────────────────────────────────┐       │
│  │      internal/store/pg (Storage Layer)               │       │
│  │  • Query builders, pagination                        │       │
│  └──────────────────────────────────────────────────────┘       │
└───────────────────────┬──────────────────────────────────────────┘
                        │
                        │ HTTP/WebSocket
                        │
┌───────────────────────▼──────────────────────────────────────────┐
│                   Frontend SPA (web/index.html)                   │
│  • Live blocks ticker (WebSocket)                                 │
│  • Recent transactions table                                      │
│  • Search interface (tx hash, block, address)                    │
└───────────────────────────────────────────────────────────────────┘
```

### 2.2 Repository Structure

**Repository Strategy:** Monorepo

All code, configuration, and documentation in a single repository for simplicity.

```
go-blockchain-explorer/
├── cmd/
│   ├── api/           # API server entry point
│   └── worker/        # Indexer worker entry point
├── internal/          # Private application code
│   ├── rpc/           # RPC client layer
│   ├── ingest/        # Data ingestion
│   ├── index/         # Indexing logic
│   ├── store/         # Storage abstraction
│   │   └── pg/        # PostgreSQL implementation
│   ├── api/           # API handlers
│   └── util/          # Shared utilities (logging, metrics)
├── migrations/        # Database migrations
├── web/               # Frontend static files
├── docker/            # Docker configurations
├── docs/              # Documentation
├── go.mod             # Go module definition
├── go.sum             # Go module checksums
├── Makefile           # Build automation
├── docker-compose.yml # Local development environment
└── README.md          # Project documentation
```

### 2.3 Process Architecture

The system runs as **two separate processes**:

1. **Indexer Worker** (`cmd/worker/main.go`):
   - Runs backfill on startup (if needed)
   - Continuously monitors blockchain for new blocks
   - Handles reorg detection and recovery
   - Writes to PostgreSQL
   - Single instance (writes are serialized for consistency)

2. **API Server** (`cmd/api/main.go`):
   - Serves REST API endpoints
   - Manages WebSocket connections
   - Serves static frontend files
   - Reads from PostgreSQL (no writes)
   - Can run multiple instances (stateless)

**Communication:** Processes communicate indirectly via PostgreSQL. No inter-process messaging needed. This simplifies the architecture while allowing independent scaling and fault isolation.

### 2.4 Concurrency Model

**Indexer Worker:**
- **Backfill Phase:** Worker pool with configurable concurrency (default: 8 workers)
  - Workers fetch blocks in parallel from RPC
  - Coordinator collects results and performs bulk inserts
  - Bounded concurrency prevents RPC rate limiting

- **Live-Tail Phase:** Single goroutine for sequential processing
  - Maintains block ordering (parent-child relationships)
  - Detects reorgs by comparing parent hashes
  - Background goroutine updates lag metrics

**API Server:**
- One goroutine per HTTP request (standard Go HTTP server)
- WebSocket hub goroutine manages subscriptions
- Broadcaster goroutine receives updates and fans out to subscribers

### 2.5 Key Architectural Decisions

**ADR-001: Monorepo vs Polyrepo**
- **Decision:** Monorepo
- **Rationale:** Single developer, 7-day timeline, simpler dependency management, easier refactoring
- **Trade-offs:** Less isolation, but acceptable for portfolio project

**ADR-002: Separate Processes vs Single Binary**
- **Decision:** Separate processes (indexer + API)
- **Rationale:** Independent scaling, fault isolation, clear separation of concerns
- **Trade-offs:** Slightly more complex deployment, but still manageable with Docker Compose

**ADR-003: PostgreSQL vs Time-Series DB**
- **Decision:** PostgreSQL
- **Rationale:** Broad applicability, demonstrates relational DB skills, sufficient performance with proper indexing
- **Trade-offs:** ClickHouse would be faster for analytics queries, but adds complexity

**ADR-004: Vanilla JS vs React/Vue**
- **Decision:** Vanilla HTML/JavaScript
- **Rationale:** No build step, minimal complexity, keeps focus on backend
- **Trade-offs:** Less polished UI, but sufficient for portfolio demonstration

**ADR-005: Reorg Handling Strategy**
- **Decision:** Mark orphaned blocks (soft delete) rather than hard delete
- **Rationale:** Preserves historical data, allows reorg analysis, simpler rollback
- **Trade-offs:** Slightly more storage, but negligible for demo scale

## 3. Data Architecture

### 3.1 Database Schema

**Database:** PostgreSQL 16

**Schema Design Principles:**
- Normalized structure for blocks, transactions, and logs
- Composite indexes for common query patterns
- `orphaned` flag for soft-delete pattern during reorgs
- Native PostgreSQL types (bytea for hashes, numeric for wei values)

#### 3.1.1 Blocks Table

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

**Key Design Choices:**
- `height` as primary key (natural identifier for blocks)
- `hash` has unique constraint (prevents duplicate blocks)
- `parent_hash` enables reorg detection
- `orphaned` flag for soft-delete during reorgs
- `created_at`/`updated_at` for debugging

#### 3.1.2 Transactions Table

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

**Key Design Choices:**
- `hash` as primary key (unique transaction identifier)
- Foreign key to `blocks(height)` with CASCADE delete
- Composite indexes on `(from_addr, block_height)` and `(to_addr, block_height)` for fast address lookups
- `to_addr` nullable for contract creation transactions
- `success` boolean indicates transaction status

#### 3.1.3 Logs Table

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

**Key Design Choices:**
- `id` as surrogate primary key (logs don't have natural IDs)
- Foreign key to `transactions(hash)` with CASCADE delete
- Unique constraint on `(tx_hash, log_index)` prevents duplicates
- Separate columns for each topic (up to 4 topics per Ethereum log)
- Index on `(address, topic0)` for event filtering

### 3.2 Indexing Strategy

**Composite Indexes Rationale:**

1. **Address Transaction History** (`idx_tx_from_addr_block`, `idx_tx_to_addr_block`):
   - Query: "Get all transactions for address X, ordered by block height"
   - Composite index on `(address, block_height DESC)` allows index-only scan
   - Covers both sent and received transactions

2. **Reorg Queries** (`idx_blocks_orphaned_height`):
   - Query: "Get all non-orphaned blocks in order"
   - Composite index on `(orphaned, height DESC)` enables fast filtering

3. **Event Log Filtering** (`idx_logs_address_topic0`):
   - Query: "Get logs for contract address X with topic0 Y"
   - Composite index enables efficient event filtering

### 3.3 Data Flow

**Write Path (Indexer Worker):**
1. RPC client fetches block from Ethereum node
2. Ingestion layer parses block data into domain models
3. Indexing layer coordinates writing:
   - Backfill: Batches of blocks inserted in transaction
   - Live-tail: Single block per transaction
   - Reorg: Mark orphaned blocks + insert new blocks in same transaction
4. Storage layer executes SQL with `pgx` connection pool

**Read Path (API Server):**
1. HTTP request arrives at API handler
2. Handler validates input, constructs query parameters
3. Storage layer executes parameterized query
4. Results marshaled to JSON and returned
5. WebSocket subscribers receive real-time updates

### 3.4 Migration Strategy

**Tool:** golang-migrate/migrate

**Migration Files:** `migrations/{version}_{description}.{up|down}.sql`

Example:
```
migrations/
├── 000001_initial_schema.up.sql
├── 000001_initial_schema.down.sql
├── 000002_add_indexes.up.sql
└── 000002_add_indexes.down.sql
```

**Execution:** Migrations run automatically on service startup via `migrate` library embedded in both `cmd/worker` and `cmd/api`.

**Version Tracking:** `schema_migrations` table created by `migrate` tracks applied migrations.

## 4. Component & Integration Overview

### 4.1 Component Breakdown

#### 4.1.1 RPC Client Layer (`internal/rpc/`)

**Responsibilities:**
- Establish and maintain connection to Ethereum RPC endpoint
- Execute JSON-RPC methods (`eth_getBlockByNumber`, `eth_getTransactionReceipt`, etc.)
- Retry transient failures with exponential backoff
- Classify errors (transient vs permanent)
- Connection pooling and timeout handling

**Key Types:**
```go
type Client struct {
    ethClient *ethclient.Client
    config    *Config
    metrics   *Metrics
}

type Config struct {
    RPCURL          string
    Timeout         time.Duration
    MaxRetries      int
    RetryBackoff    time.Duration
}
```

**Error Handling:**
- Network errors → Retry with exponential backoff (max 5 retries)
- Rate limit errors → Backoff and retry
- Invalid parameters → Fail immediately
- All errors logged with context

#### 4.1.2 Ingestion Layer (`internal/ingest/`)

**Responsibilities:**
- Fetch blocks from RPC client
- Parse raw blockchain data
- Normalize into internal domain models
- Extract transactions and logs

**Key Types:**
```go
type Block struct {
    Height      uint64
    Hash        []byte
    ParentHash  []byte
    Miner       []byte
    GasUsed     *big.Int
    Timestamp   uint64
    Transactions []Transaction
}

type Transaction struct {
    Hash      []byte
    FromAddr  []byte
    ToAddr    []byte
    ValueWei  *big.Int
    // ...
}
```

**Design Pattern:** Domain model separation from blockchain types (`geth` types → internal types)

#### 4.1.3 Indexing Layer (`internal/index/`)

**Responsibilities:**
- Coordinate backfill (parallel workers)
- Coordinate live-tail (sequential processing)
- Detect and handle reorgs
- Manage indexing state

**Key Components:**

**Backfill Coordinator:**
```go
type BackfillCoordinator struct {
    rpcClient     *rpc.Client
    ingester      *ingest.Ingester
    store         store.Store
    workerCount   int
    batchSize     int
}

func (bc *BackfillCoordinator) Backfill(ctx context.Context, startHeight, endHeight uint64) error {
    // Create worker pool
    // Distribute blocks across workers
    // Collect results and bulk insert
}
```

**Live-Tail Coordinator:**
```go
type LiveTailCoordinator struct {
    rpcClient *rpc.Client
    ingester  *ingest.Ingester
    store     store.Store
    pollInterval time.Duration
}

func (ltc *LiveTailCoordinator) Start(ctx context.Context) error {
    // Poll for new blocks every N seconds
    // Check parent hash matches DB head
    // If mismatch, trigger reorg handling
    // Insert block sequentially
}
```

**Reorg Handler:**
```go
type ReorgHandler struct {
    store store.Store
    maxDepth int
}

func (rh *ReorgHandler) HandleReorg(ctx context.Context, newBlock *ingest.Block) error {
    // Walk backwards to find common ancestor
    // Mark orphaned blocks
    // Re-process canonical chain from fork point
}
```

#### 4.1.4 Storage Layer (`internal/store/pg/`)

**Responsibilities:**
- Abstract database operations
- Provide repository pattern interface
- Execute parameterized queries
- Manage transactions

**Interface:**
```go
type Store interface {
    InsertBlocks(ctx context.Context, blocks []*ingest.Block) error
    GetBlockByHeight(ctx context.Context, height uint64) (*ingest.Block, error)
    GetBlockByHash(ctx context.Context, hash []byte) (*ingest.Block, error)
    GetTransactionsByAddress(ctx context.Context, addr []byte, limit, offset int) ([]*ingest.Transaction, error)
    MarkBlocksOrphaned(ctx context.Context, heights []uint64) error
    // ... more methods
}
```

**Implementation:** Uses `pgx` for high-performance PostgreSQL access, connection pooling, and batch operations.

#### 4.1.5 API Layer (`internal/api/`)

**Responsibilities:**
- HTTP request handling (chi router)
- WebSocket connection management
- Request validation
- Response formatting
- Metrics middleware

**REST Endpoints:**
- `GET /v1/blocks` - List recent blocks
- `GET /v1/blocks/{height}` - Block by height
- `GET /v1/txs/{hash}` - Transaction details
- `GET /v1/address/{addr}/txs` - Address transaction history
- `GET /v1/logs` - Query logs
- `GET /v1/stats/chain` - Chain statistics
- `GET /metrics` - Prometheus metrics
- `GET /health` - Health check

**WebSocket:**
- `WS /v1/stream` - Real-time updates (channels: `newBlocks`, `newTxs`)

**Middleware Stack:**
- CORS
- Request logging
- Metrics collection
- Panic recovery

### 4.2 Integration Points

**External Integrations:**
1. **Ethereum RPC Node**
   - Protocol: JSON-RPC over HTTP/WebSocket
   - Providers: Alchemy, Infura, or public Sepolia nodes
   - Configuration: RPC URL via environment variable
   - Retry logic handles transient failures

2. **Prometheus (Optional)**
   - Protocol: HTTP (scrape `/metrics`)
   - Pull-based metrics collection
   - No code changes needed to integrate

**Internal Integration:**
- Components communicate via Go interfaces
- No network calls between internal components
- Dependency injection for testability

### 4.3 Epic-to-Component Mapping

| Epic | Primary Components | Data Models | APIs |
|------|-------------------|-------------|------|
| **Epic 1: Core Indexing & Data Pipeline** | `internal/rpc`, `internal/ingest`, `internal/index`, `internal/store/pg`, migrations | blocks, transactions, logs | N/A (internal) |
| **Epic 2: API Layer & User Interface** | `internal/api`, `web/` | blocks, transactions, logs (read-only) | REST endpoints, WebSocket |

## 5. Architecture Decision Records

### ADR-001: Use Modular Monolith with Separate Processes

**Status:** Accepted

**Context:**
Portfolio project with 7-day timeline. Need to demonstrate production-ready patterns without excessive complexity.

**Decision:**
Use modular monolith architecture with clear layer separation (RPC, ingestion, indexing, storage, API), but run as two separate processes (indexer worker + API server).

**Consequences:**
- **Positive:** Clear separation of concerns, independent scaling, fault isolation
- **Positive:** Simpler than microservices, no service mesh needed
- **Negative:** Slightly more complex than single binary, but manageable with Docker Compose

**Alternatives Considered:**
- Single binary: Simpler deployment, but less demonstration of scaling patterns
- Microservices: Too complex for 7-day timeline, overkill for scale

---

### ADR-002: PostgreSQL Over Time-Series Databases

**Status:** Accepted

**Context:**
Need to store and query blockchain data efficiently. Time-series databases (ClickHouse, TimescaleDB) offer better analytics performance.

**Decision:**
Use PostgreSQL 16 with composite indexes.

**Consequences:**
- **Positive:** Broad applicability, demonstrates relational DB skills, sufficient performance
- **Positive:** ACID guarantees, strong consistency, familiar query patterns
- **Negative:** Not optimal for complex analytics queries (acceptable for portfolio scope)

**Alternatives Considered:**
- ClickHouse: Better for analytics, but adds complexity and less common in job requirements
- TimescaleDB: Good middle ground, but requires PostgreSQL extension setup

---

### ADR-003: Worker Pool Pattern for Backfill

**Status:** Accepted

**Context:**
Need to backfill 5,000 blocks in under 5 minutes. Sequential processing would take too long.

**Decision:**
Use worker pool pattern with configurable concurrency (default: 8 workers) for parallel block fetching during backfill.

**Consequences:**
- **Positive:** Achieves performance target (5,000 blocks in <5 min)
- **Positive:** Demonstrates Go concurrency patterns
- **Positive:** Bounded concurrency prevents RPC rate limiting
- **Negative:** More complex than sequential processing, but manageable

**Implementation Notes:**
- Workers fetch blocks in parallel
- Coordinator collects results and performs bulk inserts
- Configurable via environment variable

---

### ADR-004: Soft Delete for Reorgs

**Status:** Accepted

**Context:**
Chain reorganizations require handling orphaned blocks. Could delete them or mark as orphaned.

**Decision:**
Mark orphaned blocks with `orphaned = TRUE` flag rather than deleting them.

**Consequences:**
- **Positive:** Preserves historical data for analysis
- **Positive:** Simpler rollback if needed
- **Positive:** Allows querying orphaned blocks for debugging
- **Negative:** Slightly more storage, but negligible at demo scale

**Implementation Notes:**
- `orphaned` boolean column on `blocks` table
- Index on `(orphaned, height)` for efficient filtering
- API queries filter `WHERE orphaned = FALSE`

---

### ADR-005: Vanilla HTML/JS for Frontend

**Status:** Accepted

**Context:**
Need minimal frontend to demonstrate live updates and search. Could use React/Vue or vanilla JS.

**Decision:**
Use vanilla HTML/JavaScript with no build step.

**Consequences:**
- **Positive:** No build complexity (webpack, npm scripts)
- **Positive:** Keeps focus on backend capabilities
- **Positive:** Faster development (no framework learning curve)
- **Negative:** Less polished UI, but sufficient for portfolio demonstration

**Implementation Notes:**
- Native WebSocket API for real-time updates
- Fetch API for REST calls
- Simple DOM manipulation (no virtual DOM)

---

### ADR-006: chi Router Over Gin/Echo

**Status:** Accepted

**Context:**
Need HTTP router for REST API. Popular options: chi, Gin, Echo, gorilla/mux.

**Decision:**
Use chi router (v5).

**Consequences:**
- **Positive:** Lightweight and idiomatic, stays close to standard library
- **Positive:** Excellent middleware support
- **Positive:** Good performance without heavy dependencies
- **Negative:** Less feature-rich than Gin, but sufficient for this project

**Alternatives Considered:**
- Gin: More features, but heavier and more opinionated
- Echo: Similar to chi, but chi has better middleware ecosystem
- gorilla/mux: Older, chi is more modern

---

### ADR-007: Prometheus for Metrics

**Status:** Accepted

**Context:**
Need observability for indexer performance and API latency.

**Decision:**
Use Prometheus with `prometheus/client_golang` library.

**Consequences:**
- **Positive:** Industry standard, excellent for time-series metrics
- **Positive:** Pull-based (no need to push to external service)
- **Positive:** Wide ecosystem support
- **Negative:** Requires Prometheus server for visualization, but optional for demo

**Metrics Exposed:**
- `explorer_blocks_indexed_total` (counter)
- `explorer_index_lag_blocks` (gauge)
- `explorer_rpc_errors_total` (counter)
- `explorer_api_latency_ms` (histogram)

---

### ADR-008: log/slog for Structured Logging

**Status:** Accepted

**Context:**
Need structured logging for debugging and operational visibility.

**Decision:**
Use standard library `log/slog` (Go 1.21+ and available in all Go 1.24+ versions).

**Consequences:**
- **Positive:** Native to Go 1.21+, zero dependencies
- **Positive:** Structured JSON output
- **Positive:** Performant and well-integrated
- **Negative:** Less feature-rich than zerolog or zap, but sufficient

**Log Format:**
```json
{
  "time": "2025-10-29T10:30:00Z",
  "level": "INFO",
  "msg": "Block indexed",
  "block_height": 5000,
  "tx_count": 42
}
```

## 6. Implementation Guidance

### 6.1 Development Workflow

**Day 1: Foundation**
1. Initialize Go module and directory structure
2. Set up Docker Compose with PostgreSQL
3. Implement RPC client with retry logic
4. Create database schema and migrations
5. Test RPC connection and basic block fetching

**Day 2: Backfill Pipeline**
1. Implement ingestion layer (parse blocks)
2. Build storage layer with pgx
3. Create worker pool for parallel backfill
4. Add bulk insert optimization
5. Test backfilling 1,000 blocks

**Day 3: Live-Tail & Reorg**
1. Implement live-tail coordinator
2. Add reorg detection logic
3. Implement reorg recovery
4. Add Prometheus metrics
5. Add structured logging

**Day 4: REST API**
1. Set up chi router and middleware
2. Implement REST endpoints (blocks, txs, address)
3. Add pagination support
4. Add health check and metrics endpoints
5. Test API with curl/Postman

**Day 5: WebSocket & Frontend**
1. Implement WebSocket streaming
2. Create minimal HTML frontend
3. Add live blocks ticker
4. Add transaction search interface
5. Test real-time updates

**Day 6: Testing & Performance**
1. Write unit tests for critical paths
2. Add integration tests for indexer
3. Performance testing (backfill speed, API latency)
4. Fix bugs and optimize
5. Validate acceptance criteria

**Day 7: Documentation & Polish**
1. Write comprehensive README
2. Create API documentation (API.md)
3. Write architecture documentation (Design.md)
4. Add demo script and screenshots
5. Final testing and cleanup

### 6.2 Critical Implementation Notes

**RPC Client Retry Logic:**
```go
func (c *Client) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
    var block *types.Block
    var err error

    for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
        block, err = c.ethClient.BlockByNumber(ctx, big.NewInt(int64(height)))
        if err == nil {
            return block, nil
        }

        if isTransientError(err) {
            backoff := time.Duration(attempt) * c.config.RetryBackoff
            time.Sleep(backoff)
            continue
        }

        return nil, err // Permanent error, don't retry
    }

    return nil, fmt.Errorf("max retries exceeded: %w", err)
}
```

**Reorg Detection:**
```go
func (ltc *LiveTailCoordinator) processBlock(ctx context.Context, block *ingest.Block) error {
    // Get current head from database
    dbHead, err := ltc.store.GetLatestBlock(ctx)
    if err != nil {
        return err
    }

    // Check if new block's parent matches DB head
    if !bytes.Equal(block.ParentHash, dbHead.Hash) {
        // Reorg detected!
        return ltc.reorgHandler.HandleReorg(ctx, block)
    }

    // Normal case: append block
    return ltc.store.InsertBlocks(ctx, []*ingest.Block{block})
}
```

**Bulk Insert Optimization:**
```go
func (s *PostgresStore) InsertBlocks(ctx context.Context, blocks []*ingest.Block) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // Bulk insert blocks using COPY or batch insert
    _, err = tx.CopyFrom(
        ctx,
        pgx.Identifier{"blocks"},
        []string{"height", "hash", "parent_hash", "miner", "gas_used", "timestamp", "tx_count"},
        pgx.CopyFromSlice(len(blocks), func(i int) ([]interface{}, error) {
            b := blocks[i]
            return []interface{}{b.Height, b.Hash, b.ParentHash, b.Miner, b.GasUsed, b.Timestamp, b.TxCount}, nil
        }),
    )
    if err != nil {
        return err
    }

    // Bulk insert transactions
    // ... similar pattern

    return tx.Commit(ctx)
}
```

### 6.3 Configuration Management

**Environment Variables:**
```bash
# RPC Configuration
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
RPC_TIMEOUT=30s
RPC_MAX_RETRIES=5

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=postgres
DB_MAX_CONNS=20

# Indexer Configuration
BACKFILL_START_HEIGHT=0
BACKFILL_END_HEIGHT=5000
BACKFILL_WORKERS=8
BACKFILL_BATCH_SIZE=100
LIVE_TAIL_POLL_INTERVAL=5s

# API Configuration
API_PORT=8080
API_CORS_ORIGINS=*

# Metrics
METRICS_ENABLED=true
METRICS_PORT=9090
```

### 6.4 Testing Strategy

**Unit Tests:**
- RPC client (mocked ethclient)
- Ingestion layer (parse blocks)
- Reorg handler (various fork scenarios)
- Storage layer (mocked pgx)

**Integration Tests:**
- End-to-end backfill (test database)
- Reorg recovery (simulate fork)
- API endpoints (test database)

**Test Coverage Target:** >70% for critical paths

**Testing Tools:**
- Go standard library `testing`
- `testify/assert` for assertions
- `testcontainers-go` for PostgreSQL test instances (optional)

## 7. Proposed Source Tree

```
go-blockchain-explorer/
├── cmd/
│   ├── api/
│   │   └── main.go                    # API server entry point
│   └── worker/
│       └── main.go                    # Indexer worker entry point
│
├── internal/
│   ├── rpc/
│   │   ├── client.go                  # RPC client with retry logic
│   │   ├── client_test.go
│   │   └── errors.go                  # Error classification
│   │
│   ├── ingest/
│   │   ├── ingester.go                # Block ingestion and parsing
│   │   ├── ingester_test.go
│   │   └── models.go                  # Domain models (Block, Transaction, Log)
│   │
│   ├── index/
│   │   ├── backfill.go                # Backfill coordinator with worker pool
│   │   ├── backfill_test.go
│   │   ├── livetail.go                # Live-tail coordinator
│   │   ├── livetail_test.go
│   │   ├── reorg.go                   # Reorg detection and recovery
│   │   └── reorg_test.go
│   │
│   ├── store/
│   │   ├── store.go                   # Storage interface
│   │   └── pg/
│   │       ├── postgres.go            # PostgreSQL implementation
│   │       ├── postgres_test.go
│   │       ├── blocks.go              # Block queries
│   │       ├── transactions.go        # Transaction queries
│   │       └── logs.go                # Log queries
│   │
│   ├── api/
│   │   ├── server.go                  # HTTP server setup
│   │   ├── handlers.go                # REST endpoint handlers
│   │   ├── handlers_test.go
│   │   ├── websocket.go               # WebSocket hub and handlers
│   │   ├── middleware.go              # Logging, metrics, CORS
│   │   └── pagination.go              # Pagination utilities
│   │
│   └── util/
│       ├── logger.go                  # Structured logging setup
│       ├── metrics.go                 # Prometheus metrics
│       └── config.go                  # Configuration loading
│
├── migrations/
│   ├── 000001_initial_schema.up.sql
│   ├── 000001_initial_schema.down.sql
│   ├── 000002_add_indexes.up.sql
│   └── 000002_add_indexes.down.sql
│
├── web/
│   ├── index.html                     # Single-page frontend
│   ├── style.css                      # Minimal CSS
│   └── app.js                         # WebSocket client and UI logic
│
├── docker/
│   ├── Dockerfile.api                 # API server container
│   ├── Dockerfile.worker              # Indexer worker container
│   └── postgres/
│       └── init.sql                   # Database initialization (optional)
│
├── docs/
│   ├── README.md                      # Project overview
│   ├── API.md                         # API documentation
│   ├── Design.md                      # Architecture and design decisions
│   ├── PRD.md                         # Product Requirements Document
│   ├── epic-stories.md                # Epic and story breakdown
│   ├── solution-architecture.md       # This document
│   └── product-brief-*.md             # Product brief
│
├── .gitignore
├── go.mod                             # Go module definition
├── go.sum                             # Go module checksums
├── Makefile                           # Build automation
├── docker-compose.yml                 # Local development environment
└── README.md                          # Getting started guide
```

**Key Directory Purposes:**
- `cmd/`: Entry points for binaries (api, worker)
- `internal/`: Private application code (not importable by other projects)
- `migrations/`: Database schema migrations
- `web/`: Static frontend files
- `docker/`: Docker and Docker Compose configurations
- `docs/`: All project documentation

## 8. Testing Strategy

### 8.1 Testing Approach

**Test Pyramid:**
- **Unit Tests:** Fast, isolated tests for individual functions/methods
- **Integration Tests:** Test component interactions (e.g., storage layer with real DB)
- **End-to-End Tests:** Full workflow tests (backfill, API queries)

**Coverage Target:** >70% for critical paths

**Testing Tools:**
- Go standard library `testing`
- `testify/assert` and `testify/require` for assertions
- `testify/mock` for mocking interfaces
- `testcontainers-go` for PostgreSQL test instances (optional, can use Docker Compose)

### 8.2 Unit Test Examples

**RPC Client:**
```go
func TestClient_GetBlockByNumber_Success(t *testing.T) {
    // Mock ethclient
    mockClient := &MockEthClient{}
    mockClient.On("BlockByNumber", mock.Anything, big.NewInt(100)).
        Return(&types.Block{...}, nil)

    client := &Client{ethClient: mockClient}

    block, err := client.GetBlockByNumber(context.Background(), 100)

    require.NoError(t, err)
    assert.NotNil(t, block)
}

func TestClient_GetBlockByNumber_RetryOnTransientError(t *testing.T) {
    // Test retry logic
    mockClient := &MockEthClient{}
    mockClient.On("BlockByNumber", mock.Anything, big.NewInt(100)).
        Return(nil, errors.New("network timeout")).Once()
    mockClient.On("BlockByNumber", mock.Anything, big.NewInt(100)).
        Return(&types.Block{...}, nil).Once()

    client := &Client{ethClient: mockClient, config: &Config{MaxRetries: 3}}

    block, err := client.GetBlockByNumber(context.Background(), 100)

    require.NoError(t, err)
    assert.NotNil(t, block)
}
```

**Reorg Handler:**
```go
func TestReorgHandler_HandleReorg_SimpleReorg(t *testing.T) {
    // Setup: DB has blocks 1-100
    // New block 101 has parent_hash that doesn't match block 100
    // Should mark block 100 as orphaned and insert new block 100 + 101

    mockStore := &MockStore{}
    mockStore.On("GetLatestBlock", mock.Anything).Return(&Block{Height: 100, Hash: []byte("old100")}, nil)
    mockStore.On("MarkBlocksOrphaned", mock.Anything, []uint64{100}).Return(nil)
    mockStore.On("InsertBlocks", mock.Anything, mock.Anything).Return(nil)

    handler := &ReorgHandler{store: mockStore, maxDepth: 6}

    newBlock := &Block{Height: 101, ParentHash: []byte("new100")}
    err := handler.HandleReorg(context.Background(), newBlock)

    require.NoError(t, err)
    mockStore.AssertExpectations(t)
}
```

### 8.3 Integration Test Examples

**Storage Layer:**
```go
func TestPostgresStore_InsertBlocks_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Start test database (Docker or testcontainers)
    db := setupTestDB(t)
    defer db.Close()

    store := NewPostgresStore(db)

    blocks := []*Block{
        {Height: 1, Hash: []byte("hash1"), ParentHash: []byte("genesis"), ...},
        {Height: 2, Hash: []byte("hash2"), ParentHash: []byte("hash1"), ...},
    }

    err := store.InsertBlocks(context.Background(), blocks)
    require.NoError(t, err)

    // Verify blocks were inserted
    fetched, err := store.GetBlockByHeight(context.Background(), 1)
    require.NoError(t, err)
    assert.Equal(t, blocks[0].Hash, fetched.Hash)
}
```

**API Endpoints:**
```go
func TestAPI_GetBlock_Success(t *testing.T) {
    // Setup test database with sample data
    db := setupTestDB(t)
    defer db.Close()

    // Start test server
    server := setupTestServer(db)
    defer server.Close()

    // Make request
    resp, err := http.Get(server.URL + "/v1/blocks/100")
    require.NoError(t, err)
    defer resp.Body.Close()

    // Verify response
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var block BlockResponse
    json.NewDecoder(resp.Body).Decode(&block)
    assert.Equal(t, uint64(100), block.Height)
}
```

### 8.4 End-to-End Test Scenarios

1. **Backfill Workflow:**
   - Start with empty database
   - Run backfill for 100 blocks
   - Verify all blocks, transactions, and logs are indexed
   - Check metrics are updated correctly

2. **Reorg Recovery:**
   - Index blocks 1-100
   - Simulate reorg at block 95
   - Verify blocks 95-100 marked as orphaned
   - Verify new canonical chain indexed correctly

3. **API Query Performance:**
   - Index 5,000 blocks
   - Query address transaction history
   - Measure p95 latency (should be <150ms)

4. **WebSocket Streaming:**
   - Connect WebSocket client
   - Index new block
   - Verify client receives update within 1 second

### 8.5 Performance Benchmarks

**Backfill Performance:**
```bash
# Benchmark: Backfill 5,000 blocks
make benchmark-backfill

Expected: <5 minutes on standard hardware
Measured: [To be filled during implementation]
```

**API Latency:**
```bash
# Benchmark: API endpoint latency
go test -bench=BenchmarkGetBlock -benchmem

Expected: p95 <150ms
Measured: [To be filled during implementation]
```

### 8.6 Test Execution

**Running Tests:**
```bash
# Run all unit tests
make test

# Run unit tests with coverage
make test-coverage

# Run integration tests (requires Docker)
make test-integration

# Run all tests including integration
make test-all
```

**CI/CD Integration:**
- Unit tests run on every commit
- Integration tests run on PR merge
- Coverage report generated and tracked

## 9. Deployment & Operations

### 9.1 Local Development Setup

**Prerequisites:**
- Docker 24.0+
- Docker Compose 2.21+
- Go 1.24+ (for local development without Docker, required by go-ethereum v1.16.5)
- Make (optional, for build automation)

**Quick Start:**
```bash
# Clone repository
git clone https://github.com/yourusername/go-blockchain-explorer
cd go-blockchain-explorer

# Copy example environment file
cp .env.example .env

# Edit .env with your Ethereum RPC URL (Alchemy/Infura API key)
vim .env

# Start all services
docker compose up

# Services will be available at:
# - API: http://localhost:8080
# - Frontend: http://localhost:8080
# - Metrics: http://localhost:9090/metrics
# - PostgreSQL: localhost:5432
```

### 9.2 Docker Compose Configuration

**docker-compose.yml:**
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: blockchain_explorer
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  worker:
    build:
      context: .
      dockerfile: docker/Dockerfile.worker
    env_file:
      - .env
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: blockchain_explorer
      DB_USER: postgres
      DB_PASSWORD: postgres
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  api:
    build:
      context: .
      dockerfile: docker/Dockerfile.api
    env_file:
      - .env
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: blockchain_explorer
      DB_USER: postgres
      DB_PASSWORD: postgres
      API_PORT: 8080
    ports:
      - "8080:8080"
      - "9090:9090"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

### 9.3 Health Checks

**API Health Check:**
```
GET /health

Response:
{
  "status": "healthy",
  "database": "connected",
  "indexer_last_block": 5000,
  "indexer_last_updated": "2025-10-29T10:30:00Z",
  "indexer_lag_seconds": 5,
  "version": "1.0.0"
}
```

**Database Health:**
- Simple SELECT query to verify connectivity
- Check last indexed block timestamp (< 5 minutes = healthy)

**RPC Health:**
- Check last successful RPC call timestamp
- Track RPC error rate

### 9.4 Monitoring & Observability

**Prometheus Metrics:**
```
# Indexer metrics
explorer_blocks_indexed_total{} counter
explorer_index_lag_blocks{} gauge
explorer_index_lag_seconds{} gauge
explorer_rpc_errors_total{error_type="network"} counter
explorer_rpc_errors_total{error_type="rate_limit"} counter

# API metrics
explorer_api_requests_total{method="GET",endpoint="/v1/blocks",status="200"} counter
explorer_api_latency_ms{method="GET",endpoint="/v1/blocks"} histogram
explorer_api_websocket_connections{} gauge
```

**Structured Logs:**
```json
// Indexer logs
{"time":"2025-10-29T10:30:00Z","level":"INFO","msg":"Backfill started","start_height":0,"end_height":5000,"workers":8}
{"time":"2025-10-29T10:30:05Z","level":"INFO","msg":"Backfill progress","indexed":500,"total":5000,"percent":10}
{"time":"2025-10-29T10:32:00Z","level":"INFO","msg":"Backfill complete","duration_seconds":120,"blocks_per_second":41.67}

// Reorg logs
{"time":"2025-10-29T10:35:00Z","level":"WARN","msg":"Reorg detected","fork_point":4995,"depth":5}
{"time":"2025-10-29T10:35:01Z","level":"INFO","msg":"Reorg recovered","orphaned_blocks":5,"new_blocks":5}

// API logs
{"time":"2025-10-29T10:40:00Z","level":"INFO","msg":"API request","method":"GET","path":"/v1/blocks/5000","status":200,"latency_ms":42}
```

**Dashboard Metrics (if Prometheus + Grafana used):**
- Blocks indexed over time
- Indexer lag (blocks and seconds)
- API request rate and latency
- RPC error rate
- WebSocket connection count

### 9.5 Operational Runbook

**Common Operations:**

**1. Restart Indexer:**
```bash
docker compose restart worker

# Check logs
docker compose logs -f worker
```

**2. View API Logs:**
```bash
docker compose logs -f api
```

**3. Check Indexer Status:**
```bash
curl http://localhost:8080/health | jq
```

**4. View Metrics:**
```bash
curl http://localhost:9090/metrics
```

**5. Manual Database Query:**
```bash
docker compose exec postgres psql -U postgres -d blockchain_explorer

# Check latest block
SELECT height, hash, tx_count, orphaned FROM blocks ORDER BY height DESC LIMIT 10;

# Check indexer progress
SELECT COUNT(*) FROM blocks WHERE orphaned = FALSE;
```

**Troubleshooting:**

**Issue: Indexer not progressing**
- Check RPC connectivity: `docker compose logs worker | grep RPC`
- Verify RPC_URL in `.env` is correct
- Check rate limits: Look for "rate limit" errors in logs

**Issue: API returning errors**
- Check database connectivity: `docker compose exec postgres pg_isready`
- Verify migrations ran: `SELECT version FROM schema_migrations;`
- Check API logs: `docker compose logs api`

**Issue: High indexer lag**
- Increase worker count: Set `BACKFILL_WORKERS=16` in `.env`
- Check RPC performance: Monitor `explorer_rpc_errors_total`
- Verify database performance: Check PostgreSQL logs for slow queries

### 9.6 Scaling Considerations

**For Portfolio Demo:**
- Single instance of each service is sufficient
- Standard hardware (4 cores, 8GB RAM) adequate

**If Scaling Beyond Demo:**
- **API Server:** Can run multiple instances behind load balancer (stateless)
- **Indexer Worker:** Single instance for consistency (writes must be serialized)
- **Database:** Add read replicas for API queries (write to primary)
- **RPC:** Use multiple RPC providers with failover

## 10. Security

### 10.1 Security Considerations

**Authentication & Authorization:**
- **Not Implemented in MVP:** No authentication for demo purposes
- **Future:** API key-based authentication for rate limiting and access control

**Data Security:**
- **Public Data:** Blockchain data is public, no sensitive information
- **Database:** Local PostgreSQL, no external access in demo setup
- **RPC Credentials:** API keys stored in `.env` file (not committed to git)

**Input Validation:**
- All API inputs validated (addresses as hex, block heights as positive integers)
- SQL injection prevented via parameterized queries (pgx)
- WebSocket message validation

**CORS:**
- Configurable CORS origins (default: allow all for demo)
- Production: Restrict to specific origins

**Rate Limiting:**
- **Not Implemented in MVP:** Simple demo doesn't require rate limiting
- **Future:** Implement per-IP or per-API-key rate limiting

**Secrets Management:**
- Environment variables for configuration (not hardcoded)
- `.env` file excluded from git (`.gitignore`)
- Docker secrets for production deployment

### 10.2 Security Best Practices

1. **Keep Dependencies Updated:**
   - Regularly update Go modules: `go get -u`
   - Monitor security advisories for dependencies

2. **Database Security:**
   - Use strong passwords in production
   - Restrict PostgreSQL access to localhost or internal network
   - Use SSL/TLS for database connections in production

3. **RPC Security:**
   - Rotate API keys periodically
   - Use multiple RPC providers for redundancy
   - Monitor RPC usage to detect anomalies

4. **Docker Security:**
   - Run containers as non-root user
   - Use minimal base images (alpine)
   - Scan images for vulnerabilities

### 10.3 Security Specialist Handoff (Optional)

For production deployment beyond portfolio demonstration, consider engaging a security specialist to review:

- **Infrastructure Security:** Network segmentation, firewall rules, TLS configuration
- **Application Security:** OWASP Top 10 vulnerabilities, penetration testing
- **Compliance:** If handling any sensitive data (unlikely for blockchain explorer)
- **Incident Response:** Monitoring, alerting, and response procedures

---

## Specialist Sections

### Testing Specialist Section

**Status:** Handled inline (see Section 8)

**Coverage:** Unit tests, integration tests, end-to-end tests, performance benchmarks

### DevOps Specialist Section

**Status:** Handled inline (see Section 9)

**Coverage:** Docker Compose setup, health checks, monitoring, operational runbook

### Security Specialist Section

**Status:** Handled inline (see Section 10)

**Coverage:** Security considerations, best practices, input validation

**Note:** For production deployment beyond portfolio demonstration, consider engaging specialists for:
- Advanced DevOps (Kubernetes, multi-region)
- Comprehensive security audit (penetration testing, compliance)
- Advanced testing (load testing, chaos engineering)

---

_Generated using BMad Method Solution Architecture workflow_
