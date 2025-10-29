# Blockchain Explorer - Epic Breakdown

**Author:** Hieu
**Date:** 2025-10-29
**Project Level:** Level 2 (Small complete system)
**Target Scale:** 12-15 stories across 2 epics

---

## Epic Overview

This project is organized into 2 epics that map to the 7-day implementation schedule:

1. **Epic 1: Core Indexing & Data Pipeline** (7-9 stories, Days 1-3) - Build the foundational blockchain data ingestion system
2. **Epic 2: API Layer & User Interface** (5-6 stories, Days 4-5) - Implement data access APIs and demonstration frontend

Operational concerns (testing, documentation, deployment) are integrated across both epics rather than separated.

---

## Epic Details

## Epic 1: Core Indexing & Data Pipeline

**Goal:** Build a production-grade blockchain data pipeline that efficiently indexes Ethereum blocks, handles chain reorganizations, and provides operational visibility.

**Success Criteria:**
- Successfully backfills 5,000 blocks in under 5 minutes
- Live-tail maintains <2 second lag from network head
- Automatic reorg detection and recovery for forks up to 6 blocks deep
- Prometheus metrics accurately reflect system state
- System runs continuously for 24+ hours without issues

### Story 1.1: Ethereum RPC Client with Retry Logic

**Description:** Implement RPC client layer for communicating with Ethereum nodes (Sepolia testnet) with robust error handling and retry logic.

**Key Capabilities:**
- Connect to Ethereum RPC endpoints (support Alchemy, Infura, public nodes via configuration)
- Implement exponential backoff for transient failures
- Classify errors (transient network issues vs permanent failures)
- Connection pooling and rate limit awareness
- Logging for RPC errors and performance

**Acceptance Criteria:**
- Can successfully fetch blocks via `eth_getBlockByNumber`
- Can fetch transactions via `eth_getTransactionReceipt`
- Transient failures trigger retry with exponential backoff (max 5 retries)
- Permanent failures (invalid parameters) fail immediately
- RPC errors are logged with structured context

**Technical Notes:**
- Use go-ethereum client library
- Configure RPC endpoint via environment variable
- Implement connection timeout (10s) and request timeout (30s)

---

### Story 1.2: PostgreSQL Schema and Migrations

**Description:** Design and implement PostgreSQL schema optimized for blockchain data access patterns with migration system.

**Key Tables:**
- **blocks**: height (PK), hash, parent_hash, miner, gas_used, tx_count, timestamp, orphaned flag
- **transactions**: hash (PK), block_height (FK), from_addr, to_addr, value_wei, fee_wei, gas_used, success
- **logs**: tx_hash (FK), log_index, address, topic0, topic1, topic2, topic3, data

**Indexes:**
- Composite index on transactions (block_height, from_addr)
- Composite index on transactions (block_height, to_addr)
- Index on blocks (orphaned, height) for reorg queries
- Index on logs (address, topic0) for event filtering

**Acceptance Criteria:**
- Schema supports all data elements from Ethereum blocks/transactions/logs
- Composite indexes enable fast address transaction lookups (<150ms)
- Migration system (golang-migrate or embedded) allows schema versioning
- Foreign key constraints ensure referential integrity
- Orphaned flag enables soft-delete pattern for reorgs

**Technical Notes:**
- Use pgx native types (bytea for hashes/addresses, numeric for wei values)
- Design for 5K-50K blocks initially, scalable to millions
- Include created_at/updated_at timestamps for debugging

---

### Story 1.3: Parallel Backfill Worker Pool

**Description:** Implement parallel worker pool pattern to backfill historical blocks efficiently using bulk inserts.

**Key Capabilities:**
- Configurable worker pool size (default: 8 workers)
- Fetch blocks in parallel from RPC node
- Batch database inserts (e.g., 100 blocks per transaction)
- Progress tracking and logging
- Graceful shutdown on error or completion

**Acceptance Criteria:**
- Can backfill 5,000 blocks in under 5 minutes on standard hardware
- Workers process blocks in parallel without database contention
- Bulk inserts optimize database write performance
- Progress logged every 500 blocks (e.g., "Backfilled 500/5000 blocks (10%)")
- Worker pool size configurable via environment variable

**Technical Notes:**
- Use Go worker pool pattern with channels
- Implement bounded concurrency to avoid overwhelming RPC provider
- Transaction per batch for atomicity
- Handle "already exists" gracefully (idempotent operation)

---

### Story 1.4: Live-Tail Mechanism for New Blocks

**Description:** Implement continuous monitoring of blockchain head to index new blocks in real-time as they are produced.

**Key Capabilities:**
- Subscribe to new block headers via WebSocket or polling
- Fetch and index new blocks within 2 seconds of network publication
- Maintain sequential block processing for consistency
- Recover gracefully from missed blocks (gap detection)
- Coordinate with backfill process during initial sync

**Acceptance Criteria:**
- Detects new blocks within 2 seconds of network head
- Indexes blocks sequentially to maintain parent-child relationships
- If block N+1 arrives before block N, waits and retries N
- Logs current head and lag metrics (e.g., "Head: 5000, Lag: 1 block, 0.8s")
- Runs continuously without crashing or memory leaks

**Technical Notes:**
- Use go-ethereum subscription or polling every 5-10 seconds
- Single goroutine for live-tail to maintain ordering
- Check parent_hash matches DB head before inserting
- If parent mismatch detected, trigger reorg handling

---

### Story 1.5: Chain Reorganization Detection and Recovery

**Description:** Automatically detect chain reorganizations and heal the database by marking orphaned blocks and re-indexing the canonical chain.

**Key Capabilities:**
- Detect reorg when new block's parent_hash doesn't match DB head
- Walk backwards to find common ancestor (fork point)
- Mark orphaned blocks (set orphaned=true, don't delete)
- Re-process canonical chain from fork point forward
- Handle reorgs up to 6 blocks deep

**Acceptance Criteria:**
- When reorg detected, system automatically heals without manual intervention
- Orphaned blocks are marked (orphaned=true) rather than deleted
- Canonical chain is correctly indexed from fork point
- Reorg events logged with details (depth, fork point, new head)
- System handles reorgs up to 6 blocks without issues

**Technical Notes:**
- Walk parent chain backwards comparing hashes to DB
- Use database transaction for marking orphaned + inserting new blocks
- Log reorg depth and impacted block range
- Consider implementing reorg counter metric

---

### Story 1.6: Database Migration System

**Description:** Implement database schema versioning and migration management for reproducible database setup.

**Key Capabilities:**
- Migrations define schema changes (create tables, add indexes)
- Migrations run automatically on service startup
- Migration version tracking (which migrations have been applied)
- Idempotent migrations (safe to run multiple times)

**Acceptance Criteria:**
- Initial migration creates tables and indexes
- Migrations run automatically when starting indexer or API service
- Migration status visible in logs ("Applied migration 001_initial_schema")
- Database tracks which migrations have run (schema_migrations table)
- Can add new migrations for future schema evolution

**Technical Notes:**
- Use golang-migrate library or embed migrations in Go
- Migration files numbered sequentially (001_initial.up.sql, 001_initial.down.sql)
- Connection string configurable via environment variable

---

### Story 1.7: Prometheus Metrics for Indexer

**Description:** Instrument indexer with Prometheus metrics to provide operational visibility into performance and health.

**Key Metrics:**
- `explorer_blocks_indexed_total` (counter) - Total blocks processed
- `explorer_index_lag_blocks` (gauge) - How many blocks behind network head
- `explorer_index_lag_seconds` (gauge) - Time lag from network head
- `explorer_rpc_errors_total` (counter) - RPC connection/request failures
- `explorer_backfill_duration_seconds` (histogram) - Time to backfill N blocks

**Acceptance Criteria:**
- Metrics exposed at `/metrics` endpoint (standard Prometheus format)
- Metrics accurately reflect system state in real-time
- Lag metrics updated every 30 seconds during live-tail
- RPC error counter increments on failures
- Metrics can be scraped by Prometheus

**Technical Notes:**
- Use prometheus/client_golang library
- Register metrics globally or via metrics registry
- Update lag gauge periodically in background goroutine
- Include histogram buckets appropriate for backfill times

---

### Story 1.8: Structured Logging for Debugging

**Description:** Implement structured JSON logging throughout indexer for debugging and operational visibility.

**Key Log Events:**
- Indexer startup (configuration loaded)
- Backfill progress (every N blocks)
- Live-tail block indexed
- Reorg detected and resolved
- RPC errors with context
- Database errors with context

**Acceptance Criteria:**
- All logs output as structured JSON with consistent fields
- Log level configurable (DEBUG, INFO, WARN, ERROR)
- Each log includes timestamp, level, message, and contextual fields
- Errors include stack traces or error chains when available
- Logs are machine-parseable for aggregation

**Technical Notes:**
- Use standard library log/slog (Go 1.22+)
- Log format: {"time":"...", "level":"INFO", "msg":"...", "block_height":5000}
- Include correlation IDs or request IDs where appropriate
- Avoid logging sensitive data (private keys, etc.)

---

### Story 1.9: Integration Tests for Indexer Pipeline

**Description:** Create integration tests that validate end-to-end indexer functionality including backfill, live-tail, and reorg handling.

**Test Scenarios:**
- Backfill small block range (e.g., 100 blocks) and verify data in DB
- Simulate reorg by inserting orphaned chain and validating recovery
- Test RPC retry logic with mock failing endpoint
- Verify metrics are updated correctly during indexing
- Test graceful shutdown of worker pool

**Acceptance Criteria:**
- Integration test suite covers critical paths (>70% coverage)
- Tests use test containers or in-memory DB for isolation
- Tests validate data correctness (block hashes match chain)
- Tests run in CI pipeline (or locally via Makefile)
- Tests complete in reasonable time (<5 minutes)

**Technical Notes:**
- Use Go testing package + testify for assertions
- Consider testcontainers-go for PostgreSQL test instances
- Mock RPC client for controlled test scenarios
- Tests should be reproducible and not depend on external services

---

## Epic 2: API Layer & User Interface

**Goal:** Provide RESTful and WebSocket APIs for querying indexed blockchain data, along with a minimal frontend for demonstration purposes.

**Success Criteria:**
- REST API endpoints return correct data with <150ms p95 latency
- WebSocket streaming delivers real-time updates to connected clients
- Frontend displays live blocks and allows transaction search
- API includes health checks and metrics exposure
- System is demo-ready with clear API examples

### Story 2.1: REST API Endpoints for Blockchain Queries

**Description:** Implement REST API endpoints for querying blocks, transactions, address history, logs, and chain statistics.

**Endpoints:**
- `GET /v1/blocks` - List recent blocks (paginated)
- `GET /v1/blocks/{height}` - Block details by height
- `GET /v1/blocks/{hash}` - Block details by hash
- `GET /v1/txs/{hash}` - Transaction details
- `GET /v1/address/{addr}/txs` - Transaction history for address (paginated)
- `GET /v1/logs` - Query logs with filters (address, topics)
- `GET /v1/stats/chain` - Chain statistics (head, indexed blocks, lag)
- `GET /metrics` - Prometheus metrics
- `GET /health` - Health check endpoint

**Acceptance Criteria:**
- All endpoints return correct data matching database state
- Pagination works correctly for large result sets (limit, offset)
- p95 latency <150ms for standard queries
- Errors return appropriate HTTP status codes (404, 400, 500)
- API responses follow consistent JSON structure

**Technical Notes:**
- Use chi router for HTTP handling
- Middleware for logging, metrics, CORS
- Input validation for parameters (address format, block height)
- Database queries use prepared statements with indexes

---

### Story 2.2: WebSocket Streaming for Real-Time Updates

**Description:** Implement WebSocket endpoint that streams real-time updates for new blocks and transactions to connected clients.

**Capabilities:**
- WebSocket endpoint at `/v1/stream`
- Clients can subscribe to channels: `newBlocks`, `newTxs`
- Broadcasts new blocks/transactions as they are indexed
- Handles multiple concurrent connections
- Graceful connection cleanup on disconnect

**Acceptance Criteria:**
- Clients can connect via WebSocket and subscribe to channels
- New blocks are broadcast to subscribed clients within 1 second of indexing
- Supports 10-20 concurrent connections without degradation
- Connections close gracefully on client disconnect
- WebSocket errors are logged appropriately

**Technical Notes:**
- Use gorilla/websocket or standard library WebSocket
- Implement pub/sub pattern with channels
- Broadcast from live-tail goroutine to WebSocket hub
- Handle JSON marshaling for wire format

---

### Story 2.3: Pagination Implementation for Large Result Sets

**Description:** Implement robust pagination for API endpoints that return potentially large result sets (blocks, transactions, address history).

**Capabilities:**
- Query parameters: `limit` (default 25, max 100), `offset`
- Response includes pagination metadata (total, limit, offset)
- Efficient database queries using LIMIT/OFFSET
- Optional cursor-based pagination for future optimization

**Acceptance Criteria:**
- Pagination parameters work correctly across all list endpoints
- Response includes total count, current page info
- Limit enforced (max 100 items per request)
- Offset-based pagination performs acceptably for typical use cases
- API documentation includes pagination examples

**Technical Notes:**
- Use SQL LIMIT/OFFSET for offset-based pagination
- Consider COUNT(*) query for total (may cache for performance)
- Validate limit/offset parameters (non-negative, within bounds)
- Document pagination in API spec

---

### Story 2.4: Minimal SPA Frontend with Live Blocks Ticker

**Description:** Create a simple single-page web application that displays live blockchain activity including recent blocks and a live-updating ticker.

**Features:**
- Live blocks ticker showing most recent 10 blocks (updates in real-time)
- Each block shows: height, hash (truncated), timestamp, tx count
- Real-time updates via WebSocket connection
- Minimal CSS styling (clean, readable)
- Served as static files from API server

**Acceptance Criteria:**
- Frontend loads and connects to WebSocket automatically
- Live ticker updates as new blocks are indexed
- UI is clean and functional (no need for polish)
- Works in modern browsers (Chrome, Firefox, Safari)
- Static files served from `/` route of API server

**Technical Notes:**
- Vanilla HTML/JavaScript (no framework needed)
- WebSocket client connects to `/v1/stream`
- Use CSS Grid or Flexbox for layout
- Minimal JavaScript for WebSocket handling and DOM updates

---

### Story 2.5: Transaction Search and Display Interface

**Description:** Add search functionality to frontend allowing users to search for transactions by hash, blocks by height/hash, and addresses for transaction history.

**Features:**
- Search input that accepts: transaction hash, block number, block hash, or address
- Search results display appropriate view:
  - Transaction: show details (from, to, value, block, status)
  - Block: show block details and transaction list
  - Address: show transaction history (paginated)
- Links between related entities (click block height → block detail)

**Acceptance Criteria:**
- Search correctly detects input type (tx hash, block, address)
- Results display relevant information clearly
- Pagination works for address transaction history
- Error handling for invalid inputs ("Transaction not found")
- Search is accessible via query parameter (shareable URLs)

**Technical Notes:**
- Input validation on client side (hex format, length checks)
- API calls to appropriate endpoints based on input type
- URL routing for shareable links (e.g., `/tx/{hash}`)
- Simple error messages for user feedback

---

### Story 2.6: Health Check and Metrics Exposure

**Description:** Implement health check endpoint and ensure Prometheus metrics are properly exposed for monitoring.

**Endpoints:**
- `GET /health` - Returns 200 OK if system is healthy, includes status details
- `GET /metrics` - Prometheus metrics endpoint (already implemented in Epic 1)

**Health Check Details:**
- Database connectivity (can query DB)
- Indexer status (last block indexed timestamp)
- RPC connectivity (last successful RPC call timestamp)
- Response format: JSON with status details

**Acceptance Criteria:**
- `/health` returns 200 when all systems operational
- `/health` returns 503 if critical systems unavailable (DB, RPC)
- Health check runs quickly (<1 second)
- Metrics endpoint returns Prometheus-formatted metrics
- Both endpoints documented in API specification

**Technical Notes:**
- Health check pings database with simple query
- Check indexer last_updated timestamp (stale if >5 minutes)
- Include version information in health response
- Metrics endpoint uses prometheus/client_golang handler

---

## Story Dependencies and Sequencing

### Day 1: Foundation
- Story 1.1 (RPC Client) - **MUST BE FIRST**
- Story 1.2 (Schema & Migrations) - **MUST BE FIRST**

### Day 2: Backfill
- Story 1.3 (Backfill Workers) - Depends on 1.1, 1.2
- Story 1.7 (Metrics) - Can be done in parallel with 1.3

### Day 3: Live-Tail & Reorg
- Story 1.4 (Live-Tail) - Depends on 1.1, 1.2, 1.3
- Story 1.5 (Reorg Handling) - Depends on 1.4
- Story 1.8 (Logging) - Can be done in parallel

### Day 4: API Layer
- Story 2.1 (REST API) - Depends on 1.2 (needs data)
- Story 2.6 (Health & Metrics) - Depends on 2.1

### Day 5: Frontend & WebSocket
- Story 2.2 (WebSocket) - Depends on 1.4, 2.1
- Story 2.3 (Pagination) - Part of 2.1
- Story 2.4 (Frontend) - Depends on 2.1, 2.2
- Story 2.5 (Search UI) - Depends on 2.4

### Day 6: Testing & Validation
- Story 1.6 (Migrations final polish)
- Story 1.9 (Integration Tests) - Depends on all Epic 1 stories
- Performance validation and optimization

### Day 7: Documentation & Polish
- README with setup instructions and demo script
- API documentation (API.md)
- Architecture documentation (Design.md)
- Final testing and bug fixes

---

## Notes

- **Story Points Not Assigned**: This is a solo 7-day sprint focused on completion, not relative estimation
- **Testing Integrated**: Testing is embedded throughout rather than deferred to end
- **Flexibility**: If ahead of schedule, consider stretch goals (ERC-20 decoding, better UI)
- **Risk**: Stories 1.4 and 1.5 (live-tail, reorg) identified as highest technical risk - allocate extra time if needed

---

_Detailed acceptance criteria and technical implementation details will be refined during solution architecture phase._
