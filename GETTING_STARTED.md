# Getting Started - 5 Minute Setup

The fastest way to get the Blockchain Explorer running.

## Prerequisites Check

```bash
# Verify you have everything
which go          # Should show Go path
docker --version  # Should show Docker version
```

If any are missing:
- **Go:** https://golang.org/doc/install
- **Docker:** https://docs.docker.com/get-docker/

## Step 1️⃣: Get RPC Endpoint (2 minutes)

### Fastest Option: Use Alchemy

1. Go to https://www.alchemy.com/
2. Click "Get started for free"
3. Fill in sign-up form
4. Choose "Sepolia" network
5. Copy your API key
6. Add to `.env`:
   ```bash
   RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
   ```

**Why Alchemy?**
- ⚡ 50-100ms response time
- 300 requests/second (way more than you need)
- No indexing delays

---

## Step 2️⃣: Setup Project (1 minute)

```bash
# Clone
git clone https://github.com/hieutt50/go-blockchain-explorer.git
cd go-blockchain-explorer

# Configure
cp .env.example .env
# Edit .env and add your RPC_URL (from Step 1)
nano .env

# Setup database (Docker will start automatically)
make db-setup
```

---

## Step 3️⃣: Run (1 minute)

### Option A: Both API + Worker (Recommended)

```bash
make run

# Will output:
# API listening on port 8080
# Worker indexing blocks...
```

### Option B: Separate Terminals

**Terminal 1:**
```bash
make run-api
# → API server starts on http://localhost:8080
```

**Terminal 2:**
```bash
make run-worker
# → Worker fetches and indexes blockchain data
```

---

## Step 4️⃣: View Data (1 minute)

### Web UI (Easiest)
```
Open: http://localhost:8080/
```

You'll see:
- 📊 Live blocks updating in real-time
- 📋 Latest transactions
- 🟢 WebSocket connection status

### Or Use curl

```bash
# Get latest blocks
curl http://localhost:8080/v1/blocks?limit=5

# Get chain stats
curl http://localhost:8080/v1/stats/chain

# Get transactions
curl http://localhost:8080/v1/txs?limit=10
```

---

## Done! 🎉

You now have a fully functional blockchain explorer that:
- ✅ Fetches live blockchain data
- ✅ Extracts all transactions
- ✅ Stores in PostgreSQL
- ✅ Serves via REST API + WebSocket
- ✅ Shows real-time web UI

---

## What's Indexing?

While your worker is running, check the logs:

```bash
# Watch indexing progress
tail -f logs/worker.log

# Check how many blocks are indexed
make db-status

# Query the data
curl http://localhost:8080/v1/stats/chain
```

---

## Common Next Steps

### Configure Indexing Range

Edit `.env`:
```bash
BACKFILL_START_HEIGHT=0          # Start from block 0
BACKFILL_END_HEIGHT=10000        # Index up to block 10000
BACKFILL_CONCURRENCY=4           # Fetch 4 blocks at a time (adjust for speed)
```

Then restart: `make run-worker`

### Access Database Directly

```bash
# Open PostgreSQL shell
make db-shell

# Useful queries:
SELECT COUNT(*) FROM blocks;           # How many blocks?
SELECT COUNT(*) FROM transactions;     # How many transactions?
SELECT * FROM blocks ORDER BY height DESC LIMIT 5;  # Latest blocks
```

### View Metrics

```bash
# Prometheus metrics
curl http://localhost:9090/metrics | grep blockchain_

# Watch real-time
watch -n 1 "curl -s http://localhost:9090/metrics | grep blockchain_"
```

### Stop Everything

```bash
make stop
# or
docker-compose down
```

---

## Troubleshooting

### "context deadline exceeded" Error

**Problem:** RPC endpoint too slow

**Fix:** Use Alchemy instead of public node
```bash
# Check current RPC_URL
grep RPC_URL .env

# If it says rpc.sepolia.org, change to Alchemy
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
```

### Port 8080 Already in Use

**Fix:** Use different port
```bash
# In .env, change:
API_PORT=8081

# Then restart
make run-api
# Access at http://localhost:8081
```

### Database Won't Start

**Fix:** Restart Docker
```bash
docker-compose down -v
make db-setup
```

---

## Architecture Overview

```
Internet (Ethereum Sepolia)
           ↓ RPC API
      [Worker] ← Fetches blocks & transactions
           ↓ Stores
      [PostgreSQL] ← Database
           ↓ Reads
       [API Server] ← REST + WebSocket
           ↓ Returns
    [Your Browser / App]
```

---

## Configuration Deep Dive

Your `.env` controls everything:

```bash
# RPC - How to connect to Ethereum
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# Database - Where to store data
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=postgres

# Backfill - Which blocks to fetch
BACKFILL_START_HEIGHT=0         # First block to index
BACKFILL_END_HEIGHT=1000        # Last block to index
BACKFILL_BATCH_SIZE=10          # Blocks per request
BACKFILL_CONCURRENCY=4          # Parallel requests (higher = faster, but more CPU)

# Real-time - Live block monitoring
LIVETAIL_ENABLED=true           # Enable continuous monitoring
LIVETAIL_START_FROM_TIP=true    # Start from latest block

# API - Web server
API_PORT=8080                   # Port for web interface
```

---

## Speed Expectations

With Alchemy:
- **Indexing speed:** 50-100 blocks/second
- **1000 blocks:** ~10 seconds
- **100,000 blocks:** ~15 minutes
- **Transactions extracted:** ~1000/second

---

## All Available Commands

```bash
# Quick start
make run              # Start everything (API + Worker)
make db-setup        # Initialize database

# Individual services
make run-api          # Start API only
make run-worker       # Start worker only
make stop             # Stop all services

# Database
make db-status        # Check database health
make db-shell         # Connect to database
make db-drop          # Delete all data (warning!)

# Logs
make logs             # View all logs
make logs-api         # View API logs only
make logs-worker      # View worker logs only

# Development
make build            # Compile binaries
make test             # Run tests
make fmt              # Format code
make clean            # Remove build files
```

---

## What Data Gets Indexed?

For each block, the worker extracts and stores:

### Blocks
- Block height, hash, parent hash
- Timestamp, gas used/limit
- Miner address
- Transaction count

### Transactions
- Transaction hash
- From/to addresses
- Value (ETH amount)
- Gas price, gas used
- Nonce, input data
- **Sender recovered** via ECDSA signature recovery
- **Fees calculated** (gas_used × gas_price)

### Event Logs
- Contract address
- Event topics (0-3)
- Log data
- Log index

---

## Next: Advanced Usage

Once basic setup works, see:
- [COMPLETE_WORKFLOW_GUIDE.md](COMPLETE_WORKFLOW_GUIDE.md) - Deep dive with all options
- [API.md](API.md) - All API endpoints
- [README.md](README.md) - Full documentation
- [DOCKER.md](DOCKER.md) - Docker deployment options

---

## Performance Tips

```bash
# For faster indexing:
# 1. Use Alchemy (not public node)
# 2. Increase concurrency
BACKFILL_CONCURRENCY=8          # Instead of default 4

# 3. Increase batch size (if RPC is fast)
BACKFILL_BATCH_SIZE=20          # Instead of default 10

# 4. More database connections
DB_MAX_CONNS=30                 # Instead of default 20

# Then restart:
make run-worker
```

---

## Monitoring Your Indexing

```bash
# Terminal 1: Start services
make run

# Terminal 2: Watch logs
tail -f logs/worker.log

# Terminal 3: Check progress
while true; do
  curl -s http://localhost:8080/v1/stats/chain | jq '.'
  sleep 5
done

# Terminal 4: Watch database growth
watch -n 1 "docker exec blockchain-explorer-db psql -U postgres -d blockchain_explorer -c 'SELECT COUNT(*) as blocks, (SELECT COUNT(*) FROM transactions) as txs FROM blocks;'"
```

---

**Ready to index the blockchain? Run `make db-setup && make run`** 🚀

