# Quick Start Guide

Get the blockchain explorer up and running in 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- Go 1.24+ installed
- Ethereum RPC endpoint (Alchemy/Infura API key)

## Step-by-Step Setup

### 1. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/hieutt50/go-blockchain-explorer.git
cd go-blockchain-explorer

# Copy environment template
cp .env.example .env

# Edit .env and add your RPC URL
# Required: Update RPC_URL with your Alchemy/Infura API key
nano .env  # or use your preferred editor
```

### 2. Setup Database (One Command!)

```bash
# This single command will:
# - Start PostgreSQL in Docker
# - Create the database
# - Run all migrations
make db-setup
```

Expected output:
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

### 3. Run the Application

```bash
# Start both API server and worker
make run
```

This will start:
- **API Server** on port 8080
- **Worker** (blockchain indexer) in background
- Logs are saved to `logs/api.log` and `logs/worker.log`

### 4. Verify Everything Works

```bash
# Check health
curl http://localhost:8080/health

# Get latest blocks
curl http://localhost:8080/v1/blocks?limit=5

# Check worker metrics
curl http://localhost:9090/metrics
```

### 5. Access pgAdmin (Optional)

Open http://localhost:5050 in your browser:
- **Email**: admin@blockchain-explorer.local
- **Password**: admin

**Connect to database:**
1. Right-click "Servers" → Register → Server
2. General: Name = "Blockchain Explorer"
3. Connection:
   - Host: `postgres`
   - Port: `5432`
   - Database: `blockchain_explorer`
   - Username: `postgres`
   - Password: `postgres`

## Common Commands

```bash
# Database
make db-status        # Check database status
make db-shell         # Open PostgreSQL shell
make migrate          # Run migrations
make migrate-down     # Rollback migrations (with confirmation)

# Application
make run              # Start API + worker
make stop             # Stop services
make status           # Check service status
make logs-api         # View API logs
make logs-worker      # View worker logs

# Docker
make docker-up        # Start PostgreSQL + pgAdmin
make docker-down      # Stop Docker services

# Development
make build            # Build binaries
make test             # Run tests
make fmt              # Format code
make help             # Show all commands
```

## Troubleshooting

### Database won't start

```bash
# Check Docker is running
docker ps

# Restart database
docker-compose restart postgres

# Or remove and recreate
docker-compose down -v
make db-setup
```

### Migrations fail

```bash
# Check if database exists
make db-status

# Recreate database
make db-drop
make db-setup
```

### Can't connect to RPC endpoint

```bash
# Test RPC connection
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  $RPC_URL

# Update RPC_URL in .env file
nano .env
```

### Port already in use

```bash
# Find what's using port 8080
lsof -i :8080

# Kill the process or change port in .env
API_PORT=8081
```

### View detailed logs

```bash
# Follow API logs
tail -f logs/api.log

# Follow worker logs
tail -f logs/worker.log

# View last 100 lines
tail -100 logs/api.log
```

## What's Next?

### Test the API

```bash
# Get latest block
curl http://localhost:8080/v1/blocks/latest

# Get block by number
curl http://localhost:8080/v1/blocks/18500000

# Get transaction by hash
curl http://localhost:8080/v1/txs/0xYOUR_TX_HASH

# List recent transactions
curl http://localhost:8080/v1/txs?limit=10

# Get address history
curl http://localhost:8080/v1/address/0xYOUR_ADDRESS/txs

# Get chain stats
curl http://localhost:8080/v1/stats/chain
```

### WebSocket Streaming

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/v1/ws');

// Subscribe to new blocks
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'blocks'
}));

// Handle messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('New block:', data);
};
```

### Monitor Performance

```bash
# Watch metrics
watch -n 1 curl -s http://localhost:9090/metrics | grep blockchain_

# Monitor database size
make db-shell
# Then in psql:
SELECT pg_size_pretty(pg_database_size('blockchain_explorer'));

# Check container resources
docker stats blockchain-explorer-db
```

### Production Deployment

```bash
# Build optimized binaries
make build

# Run binaries directly
./bin/api &
./bin/worker &

# Set up systemd services (Linux)
# See docs for systemd service examples
```

## Architecture Overview

```
┌─────────────┐
│  Ethereum   │
│   Network   │
└──────┬──────┘
       │ RPC/WebSocket
       │
┌──────▼──────┐
│   Worker    │ ← Indexes blocks & transactions
│  (Indexer)  │
└──────┬──────┘
       │
       │ Writes to
       │
┌──────▼──────────────┐
│   PostgreSQL        │
│   (Docker)          │
└──────┬──────────────┘
       │
       │ Reads from
       │
┌──────▼──────┐     ┌──────────────┐
│  API Server │────►│   Clients    │
│  (REST/WS)  │     │ (Web/Mobile) │
└─────────────┘     └──────────────┘
```

## Directory Structure

```
go-blockchain-explorer/
├── cmd/
│   ├── api/           # API server entry point
│   └── worker/        # Worker entry point
├── internal/
│   ├── api/           # HTTP handlers and WebSocket
│   ├── db/            # Database layer
│   ├── indexer/       # Blockchain indexing logic
│   ├── rpc/           # RPC client
│   └── util/          # Logging & utilities
├── migrations/        # Database migrations
├── scripts/           # Helper scripts
│   ├── run.sh         # Start services
│   ├── stop.sh        # Stop services
│   └── status.sh      # Check status
├── logs/              # Application logs (created on run)
├── .env              # Configuration (create from .env.example)
└── Makefile          # Build and development commands
```

## Support

- **Documentation**: See [README.md](README.md) for full documentation
- **Docker Guide**: See [DOCKER.md](DOCKER.md) for Docker-specific instructions
- **Issues**: [GitHub Issues](https://github.com/hieutt50/go-blockchain-explorer/issues)

## Clean Start

If you want to start completely fresh:

```bash
# Stop everything
make stop
docker-compose down -v

# Remove logs
rm -rf logs/*.log logs/*.pid

# Start fresh
make db-setup
make run
```

That's it! You're ready to explore the blockchain! 🚀
