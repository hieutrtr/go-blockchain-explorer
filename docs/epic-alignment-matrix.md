# Epic Alignment Matrix - Blockchain Explorer

**Date:** 2025-10-29
**Project:** Blockchain Explorer
**Purpose:** Maps epics to architecture components, data models, APIs, and integration points for implementation guidance

---

## Matrix Overview

This document provides a comprehensive mapping between product epics (from PRD) and technical architecture components (from solution-architecture.md), enabling developers to quickly understand:

- Which architecture components implement which epic
- What data models are involved
- Which APIs are exposed
- What external integrations are needed
- Implementation dependencies and sequencing

---

## Epic 1: Core Indexing & Data Pipeline

**Epic Goal:** Build production-grade blockchain data pipeline with parallel backfill, real-time indexing, and reorg handling

**Stories:** 9 stories (1.1 - 1.9)
**Implementation Timeline:** Days 1-3 of 7-day sprint
**Priority:** CRITICAL - Foundation for entire system

### Architecture Component Mapping

| Story | Component Path | Component Name | Responsibility |
|-------|---------------|----------------|----------------|
| 1.1 RPC Client | `internal/rpc/` | RPC Client Layer | Ethereum node communication with retry logic |
| 1.2 Schema & Migrations | `migrations/` | Database Schema | PostgreSQL tables, indexes, foreign keys |
| 1.3 Parallel Backfill | `internal/index/backfill.go` | Backfill Coordinator | Worker pool for parallel historical block fetching |
| 1.3 Parallel Backfill | `internal/ingest/` | Ingestion Layer | Parse and normalize blockchain data |
| 1.4 Live-Tail | `internal/index/livetail.go` | Live-Tail Coordinator | Sequential new block processing |
| 1.5 Reorg Handling | `internal/index/reorg.go` | Reorg Handler | Detect forks and heal database |
| 1.6 Migrations | `migrations/` | Migration System | Schema versioning with golang-migrate |
| 1.7 Metrics | `internal/util/metrics.go` | Metrics | Prometheus instrumentation |
| 1.8 Logging | `internal/util/logger.go` | Logger | Structured JSON logging with log/slog |
| 1.9 Testing | `internal/*/`*_test.go` | Test Suite | Unit and integration tests |

### Data Model Mapping

| Data Model | Table Name | Owned By Stories | Used By Stories | Schema Location |
|------------|-----------|------------------|-----------------|-----------------|
| **Block** | `blocks` | 1.2, 1.3, 1.4, 1.5 | All Epic 1 stories, All Epic 2 stories | `migrations/000001_initial_schema.up.sql` |
| **Transaction** | `transactions` | 1.2, 1.3, 1.4 | All Epic 1 stories, All Epic 2 stories | `migrations/000001_initial_schema.up.sql` |
| **Log** | `logs` | 1.2, 1.3, 1.4 | 2.1 (logs query endpoint) | `migrations/000001_initial_schema.up.sql` |
| **Schema Version** | `schema_migrations` | 1.6 (golang-migrate) | All services | Auto-created by migration tool |

### Data Model Details

**blocks Table:**
```sql
height BIGINT PRIMARY KEY
hash BYTEA NOT NULL UNIQUE
parent_hash BYTEA NOT NULL
miner BYTEA NOT NULL
gas_used NUMERIC NOT NULL
gas_limit NUMERIC NOT NULL
timestamp BIGINT NOT NULL
tx_count INTEGER NOT NULL
orphaned BOOLEAN NOT NULL DEFAULT FALSE  -- Reorg support
created_at TIMESTAMP NOT NULL DEFAULT NOW()
updated_at TIMESTAMP NOT NULL DEFAULT NOW()
```

**transactions Table:**
```sql
hash BYTEA PRIMARY KEY
block_height BIGINT NOT NULL REFERENCES blocks(height) ON DELETE CASCADE
tx_index INTEGER NOT NULL
from_addr BYTEA NOT NULL
to_addr BYTEA  -- NULL for contract creation
value_wei NUMERIC NOT NULL
fee_wei NUMERIC NOT NULL
gas_used NUMERIC NOT NULL
gas_price NUMERIC NOT NULL
nonce BIGINT NOT NULL
success BOOLEAN NOT NULL
created_at TIMESTAMP NOT NULL DEFAULT NOW()
```

**logs Table:**
```sql
id BIGSERIAL PRIMARY KEY
tx_hash BYTEA NOT NULL REFERENCES transactions(hash) ON DELETE CASCADE
log_index INTEGER NOT NULL
address BYTEA NOT NULL
topic0 BYTEA, topic1 BYTEA, topic2 BYTEA, topic3 BYTEA
data BYTEA NOT NULL
created_at TIMESTAMP NOT NULL DEFAULT NOW()
UNIQUE(tx_hash, log_index)
```

### Critical Indexes

| Index Name | Table | Columns | Purpose | Performance Impact |
|------------|-------|---------|---------|-------------------|
| `idx_blocks_orphaned_height` | blocks | (orphaned, height DESC) | Reorg queries, API queries | Enables fast "non-orphaned blocks" filtering |
| `idx_blocks_timestamp` | blocks | (timestamp DESC) | Recent blocks queries | API `/v1/blocks` endpoint |
| `idx_blocks_hash` | blocks | (hash) | Block lookup by hash | API `/v1/blocks/{hash}` endpoint |
| `idx_tx_block_height` | transactions | (block_height) | Block's transactions | JOIN queries |
| `idx_tx_from_addr_block` | transactions | (from_addr, block_height DESC) | Address tx history | API `/v1/address/{addr}/txs` - sent transactions |
| `idx_tx_to_addr_block` | transactions | (to_addr, block_height DESC) | Address tx history | API `/v1/address/{addr}/txs` - received transactions |
| `idx_tx_block_index` | transactions | (block_height, tx_index) | Transaction ordering | Ensures correct tx sequence in block |
| `idx_logs_tx_hash` | logs | (tx_hash) | Transaction's logs | JOIN queries |
| `idx_logs_address_topic0` | logs | (address, topic0) | Event filtering | API `/v1/logs` endpoint |

### API Endpoints (Internal - No external APIs)

Epic 1 exposes **no external APIs**. All operations are internal to the indexer worker process.

**Internal Interfaces:**
- `internal/store.Store` interface (repository pattern) - used by all indexing components
- `internal/rpc.Client` interface - used by ingestion layer
- Prometheus metrics exposed via Epic 2's API server

### External Integration Points

| Integration | Type | Direction | Configuration | Error Handling |
|-------------|------|-----------|---------------|----------------|
| **Ethereum Sepolia RPC** | JSON-RPC over HTTP/WebSocket | Outbound | `RPC_URL` env var (Alchemy/Infura API key) | Exponential backoff retry (max 5 attempts) |
| **PostgreSQL 16** | SQL via pgx driver | Outbound | `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | Connection pool with automatic reconnect |

### Implementation Sequence

**Must implement in order:**

1. **Story 1.1 (RPC Client)** → **Story 1.2 (Schema)** - Foundation layer
   - RPC client enables blockchain data access
   - Schema enables data persistence

2. **Story 1.3 (Backfill)** - Depends on 1.1, 1.2
   - Uses RPC client to fetch blocks
   - Writes to database via store layer

3. **Story 1.4 (Live-Tail)** - Depends on 1.1, 1.2, 1.3
   - Continues where backfill leaves off
   - Uses same RPC and storage components

4. **Story 1.5 (Reorg)** - Depends on 1.4
   - Triggered by live-tail when parent hash mismatch detected
   - Requires working live-tail to detect reorgs

5. **Stories 1.6-1.9** - Can be done in parallel with above
   - 1.6 (Migrations): Part of 1.2 setup
   - 1.7 (Metrics): Instrument as you build
   - 1.8 (Logging): Instrument as you build
   - 1.9 (Testing): Write tests alongside implementation

### Dependencies on Other Epics

**Epic 1 is independent** - No dependencies on Epic 2

**Epic 2 depends on Epic 1:**
- Epic 2 reads data written by Epic 1
- Epic 2 APIs query database populated by Epic 1
- Epic 2 WebSocket streams updates as Epic 1 indexes new blocks

### Testing Strategy

| Story | Test Type | Test Location | Coverage Target |
|-------|-----------|---------------|-----------------|
| 1.1 RPC Client | Unit | `internal/rpc/client_test.go` | Retry logic, error handling |
| 1.2 Schema | Integration | `internal/store/pg/postgres_test.go` | Schema constraints, indexes |
| 1.3 Backfill | Integration | `internal/index/backfill_test.go` | Worker pool, bulk inserts |
| 1.4 Live-Tail | Integration | `internal/index/livetail_test.go` | Sequential processing, gap detection |
| 1.5 Reorg | Unit | `internal/index/reorg_test.go` | Fork detection, ancestor walk |
| 1.7 Metrics | Unit | `internal/util/metrics_test.go` | Metric registration, updates |
| 1.9 Integration | E2E | `internal/index/integration_test.go` | Full pipeline (RPC → DB) |

### Operational Metrics (Prometheus)

| Metric Name | Type | Labels | Purpose |
|-------------|------|--------|---------|
| `explorer_blocks_indexed_total` | Counter | - | Total blocks processed |
| `explorer_index_lag_blocks` | Gauge | - | Blocks behind network head |
| `explorer_index_lag_seconds` | Gauge | - | Time lag from network head |
| `explorer_rpc_errors_total` | Counter | `error_type` | RPC failures by type |
| `explorer_backfill_duration_seconds` | Histogram | - | Backfill performance |

### Configuration (Environment Variables)

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

# Metrics
METRICS_ENABLED=true
```

### Readiness Status

✅ **READY FOR IMPLEMENTATION**

- All components defined with clear boundaries
- Data models fully specified with SQL DDL
- Integration points documented with configuration
- Testing strategy defined
- Implementation sequence established
- No blocking dependencies

---

## Epic 2: API Layer & User Interface

**Epic Goal:** Provide RESTful and WebSocket APIs for querying indexed data, plus minimal demonstration frontend

**Stories:** 6 stories (2.1 - 2.6)
**Implementation Timeline:** Days 4-5 of 7-day sprint
**Priority:** HIGH - User-facing layer

### Architecture Component Mapping

| Story | Component Path | Component Name | Responsibility |
|-------|---------------|----------------|----------------|
| 2.1 REST Endpoints | `internal/api/handlers.go` | HTTP Handlers | REST API implementation |
| 2.1 REST Endpoints | `internal/api/server.go` | HTTP Server | chi router, middleware setup |
| 2.1 REST Endpoints | `internal/api/middleware.go` | Middleware | Logging, CORS, metrics, recovery |
| 2.1 REST Endpoints | `internal/store/pg/` | Storage Layer | Database queries (read-only) |
| 2.2 WebSocket | `internal/api/websocket.go` | WebSocket Hub | Connection management, pub/sub |
| 2.3 Pagination | `internal/api/pagination.go` | Pagination | LIMIT/OFFSET helpers |
| 2.4 Frontend | `web/index.html` | HTML Structure | Page layout, semantic markup |
| 2.4 Frontend | `web/style.css` | Styling | Minimal CSS for readability |
| 2.4 Frontend | `web/app.js` | JavaScript Logic | WebSocket client, DOM manipulation |
| 2.5 Search UI | `web/app.js` | Search Logic | Input validation, API calls, routing |
| 2.6 Health & Metrics | `internal/api/handlers.go` | Health Handler | Health check logic |
| 2.6 Health & Metrics | `internal/util/metrics.go` | Metrics Handler | Prometheus metrics endpoint |

### Data Model Mapping (Read-Only Access)

| Data Model | Table Name | Used By Stories | Query Patterns |
|------------|-----------|-----------------|----------------|
| **Block** | `blocks` | 2.1, 2.2, 2.4, 2.5 | By height, by hash, recent blocks (paginated) |
| **Transaction** | `transactions` | 2.1, 2.2, 2.4, 2.5 | By hash, by address (sent/received), recent (paginated) |
| **Log** | `logs` | 2.1 | By address + topics, by transaction hash |

**Note:** Epic 2 performs **read-only** operations. No writes to database.

### API Endpoints (External - RESTful + WebSocket)

#### REST Endpoints

| Method | Path | Handler | Purpose | Query Params | Response |
|--------|------|---------|---------|--------------|----------|
| GET | `/v1/blocks` | `handlers.ListBlocks` | Recent blocks | `limit`, `offset` | Array of block summaries |
| GET | `/v1/blocks/{height}` | `handlers.GetBlockByHeight` | Block by height | - | Block details + transactions |
| GET | `/v1/blocks/hash/{hash}` | `handlers.GetBlockByHash` | Block by hash | - | Block details + transactions |
| GET | `/v1/txs/{hash}` | `handlers.GetTransaction` | Transaction details | - | Transaction with block info |
| GET | `/v1/address/{addr}/txs` | `handlers.GetAddressTransactions` | Address tx history | `limit`, `offset` | Array of transactions |
| GET | `/v1/logs` | `handlers.GetLogs` | Event logs | `address`, `topic0`, `topic1`, `limit`, `offset` | Array of logs |
| GET | `/v1/stats/chain` | `handlers.GetChainStats` | Chain statistics | - | Latest block, count, lag |
| GET | `/health` | `handlers.HealthCheck` | Health check | - | Status: healthy/degraded/unhealthy |
| GET | `/metrics` | `promhttp.Handler()` | Prometheus metrics | - | Prometheus text format |

#### WebSocket Endpoint

| Path | Protocol | Channels | Message Format | Purpose |
|------|----------|----------|----------------|---------|
| `/v1/stream` | WebSocket | `newBlocks`, `newTxs` | JSON | Real-time block/tx broadcasts |

**Subscription Message:**
```json
{"action": "subscribe", "channel": "newBlocks"}
```

**Block Update Message:**
```json
{
  "channel": "newBlocks",
  "data": {
    "height": 5000,
    "hash": "0x...",
    "timestamp": 1698765432,
    "tx_count": 42
  }
}
```

### External Integration Points

| Integration | Type | Direction | Configuration | Notes |
|-------------|------|-----------|---------------|-------|
| **PostgreSQL 16** | SQL via pgx driver | Outbound | Same as Epic 1 | Read-only queries with indexes |
| **Browser Clients** | HTTP/WebSocket | Inbound | `API_PORT=8080` | CORS enabled for local development |
| **Prometheus (Optional)** | HTTP (scrape) | Inbound | `/metrics` endpoint | Pull-based metrics collection |

### Implementation Sequence

**Must implement in order:**

1. **Story 2.1 (REST API)** - Foundation for Epic 2
   - Implements chi router, middleware, handlers
   - Queries database populated by Epic 1

2. **Story 2.3 (Pagination)** - Part of 2.1
   - Helper functions for pagination logic
   - Integrated into list endpoints

3. **Story 2.6 (Health & Metrics)** - Extends 2.1
   - Adds health check and metrics endpoints
   - Reuses Epic 1 metrics registry

4. **Story 2.2 (WebSocket)** - Depends on 2.1
   - Adds WebSocket to existing HTTP server
   - Broadcasts updates from live-tail process

5. **Story 2.4 (Frontend)** - Depends on 2.1, 2.2
   - Consumes REST endpoints
   - Connects to WebSocket stream

6. **Story 2.5 (Search UI)** - Depends on 2.4
   - Enhances frontend with search capability
   - Uses existing API endpoints

### Dependencies on Other Epics

**Epic 2 depends on Epic 1:**
- Requires database to be populated with blocks/transactions/logs
- Reads data written by Epic 1 indexer
- WebSocket streams updates as Epic 1 processes new blocks
- Cannot be implemented until Epic 1 has working backfill

**Epic 1 does NOT depend on Epic 2:**
- Indexer runs independently
- Can be tested without API layer

### Testing Strategy

| Story | Test Type | Test Location | Coverage Target |
|-------|-----------|---------------|-----------------|
| 2.1 REST API | Integration | `internal/api/handlers_test.go` | All endpoints with mock data |
| 2.1 REST API | Unit | `internal/api/server_test.go` | Router setup, middleware |
| 2.2 WebSocket | Integration | `internal/api/websocket_test.go` | Connection, subscription, broadcast |
| 2.3 Pagination | Unit | `internal/api/pagination_test.go` | Edge cases (offset, limits) |
| 2.4 Frontend | Manual | Browser | Visual inspection, WebSocket connection |
| 2.5 Search UI | Manual | Browser | Search functionality, error handling |
| 2.6 Health | Integration | `internal/api/handlers_test.go` | Health check scenarios |

### Operational Metrics (Prometheus)

| Metric Name | Type | Labels | Purpose |
|-------------|------|--------|---------|
| `explorer_api_requests_total` | Counter | `method`, `endpoint`, `status` | Request counts by endpoint |
| `explorer_api_latency_ms` | Histogram | `method`, `endpoint` | Response time distribution |
| `explorer_api_websocket_connections` | Gauge | - | Active WebSocket connections |

### Configuration (Environment Variables)

```bash
# API Configuration
API_PORT=8080
API_CORS_ORIGINS=*  # For demo; restrict in production

# Metrics (shared with Epic 1)
METRICS_ENABLED=true
METRICS_PORT=9090  # Separate port for metrics (optional)
```

### Frontend Architecture

**Technology:** Vanilla HTML/JavaScript (no build step)

**File Structure:**
```
web/
├── index.html    # Main page with search, live ticker, recent txs table
├── style.css     # Minimal styling
└── app.js        # WebSocket client, API calls, DOM manipulation
```

**UI Components:**

1. **Header** - Logo, title, search bar
2. **Live Blocks Ticker** - 10 most recent blocks, updates via WebSocket
3. **Recent Transactions Table** - 25 most recent transactions, paginated
4. **Search Results Panel** - Displays block/transaction/address details based on search
5. **Footer** - GitHub link, stats

**Interaction Flow:**

```
User loads page → Frontend connects to WebSocket (/v1/stream)
                → Frontend fetches recent blocks (GET /v1/blocks?limit=10)
                → Frontend fetches recent txs (GET /v1/blocks?limit=25)

New block indexed → Live-tail broadcasts to WebSocket
                 → Frontend receives update
                 → Frontend updates ticker (prepend new block)

User searches "0x123..." → Frontend detects tx hash format
                        → Frontend calls GET /v1/txs/0x123...
                        → Frontend displays transaction details
```

### Readiness Status

✅ **READY FOR IMPLEMENTATION**

- All API endpoints specified with request/response formats
- WebSocket protocol defined with message schemas
- Frontend structure and interactions documented
- Testing strategy defined
- Depends on Epic 1 completion (data must exist to query)

---

## Cross-Epic Integration

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         Epic 1: Indexer Worker                   │
│  RPC Client → Ingestion → Indexing → Storage (PostgreSQL)       │
│                                        ↓                         │
│                              (Writes blocks, txs, logs)          │
└─────────────────────────────────────────────────────────────────┘
                                        ↓
                              PostgreSQL Database
                                (shared resource)
                                        ↓
┌─────────────────────────────────────────────────────────────────┐
│                        Epic 2: API Server                        │
│  Storage (PostgreSQL) → API Handlers → HTTP Response            │
│                      ↘ WebSocket Hub → Browser Clients          │
└─────────────────────────────────────────────────────────────────┘
```

### Communication Patterns

| From | To | Method | Data Format | Frequency |
|------|----|----|-------------|-----------|
| Epic 1 Indexer | PostgreSQL | SQL INSERT | Blocks, Txs, Logs | Per block (5-15s intervals) |
| PostgreSQL | Epic 2 API | SQL SELECT | Query results | Per API request |
| Epic 1 Live-Tail | Epic 2 WebSocket | In-memory channel | Block updates | Per new block (~12s intervals) |
| Epic 2 API | Browser | HTTP JSON | API responses | Per user request |
| Epic 2 WebSocket | Browser | WebSocket JSON | Real-time updates | Per new block |

**Note:** Epic 1 and Epic 2 run as **separate processes** and communicate only via PostgreSQL and optional in-memory channels for WebSocket updates.

### Shared Resources

| Resource | Epic 1 Usage | Epic 2 Usage | Conflict Resolution |
|----------|--------------|--------------|---------------------|
| **PostgreSQL** | Write (INSERT, UPDATE) | Read (SELECT) | No conflict - Epic 2 is read-only |
| **Metrics Registry** | Indexer metrics | API metrics | Separate metric namespaces |
| **Log Stream** | Indexer logs | API logs | Separate log contexts with service labels |

### Process Architecture

```
Docker Compose
├── postgres (PostgreSQL 16)
│   └── Stores all indexed data
│
├── worker (Indexer Worker - Epic 1)
│   ├── Reads from: Ethereum RPC
│   ├── Writes to: PostgreSQL
│   └── Exposes: Metrics at :9091 (internal)
│
└── api (API Server - Epic 2)
    ├── Reads from: PostgreSQL
    ├── Exposes: HTTP API at :8080
    ├── Exposes: WebSocket at :8080/v1/stream
    └── Exposes: Metrics at :9090
```

### Deployment Sequence

1. **Start PostgreSQL** (`docker compose up postgres`)
2. **Start Worker** (`docker compose up worker`)
   - Runs migrations automatically
   - Begins backfill (5,000 blocks)
   - Switches to live-tail when caught up
3. **Start API Server** (`docker compose up api`)
   - Runs migrations (idempotent, no-op if already applied)
   - Begins serving API requests
   - WebSocket hub starts
4. **Open Frontend** (browser to `http://localhost:8080`)

**Dependencies:**
- API Server **must** start after Worker has indexed at least some data
- Frontend **must** connect after API Server is running
- PostgreSQL **must** be available before both Worker and API Server

---

## Implementation Readiness Summary

### Epic 1: Core Indexing & Data Pipeline

| Aspect | Status | Notes |
|--------|--------|-------|
| Architecture Components | ✅ Defined | All 5 layers specified |
| Data Models | ✅ Complete | SQL DDL for all tables |
| Indexes | ✅ Optimized | 8 indexes for performance |
| Integration Points | ✅ Documented | RPC, PostgreSQL |
| Configuration | ✅ Complete | All env vars specified |
| Testing Strategy | ✅ Defined | Unit + integration tests |
| Metrics | ✅ Defined | 5 Prometheus metrics |
| Implementation Sequence | ✅ Established | Stories 1.1 → 1.9 |

**Overall:** ✅ **READY**

### Epic 2: API Layer & User Interface

| Aspect | Status | Notes |
|--------|--------|-------|
| API Endpoints | ✅ Defined | 9 REST + 1 WebSocket |
| Data Models | ✅ Complete | Read-only access to Epic 1 tables |
| Frontend Architecture | ✅ Documented | Vanilla JS, no build |
| Integration Points | ✅ Documented | PostgreSQL, Browser |
| Configuration | ✅ Complete | API port, CORS |
| Testing Strategy | ✅ Defined | Integration + manual tests |
| Metrics | ✅ Defined | 3 API-specific metrics |
| Implementation Sequence | ✅ Established | Stories 2.1 → 2.6 |

**Overall:** ✅ **READY** (after Epic 1)

---

## Next Steps

1. ✅ **Begin Epic 1 Implementation** - Start with Story 1.1 (RPC Client) and 1.2 (Schema)
2. ⏭️ **Epic 1 Testing** - Write tests alongside implementation
3. ⏭️ **Epic 2 Implementation** - Start after Epic 1 backfill is working
4. ⏭️ **Integration Validation** - Verify full pipeline (RPC → DB → API → Frontend)
5. ⏭️ **Performance Testing** - Validate NFR001 (backfill speed) and NFR002 (API latency)

---

**Document Status:** ✅ Complete and validated against checklist
**Generated:** 2025-10-29
**Reviewed:** Winston (Architect Agent)
