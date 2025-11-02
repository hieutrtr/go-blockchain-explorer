# Complete Blockchain Explorer Workflow Guide

A comprehensive step-by-step guide to fetch blockchain data, extract it, start the API, and view results.

**Table of Contents:**
1. [Prerequisites & Setup](#prerequisites--setup)
2. [Step 1: Configure RPC Endpoint](#step-1-configure-rpc-endpoint)
3. [Step 2: Setup Database](#step-2-setup-database)
4. [Step 3: Run Worker to Fetch & Extract Data](#step-3-run-worker-to-fetch--extract-data)
5. [Step 4: Start API Server](#step-4-start-api-server)
6. [Step 5: Access & Verify Data](#step-5-access--verify-data)
7. [Monitoring & Troubleshooting](#monitoring--troubleshooting)
8. [Complete Workflow Examples](#complete-workflow-examples)

---

## Prerequisites & Setup

### System Requirements

- **Go**: 1.24+ ([Install](https://golang.org/doc/install))
- **Docker & Docker Compose**: ([Install](https://docs.docker.com/get-docker/))
- **Ethereum RPC Endpoint**: Free API key from:
  - ✅ **Alchemy** ([Sign up](https://www.alchemy.com/)) - Recommended (fast, 300 req/sec)
  - ✅ **Infura** ([Sign up](https://infura.io/)) - Fast (100 req/sec)
  - ⚠️ **Public node** (rpc.sepolia.org) - Slow, rate-limited

### Project Setup

```bash
# Clone the repository
git clone https://github.com/hieutt50/go-blockchain-explorer.git
cd go-blockchain-explorer

# Download Go dependencies
go mod download

# Create environment file from template
cp .env.example .env
```

---

## Step 1: Configure RPC Endpoint

### Why This Matters

The RPC endpoint is how your worker fetches blockchain data. Using a **fast, reliable RPC provider** is critical for:
- ✅ Avoiding timeout errors
- ✅ Indexing blocks quickly
- ✅ Handling many concurrent requests

### Option A: Use Alchemy (Fastest - Recommended)

1. **Get Free API Key:**
   - Go to https://www.alchemy.com/
   - Sign up with email
   - Create new app for "Ethereum Sepolia" testnet
   - Copy your API key

2. **Update `.env` file:**
   ```bash
   # Edit .env with your favorite editor
   nano .env
   ```

   Find and update this line:
   ```bash
   # CHANGE THIS:
   RPC_URL=https://rpc.sepolia.org

   # TO THIS (replace YOUR_API_KEY):
   RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
   ```

3. **Verify connection:**
   ```bash
   curl -X POST -H "Content-Type: application/json" \
     --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
     https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
   ```

   Should return: `{"jsonrpc":"2.0","result":"0x6a0123","id":1}` (hex block number)

### Option B: Use Infura

1. **Get Free API Key:**
   - Go to https://infura.io/
   - Sign up with email
   - Create new project for "Sepolia"
   - Copy your project ID

2. **Update `.env` file:**
   ```bash
   RPC_URL=https://sepolia.infura.io/v3/YOUR_PROJECT_ID
   ```

### Benchmark Comparison

| Provider | Speed | Rate Limit | Cost | Timeout Issues |
|----------|-------|-----------|------|----------------|
| **Alchemy** | ⚡ 50-100ms | 300 req/sec | Free | ❌ Rare |
| **Infura** | ⚡ 100-150ms | 100 req/sec | Free | ❌ Rare |
| **rpc.sepolia.org** | 🐢 500-2000ms | 1 req/sec | Free | ✅ Common |

---

## Step 2: Setup Database

### What This Does

```
┌──────────────────────────────────────────┐
│ make db-setup runs:                      │
├──────────────────────────────────────────┤
│ 1. Start PostgreSQL in Docker            │
│ 2. Wait for database to be ready         │
│ 3. Create blockchain_explorer database   │
│ 4. Run all migrations                    │
│    - Create blocks table                 │
│    - Create transactions table           │
│    - Create logs table                   │
│    - Add indexes for fast queries        │
└──────────────────────────────────────────┘
```

### Run Setup (One Command!)

```bash
make db-setup
```

**Expected output:**
```
Setting up database...
Starting PostgreSQL with Docker...
Waiting for PostgreSQL to be ready...
PostgreSQL is already running
Running database migrations...
Using Docker container...
Applying migration: migrations/000001_initial_schema.up.sql
Applying migration: migrations/000002_add_indexes.up.sql
✓ Migrations complete
✓ Database setup complete
```

### Verify Database is Ready

```bash
# Check PostgreSQL container is running
docker ps | grep postgres

# Check database is accepting connections
make db-status

# Expected output:
# ✓ PostgreSQL Docker container is running
# ✓ Database is accepting connections
```

### Database Schema

```sql
-- Blocks table
CREATE TABLE blocks (
    id SERIAL PRIMARY KEY,
    height BIGINT UNIQUE,
    hash VARCHAR(66),
    parent_hash VARCHAR(66),
    timestamp BIGINT,
    gas_used BIGINT,
    gas_limit BIGINT,
    tx_count INT,
    miner VARCHAR(42),
    orphaned BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Transactions table
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(66) UNIQUE,
    block_height BIGINT,
    from_addr VARCHAR(42),
    to_addr VARCHAR(42),  -- NULL for contract creation
    value NUMERIC,
    gas_price NUMERIC,
    gas BIGINT,
    gas_used BIGINT,
    nonce BIGINT,
    data TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (block_height) REFERENCES blocks(height)
);

-- Event Logs table
CREATE TABLE logs (
    id SERIAL PRIMARY KEY,
    tx_hash VARCHAR(66),
    block_height BIGINT,
    address VARCHAR(42),
    topic0 VARCHAR(66),
    topic1 VARCHAR(66),
    topic2 VARCHAR(66),
    topic3 VARCHAR(66),
    data TEXT,
    log_index INT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Step 3: Run Worker to Fetch & Extract Data

### What the Worker Does

The **worker** is responsible for:

1. **Fetches blocks** from Ethereum via RPC
2. **Extracts transactions** from each block
3. **Recovers sender address** using ECDSA signature recovery
4. **Calculates fees** (gas_used × gas_price)
5. **Stores everything** in PostgreSQL atomically
6. **Monitors** for new blocks in real-time (live-tail)

### Configuration

Edit `.env` to control what data gets indexed:

```bash
# Which blocks to backfill (fetch historical data)
BACKFILL_START_HEIGHT=0              # Start from genesis (0) or any block
BACKFILL_END_HEIGHT=100              # How many blocks to fetch
BACKFILL_BATCH_SIZE=10               # Fetch 10 blocks at a time
BACKFILL_CONCURRENCY=4               # Fetch 4 batches in parallel

# Real-time indexing (after backfill completes)
LIVETAIL_ENABLED=true                # Enable real-time monitoring
LIVETAIL_START_FROM_TIP=true          # Start from latest block

# Performance tuning
DB_MAX_CONNS=20                      # Database connections
RPC_TIMEOUT=10s                      # RPC request timeout
```

### Examples

#### Example 1: Index Blocks 0-1000

```bash
# Update .env
BACKFILL_START_HEIGHT=0
BACKFILL_END_HEIGHT=1000

# Run worker
make run-worker

# Expected output:
# Starting backfill phase
# Fetching blocks: [==========] 1000/1000
# Extracted 50000 transactions
# Backfill completed in 30 seconds (33 blocks/sec)
# Starting live-tail phase
```

**Time estimate:** ~30 seconds for 1000 blocks (with Alchemy)

#### Example 2: Continue from Latest Block

```bash
# Update .env to index NEXT 1000 blocks
BACKFILL_START_HEIGHT=1001
BACKFILL_END_HEIGHT=2000

# Run worker
make run-worker

# Worker automatically continues from where it left off
```

#### Example 3: Index Specific Recent Blocks

```bash
# Update .env to index blocks 7000010-7001000
BACKFILL_START_HEIGHT=7000010
BACKFILL_END_HEIGHT=7001000

# Run worker
make run-worker

# It will fetch the most recent blocks
```

### Run Worker

```bash
# Start worker in foreground (see logs in real-time)
make run-worker

# OR start in background with logging
nohup make run-worker > logs/worker.log 2>&1 &

# OR use the startup script (both API and worker)
make run
```

### Monitor Worker Progress

**In a separate terminal:**

```bash
# Watch worker logs in real-time
tail -f logs/worker.log

# Check blocks indexed so far
docker exec blockchain-explorer-db psql -U postgres -d blockchain_explorer -c \
  "SELECT COUNT(*) as blocks_indexed FROM blocks;"

# Check transactions extracted
docker exec blockchain-explorer-db psql -U postgres -d blockchain_explorer -c \
  "SELECT COUNT(*) as transactions FROM transactions;"

# Check worker metrics
curl http://localhost:9090/metrics | grep blockchain_

# Expected metrics:
# blockchain_blocks_indexed 1000
# blockchain_transactions_extracted 50000
```

### Worker Lifecycle

```
START
  ├─ Check database
  │  └─ If empty: Start full backfill
  │  └─ If has blocks: Continue from latest
  │
  ├─ BACKFILL PHASE
  │  ├─ Fetch blocks in parallel (BACKFILL_CONCURRENCY batches)
  │  ├─ Extract transactions from each block
  │  ├─ Recover sender address (ECDSA)
  │  ├─ Calculate transaction fees
  │  └─ Store atomically in database
  │
  ├─ LIVE-TAIL PHASE
  │  ├─ Poll RPC every LIVETAIL_POLL_INTERVAL (default 12s)
  │  ├─ Detect new blocks
  │  ├─ Detect chain reorganizations
  │  └─ Continue indexing in real-time
  │
  └─ Receive SIGTERM/SIGINT
     └─ Graceful shutdown (cleanup + close connections)
```

---

## Step 4: Start API Server

### What the API Server Does

The **API server** provides:
- 🌐 REST API endpoints for querying blockchain data
- 📊 WebSocket streaming for real-time updates
- 📈 Prometheus metrics for monitoring
- 🖥️ Web UI (single-page app) served at `/`

### Start API Server

**Option 1: Run API only (for development)**
```bash
make run-api

# Expected output:
# API server listening at http://localhost:8080
# WebSocket hub initialized
```

**Option 2: Run both API and worker together**
```bash
make run

# This starts:
# - API Server on port 8080
# - Worker (indexer) in background
# - Logs saved to logs/api.log and logs/worker.log
```

**Option 3: Run as background services**
```bash
# Start API
nohup make run-api > logs/api.log 2>&1 &

# In separate terminal, start worker
nohup make run-worker > logs/worker.log 2>&1 &

# View logs
tail -f logs/api.log
tail -f logs/worker.log
```

### Verify API is Running

```bash
# Test health endpoint
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","message":"API server is running"}

# Get latest blocks
curl http://localhost:8080/v1/blocks?limit=5

# Expected response: Array of 5 most recent blocks
```

---

## Step 5: Access & Verify Data

### 5.1 Web UI (Recommended for Most Users)

**Open in browser:**
```
http://localhost:8080/
```

You'll see:
- 📊 **Live Blocks Ticker** - Most recent blocks updating in real-time
- 📋 **Recent Transactions** - Latest transactions with details
- 🟢 **Connection Status** - WebSocket connection indicator
- ⚡ **Real-time Updates** - New blocks appear automatically

### 5.2 REST API Endpoints

#### Get Latest Blocks

```bash
curl "http://localhost:8080/v1/blocks?limit=10&offset=0"
```

**Response:**
```json
{
  "data": [
    {
      "height": 7001000,
      "hash": "0x1234...5678",
      "parent_hash": "0xabcd...ef01",
      "timestamp": 1698765432,
      "gas_used": 15234567,
      "gas_limit": 30000000,
      "tx_count": 142,
      "miner": "0x742d...bEb0",
      "orphaned": false
    },
    ...
  ],
  "total": 1000,
  "limit": 10,
  "offset": 0
}
```

#### Get Block by Height

```bash
curl "http://localhost:8080/v1/blocks/7001000"
```

#### Get Transactions from Block

```bash
curl "http://localhost:8080/v1/blocks/7001000/txs?limit=50"
```

#### Get Transaction by Hash

```bash
curl "http://localhost:8080/v1/txs/0x1234567890abcdef"
```

#### Get Address Transaction History

```bash
curl "http://localhost:8080/v1/address/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0/txs?limit=50"
```

#### Get Event Logs

```bash
curl "http://localhost:8080/v1/logs?address=0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0&limit=100"
```

#### Get Chain Statistics

```bash
curl "http://localhost:8080/v1/stats/chain"
```

**Response:**
```json
{
  "latest_block": 7001000,
  "total_blocks": 1000,
  "total_transactions": 50000,
  "indexer_lag_blocks": 2,
  "indexer_lag_seconds": 24,
  "last_updated": "2025-10-31T10:30:00Z"
}
```

### 5.3 WebSocket Streaming (Real-time Updates)

**Connect to WebSocket:**
```javascript
const ws = new WebSocket('ws://localhost:8080/v1/ws');

// Subscribe to new blocks
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'blocks'
}));

// Handle incoming blocks
ws.onmessage = (event) => {
  const block = JSON.parse(event.data);
  console.log('New block received:', block);
  // block = {
  //   height: 7001001,
  //   hash: "0x...",
  //   tx_count: 145,
  //   timestamp: 1698765444,
  //   ...
  // }
};
```

### 5.4 Database Queries

```bash
# Open database shell
make db-shell

# Or use Docker directly
docker exec -it blockchain-explorer-db psql -U postgres -d blockchain_explorer
```

**Useful queries:**

```sql
-- Count blocks indexed
SELECT COUNT(*) as total_blocks FROM blocks;

-- Get latest 10 blocks
SELECT height, hash, tx_count, gas_used FROM blocks
ORDER BY height DESC LIMIT 10;

-- Count total transactions
SELECT COUNT(*) as total_transactions FROM transactions;

-- Find transactions for an address
SELECT * FROM transactions
WHERE from_addr = '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0'
LIMIT 20;

-- High-value transactions (> 1 ETH)
SELECT * FROM transactions
WHERE value > 1000000000000000000
ORDER BY value DESC LIMIT 20;

-- Database size
SELECT pg_size_pretty(pg_database_size('blockchain_explorer'));
```

### 5.5 Prometheus Metrics

```bash
# View all metrics
curl http://localhost:9090/metrics

# Filter blockchain metrics
curl http://localhost:9090/metrics | grep blockchain_

# Common metrics:
# blockchain_blocks_indexed - Total blocks indexed
# blockchain_transactions_extracted - Total transactions found
# blockchain_indexing_lag_seconds - How far behind the tip
# blockchain_rpc_requests_total - Total RPC calls
# blockchain_rpc_errors_total - Total RPC errors
```

### 5.6 pgAdmin Web Interface (Optional)

**Access database GUI:**
```
http://localhost:5050
```

**Login:**
- Email: `admin@blockchain-explorer.local`
- Password: `admin`

**Connect to database:**
1. Right-click "Servers" → Register → Server
2. General tab: Name = "Blockchain Explorer"
3. Connection tab:
   - Host: `postgres`
   - Port: `5432`
   - Database: `blockchain_explorer`
   - Username: `postgres`
   - Password: `postgres`

Then browse tables and run SQL queries through the GUI.

---

## Monitoring & Troubleshooting

### Common Issues & Solutions

#### ❌ Worker Times Out: "context deadline exceeded"

**Problem:** RPC endpoint is too slow or rate-limited

**Solution:**
```bash
# 1. Check your RPC_URL in .env
cat .env | grep RPC_URL

# 2. If using public node (rpc.sepolia.org), upgrade to Alchemy:
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# 3. Verify connection works:
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  $RPC_URL
```

#### ❌ Database Connection Refused

**Problem:** PostgreSQL container not running

**Solution:**
```bash
# Check container status
docker ps | grep postgres

# If not running, restart it
docker-compose restart postgres

# Or recreate everything
make db-drop
make db-setup
```

#### ❌ Port 8080 Already in Use

**Problem:** Another service using port 8080

**Solution:**
```bash
# Find what's using the port
lsof -i :8080

# Option 1: Kill that process
kill -9 <PID>

# Option 2: Use different port
# Edit .env and change:
API_PORT=8081
```

#### ❌ Migrations Failed

**Problem:** Database schema not created

**Solution:**
```bash
# Check database status
make db-status

# If database doesn't exist
make db-create

# Run migrations manually
make migrate

# Or fresh start
make db-drop
make db-setup
```

### Monitoring Commands

```bash
# Real-time logs
tail -f logs/worker.log
tail -f logs/api.log

# Monitor database growth
watch -n 1 "docker exec blockchain-explorer-db psql -U postgres -d blockchain_explorer -c 'SELECT COUNT(*) as blocks, (SELECT COUNT(*) FROM transactions) as txs FROM blocks;'"

# Monitor worker metrics
watch -n 5 "curl -s http://localhost:9090/metrics | grep blockchain_"

# Check system resources
docker stats blockchain-explorer-db

# View all active processes
ps aux | grep -E "(api|worker)" | grep -v grep
```

### Performance Optimization

```bash
# Increase worker concurrency for faster indexing
BACKFILL_CONCURRENCY=8  # Default is 4

# Increase batch size
BACKFILL_BATCH_SIZE=20  # Default is 10

# Increase database connections
DB_MAX_CONNS=30  # Default is 20

# Then restart worker
make run-worker
```

---

## Complete Workflow Examples

### Example 1: Fresh Start - Index First 1000 Blocks

```bash
# 1. Configure RPC endpoint
nano .env
# Update RPC_URL with your Alchemy/Infura key

# 2. Setup database
make db-setup

# 3. Configure indexing
nano .env
# Set:
# BACKFILL_START_HEIGHT=0
# BACKFILL_END_HEIGHT=1000

# 4. Run worker to fetch and extract
make run-worker

# Wait for completion (~30 seconds with Alchemy)
# Output: "backfill phase completed"

# 5. In another terminal, start API
make run-api

# 6. Open browser
open http://localhost:8080/

# 7. Verify data
curl http://localhost:8080/v1/blocks?limit=5
curl http://localhost:8080/v1/stats/chain
```

### Example 2: Index Recent Blocks (7M - 7.1M) with Real-time Monitoring

```bash
# 1. Assume database already setup and has data

# 2. Configure to index recent blocks
cat > .env << EOF
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=postgres
BACKFILL_START_HEIGHT=7000000
BACKFILL_END_HEIGHT=7100000
LIVETAIL_ENABLED=true
EOF

# 3. Start worker
make run-worker

# 4. In separate terminal, start API and web UI
make run-api
# Then open http://localhost:8080/

# 5. Monitor indexing progress
watch -n 1 "curl -s http://localhost:8080/v1/stats/chain | jq '.total_blocks, .total_transactions'"

# 6. Watch live updates appear in web UI automatically
```

### Example 3: Development Workflow - Make Changes and Test

```bash
# 1. Make code changes (e.g., add new API endpoint)
# ... edit files ...

# 2. Rebuild and restart
make fmt          # Format code
make build        # Build binaries

# 3. Stop old services
make stop

# 4. Start fresh with test data (smaller backfill)
BACKFILL_START_HEIGHT=0
BACKFILL_END_HEIGHT=100
make run

# 5. Test your changes
curl http://localhost:8080/v1/YOUR_NEW_ENDPOINT

# 6. View logs
make logs-worker
make logs-api
```

### Example 4: Production Deployment - Index Continuously

```bash
# 1. Build optimized binaries
make build

# 2. Create systemd service for API (Linux)
sudo tee /etc/systemd/system/blockchain-api.service > /dev/null << EOF
[Unit]
Description=Blockchain Explorer API
After=docker.service

[Service]
Type=simple
WorkingDirectory=/path/to/go-blockchain-explorer
EnvironmentFile=/path/to/.env
ExecStart=/path/to/go-blockchain-explorer/bin/api
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 3. Create systemd service for worker
sudo tee /etc/systemd/system/blockchain-worker.service > /dev/null << EOF
[Unit]
Description=Blockchain Explorer Worker
After=docker.service

[Service]
Type=simple
WorkingDirectory=/path/to/go-blockchain-explorer
EnvironmentFile=/path/to/.env
ExecStart=/path/to/go-blockchain-explorer/bin/worker
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 4. Start services
sudo systemctl start blockchain-api
sudo systemctl start blockchain-worker

# 5. Monitor
sudo systemctl status blockchain-api
sudo systemctl status blockchain-worker
sudo journalctl -u blockchain-api -f
sudo journalctl -u blockchain-worker -f
```

---

## Quick Reference

### Essential Commands

```bash
# Setup & Database
make db-setup              # One-time setup
make db-status             # Check database
make db-shell              # Access database

# Running Services
make run                   # Start everything
make run-api               # API only
make run-worker            # Worker only
make stop                  # Stop services
make status                # Check status

# Monitoring
make logs-api              # View API logs
make logs-worker           # View worker logs
tail -f logs/*.log         # All logs

# Development
make build                 # Build binaries
make fmt                   # Format code
make test                  # Run tests
make clean                 # Clean build files

# Docker
make docker-up             # Start containers
make docker-down           # Stop containers
docker-compose logs -f     # View Docker logs
```

### Configuration Cheat Sheet

```bash
# .env file essentials
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY  # CRITICAL
BACKFILL_START_HEIGHT=0                   # Where to start indexing
BACKFILL_END_HEIGHT=1000                 # Where to stop
BACKFILL_CONCURRENCY=4                   # Parallel fetches
LIVETAIL_ENABLED=true                    # Enable real-time monitoring
API_PORT=8080                            # API server port
DB_HOST=localhost                        # Database host
DB_PORT=5432                             # Database port
DB_NAME=blockchain_explorer              # Database name
```

### API Endpoints Quick Reference

```bash
# Blocks
GET /v1/blocks              # List blocks (paginated)
GET /v1/blocks/{height}     # Get by block height
GET /v1/blocks/{hash}       # Get by block hash

# Transactions
GET /v1/txs                 # List transactions
GET /v1/txs/{hash}          # Get by tx hash
GET /v1/blocks/{height}/txs # Get txs in a block

# Addresses
GET /v1/address/{addr}/txs  # Get address transaction history

# Logs
GET /v1/logs                # Get event logs (with filters)

# Statistics
GET /v1/stats/chain         # Get chain statistics
GET /health                 # Health check

# WebSocket
WS /v1/ws                   # Real-time streaming

# Metrics
GET /metrics                # Prometheus metrics
```

---

## Support & Resources

- **Full README:** See [README.md](README.md)
- **API Docs:** See [API.md](API.md)
- **Docker Guide:** See [DOCKER.md](DOCKER.md)
- **Quick Start:** See [QUICKSTART.md](QUICKSTART.md)
- **GitHub Issues:** https://github.com/hieutt50/go-blockchain-explorer/issues

---

**Last Updated:** 2025-11-02
**Project:** Blockchain Explorer - Production-Grade Ethereum Indexer
**Maintainer:** Hieu (https://github.com/hieutt50)
