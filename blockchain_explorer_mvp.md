# Blockchain Explorer MVP (1-week Golang Project)

## 🎯 Goal
Build a **lean Ethereum-like blockchain explorer** to demonstrate **Golang and data engineering** skills within 1 week.

---

## 🧩 Core Objectives
- Backfill last N blocks (default: 5,000)
- Live-tail new blocks and transactions
- Index blocks, txs, and logs into PostgreSQL
- Provide REST + WebSocket APIs
- Minimal single-page frontend (SPA)
- Docker Compose setup for quick launch

Target chain: **Ethereum Sepolia testnet**

---

## 🚀 Deliverables

### ✅ Must-have Features
1. **Indexer**
   - Backfill last `N` blocks (parallel workers)
   - Live tail via WS/RPC
   - Detect and recover from reorgs (≤6 blocks deep)
2. **Database Schema (PostgreSQL)**
   - Blocks, transactions, logs
   - Composite indexes for fast queries
3. **API Endpoints**
   - `GET /v1/blocks`, `GET /v1/txs/{hash}`, `GET /v1/address/{addr}/txs`
   - `GET /v1/logs`, `GET /v1/stats/chain`
   - `WS /v1/stream` for live updates
4. **Frontend (Minimal)**
   - Live blocks ticker
   - Latest transactions table
   - Basic search (tx, block, address)
5. **Operations**
   - Docker Compose (Postgres + Go services)
   - Prometheus metrics `/metrics`
   - Structured logging, Makefile, migrations

---

## 📁 Folder Structure
```
cmd/
  api/         # HTTP+WS server
  worker/      # Indexer + ingestor
internal/
  rpc/         # JSON-RPC client with retry/backoff
  ingest/      # Fetch & decode blocks
  index/       # Normalize, persist, handle reorgs
  store/pg/    # PostgreSQL store + migrations
  api/         # REST/WS handlers
  util/        # Logger, metrics, worker pools
web/
  index.html   # Minimal SPA
migrations/
docker/
Makefile
```

---

## 🗓️ Day-by-Day Plan

| Day | Focus | Deliverables |
|-----|--------|--------------|
| 1 | Skeleton + RPC client | Compose, Postgres, migrations |
| 2 | Backfill pipeline | Worker pool + bulk inserts |
| 3 | Live tail + reorg | Head tracker + orphan marking |
| 4 | REST + WS APIs | Endpoints + Prometheus metrics |
| 5 | UI | Minimal SPA (HTML/JS) |
| 6 | Tests + perf | Unit & integration tests |
| 7 | Polish | Docs, screenshots, final tuning |

---

## ⚙️ Data Model

### Blocks
| Field | Type | Notes |
|--------|------|-------|
| height | bigint | Primary key |
| hash | bytea | Unique |
| parent_hash | bytea | For reorgs |
| miner | bytea | Miner/validator address |
| gas_used | numeric | Usage metric |
| tx_count | int | Transaction count |
| orphaned | bool | For invalidated blocks |

### Transactions
| Field | Type | Notes |
|--------|------|-------|
| hash | bytea | Primary key |
| block_height | bigint | FK to blocks |
| from_addr / to_addr | bytea | Participants |
| value_wei / fee_wei | numeric | Transaction data |
| success | bool | Status |

### Logs
| Field | Type |
|--------|------|
| tx_hash | bytea |
| log_index | int |
| address | bytea |
| topic0–3 | bytea |
| data | bytea |

---

## 🌐 API Examples
```
GET /v1/blocks?limit=25
GET /v1/txs/{hash}
GET /v1/address/{addr}/txs?limit=50
GET /v1/logs?address=0x...
GET /v1/stats/chain
WS /v1/stream (channels: newBlocks, newTxs)
```

---

## 📊 Metrics (Prometheus)
| Metric | Description |
|--------|-------------|
| explorer_blocks_indexed_total | Blocks processed |
| explorer_index_lag_blocks | Head lag |
| explorer_rpc_errors_total | Node connection issues |
| explorer_api_latency_ms | API performance |

---

## 🧠 Demo Script
1. Run `docker compose up`
2. Watch indexer backfill logs
3. Open SPA – live blocks appear
4. Query tx hash → detail view
5. View `/metrics` → operational insight

---

## 🧩 Acceptance Criteria
- Backfill 5,000 blocks in minutes
- Live tail within 2s of network head
- Reorgs heal automatically
- p95 latency < 150ms for API
- Clear README + reproducible build

---

## 🧱 Stretch Goals
- ERC-20 Transfer decoding
- ClickHouse analytics
- Address labeling
- API key management
- Rate limiting

---

## 🧰 Tech Stack
- **Go 1.22+**
- **PostgreSQL 16**
- **pgx, chi, prometheus/client_golang**
- **Docker Compose**
- **HTML/JS SPA (no framework)**

---

## 📄 Documentation
- `README.md` – setup & run guide
- `API.md` – endpoint examples
- `Design.md` – architecture, reorg handling
