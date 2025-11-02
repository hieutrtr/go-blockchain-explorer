# Blockchain Explorer

A production-grade Ethereum blockchain indexer and query platform built in Go.

## Overview

This project indexes blockchain data from Ethereum Sepolia testnet, providing APIs for querying blocks, transactions, and logs with real-time updates.

## Features

- **Historical Block Indexing**: Parallel backfill for configurable block ranges
- **Real-Time Monitoring**: Live-tail mechanism for new blocks as they're produced
- **Transaction Extraction**: Full transaction extraction with ECDSA signature recovery
- **Contract Creation Handling**: Proper support for contract creation transactions
- **Chain Reorganization Handling**: Automatic detection and recovery
- **REST API**: Query blocks, transactions, and logs with pagination
- **WebSocket Streaming**: Real-time updates for new blocks and transactions
- **Minimal Frontend**: Single-page application with real blockchain data
- **Observability**: Prometheus metrics and structured JSON logging

## Project Structure

```
go-blockchain-explorer/
├── internal/
│   └── rpc/              # RPC client with retry logic (Story 1.1) ✅
│       ├── client.go     # Main client implementation
│       ├── config.go     # Configuration management
│       ├── errors.go     # Error classification
│       ├── retry.go      # Exponential backoff logic
│       └── *_test.go     # Unit tests
├── docs/                 # Documentation
│   ├── stories/          # Story files
│   ├── PRD.md           # Product requirements
│   ├── solution-architecture.md  # Architecture design
│   └── tech-spec-*.md   # Technical specifications
├── go.mod                # Go module definition
└── README.md             # This file
```

## Getting Started

### Prerequisites

- Go 1.24+ (required by go-ethereum v1.16.5)
- Ethereum RPC endpoint (Alchemy, Infura, or public node)
- PostgreSQL 16+

### Installation

#### Option A: With Docker (Recommended for Quick Start)

```bash
# Clone the repository
git clone https://github.com/hieutt50/go-blockchain-explorer.git
cd go-blockchain-explorer

# Install dependencies
go mod download

# Configure environment variables
cp .env.example .env
# Edit .env with your RPC_URL and other settings

# Start PostgreSQL using Docker
./scripts/docker-setup.sh
# Select option 1 to start PostgreSQL and pgAdmin

# Migrations will run automatically on first start
```

This will:
- Start PostgreSQL 16 in Docker
- Start pgAdmin web interface (optional)
- Automatically run database migrations
- No need to install PostgreSQL on your system

#### Option B: With Local PostgreSQL

```bash
# Clone the repository
git clone https://github.com/hieutt50/go-blockchain-explorer.git
cd go-blockchain-explorer

# Install dependencies
go mod download

# Set up PostgreSQL database
psql -U postgres
CREATE DATABASE blockchain_explorer;
\q

# Run database migrations
psql -U postgres -d blockchain_explorer -f migrations/001_initial_schema.sql

# Configure environment variables
cp .env.example .env
# Edit .env with your configuration
```

### Docker Setup (Optional)

If you chose the Docker option, you can manage your database easily:

```bash
# Interactive setup menu
./scripts/docker-setup.sh

# Or use docker-compose directly
docker-compose up -d          # Start services
docker-compose down           # Stop services
docker-compose logs -f        # View logs
docker-compose restart        # Restart services
```

**Access pgAdmin:**
- URL: http://localhost:5050
- Email: admin@blockchain-explorer.local
- Password: admin

**Connect to PostgreSQL in pgAdmin:**
1. Right-click "Servers" → Register → Server
2. General tab: Name = "Blockchain Explorer"
3. Connection tab:
   - Host: postgres
   - Port: 5432
   - Database: blockchain_explorer
   - Username: postgres
   - Password: postgres

### Configuration

Create a `.env` file with the following required variables:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=your_password
DB_MAX_CONNS=20

# RPC Configuration
RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# API Server Configuration
API_PORT=8080
API_CORS_ORIGINS=*
```

Supported RPC providers:
- **Alchemy**: `https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY`
- **Infura**: `https://sepolia.infura.io/v3/YOUR_API_KEY`
- **Public nodes**: `https://rpc.sepolia.org` (rate limited)

### Running the Application

#### Option 1: Using the Startup Script (Recommended)

```bash
# Make the script executable
chmod +x scripts/run.sh

# Source environment variables and run
source .env && ./scripts/run.sh
```

This will start both the API server and worker in the background.

#### Option 2: Using Makefile

```bash
# Run both API and worker
make run

# Or run individually
make run-api
make run-worker

# Build for production
make build
```

#### Option 3: Manual Execution

**Terminal 1 - Start API Server:**
```bash
source .env
go run cmd/api/main.go
```

The API server will:
- Listen on port 8080 (or API_PORT)
- Provide REST endpoints at `/v1/*`
- Provide WebSocket at `/v1/ws`
- Connect to PostgreSQL database

**Terminal 2 - Start Worker:**
```bash
source .env
go run cmd/worker/main.go
```

The worker will:
- Connect to Ethereum via RPC
- Index blocks and transactions
- Store data in PostgreSQL
- Expose metrics on port 9090

#### Verify Everything is Running

```bash
# Check API health
curl http://localhost:8080/health

# Check latest block
curl http://localhost:8080/v1/blocks?limit=1

# Check worker metrics
curl http://localhost:9090/metrics
```

### Worker Lifecycle Management

The worker is responsible for indexing blockchain data and extracting transactions from blocks.

#### Starting the Worker

```bash
# Option 1: Using Makefile (from .env)
make run-worker

# Option 2: Direct execution
export BACKFILL_START_HEIGHT=0
export BACKFILL_END_HEIGHT=1000
export LIVETAIL_ENABLED=true
go run ./cmd/worker/main.go

# Option 3: Using compiled binary
./bin/worker
```

#### Worker Behavior

The worker performs two main tasks:

1. **Backfill**: Indexes historical blocks from `BACKFILL_START_HEIGHT` to `BACKFILL_END_HEIGHT`
   - Extracts all transactions from each block
   - Recovers sender address via ECDSA signature recovery
   - Handles contract creation (nil to_addr)
   - Calculates transaction fees (gas_used × gas_price)
   - Stores everything atomically in PostgreSQL

2. **Live-Tail**: Continuously monitors for new blocks as they're produced
   - Enabled when `LIVETAIL_ENABLED=true`
   - Automatically indexes new blocks in real-time
   - Detects and handles chain reorganizations
   - Updates frontend via WebSocket

#### Monitoring Worker Progress

```bash
# View worker logs in real-time
make logs-worker

# Check worker metrics
curl http://localhost:9090/metrics | grep worker

# View specific metrics
curl http://localhost:9090/metrics | grep -E "blocks_indexed|transactions_extracted"

# Check database status while worker is running
make db-status
```

#### Worker Troubleshooting

```bash
# If worker crashes, restart it:
make run-worker

# Clear database and restart backfill:
make db-drop
make db-setup
make run-worker

# Check worker logs for errors:
tail -f logs/worker.log
```

### Frontend Web Interface

The project includes a minimal single-page application (SPA) for visualizing blockchain data in real-time.

#### Accessing the Frontend

Once the API server is running, open your browser and navigate to:

```
http://localhost:8080/
```

The frontend is served automatically by the API server from the `web/` directory.

#### Browser Requirements

The frontend requires a modern browser with WebSocket and ES6+ support:

- **Chrome**: 90+ (recommended)
- **Firefox**: 88+
- **Safari**: 14+
- **Edge**: 90+

**Note**: Internet Explorer is not supported.

#### Features

The frontend provides:

- **Live Blocks Ticker**: Displays the 10 most recent blocks with real-time updates
  - Block height, hash (truncated), age (relative time), transaction count
  - Automatically prepends new blocks as they arrive via WebSocket

- **Recent Transactions**: Shows the 25 most recent transactions
  - Transaction hash, from/to addresses (truncated), value in ETH, block height
  - Updates automatically when new blocks are indexed

- **WebSocket Connection Status**: Visual indicator shows connection state
  - Green dot: Connected and receiving updates
  - Red dot: Disconnected (automatic reconnection with exponential backoff)

#### WebSocket Protocol

The frontend connects to the WebSocket endpoint at `ws://localhost:8080/v1/stream`.

**Subscribe to new blocks:**
```json
{
  "type": "subscribe",
  "channel": "newBlocks"
}
```

**Receive block updates:**
```json
{
  "type": "newBlock",
  "data": {
    "height": 5001234,
    "hash": "0x1234...5678",
    "parent_hash": "0xabcd...ef01",
    "timestamp": 1698765432,
    "gas_used": 15234567,
    "gas_limit": 30000000,
    "tx_count": 142,
    "miner": "0x742d...bEb0",
    "orphaned": false
  }
}
```

**Connection Management:**
- Automatic reconnection on disconnect (exponential backoff: 1s, 2s, 4s, 8s max)
- Graceful cleanup on page unload
- Connection status displayed in header

#### Technology Stack

- **Vanilla JavaScript**: No frameworks, direct DOM manipulation
- **HTML5**: Semantic structure for accessibility
- **CSS3**: Responsive layout with CSS Grid and Flexbox
- **WebSocket API**: Real-time updates
- **Fetch API**: Initial data loading

No build step or bundler required - just open in your browser!

## Documentation

- **[API Documentation](API.md)** - Complete REST API reference with examples
- **[Swagger/OpenAPI](SWAGGER.md)** - Interactive API documentation (Try it out!)
- **[Quick Start Guide](QUICKSTART.md)** - Get up and running in 5 minutes
- **[Docker Guide](DOCKER.md)** - Docker setup and management

### Interactive API Documentation (Swagger UI)

Try out the API interactively with Swagger UI:

```bash
# Start Swagger UI
make swagger-up

# Access at http://localhost:8081
open http://localhost:8081
```

Swagger UI provides:
- Interactive API explorer
- Try out endpoints directly from your browser
- Request/response examples
- Complete API schema documentation

### Production Deployment

```bash
# Build binaries
make build

# Run with systemd or supervisor
./bin/api &
./bin/worker &

# Or use Docker (if Dockerfile is available)
docker-compose up -d
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./internal/rpc -v

# Run with race detector
go test ./internal/rpc -race

# Run with coverage
go test ./internal/rpc -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require a real Ethereum RPC endpoint:

```bash
# Run integration tests (requires RPC_URL environment variable)
export RPC_URL="https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY"
go test ./internal/rpc -v -run Integration

# Skip integration tests
go test ./internal/rpc -v -short
```

### End-to-End Transaction Extraction Tests

The project includes comprehensive E2E tests for the transaction extraction pipeline (worker → database → API → frontend):

#### Quick E2E Test (Setup + Worker + Verify)

```bash
# Full E2E test with database setup, worker execution, and verification
make test-transaction-extraction
```

This command:
1. Builds all binaries (API, worker, e2e-verify)
2. Sets up PostgreSQL with Docker
3. Runs database migrations
4. Starts the worker to index blocks 0-5 with transaction extraction
5. Runs verification tests to ensure data integrity

#### Step-by-Step E2E Testing

**Step 1: Setup Database**
```bash
# Start PostgreSQL and run migrations
make db-setup

# Or use Docker directly
docker-compose up -d postgres
```

**Step 2: Run Worker**
```bash
# Option A: Using Makefile (small backfill for testing)
make e2e-test-setup

# Option B: Direct execution with custom block range
export BACKFILL_START_HEIGHT=0
export BACKFILL_END_HEIGHT=10
make run-worker

# Option C: Using binary
./bin/worker
```

**Step 3: Verify Transaction Extraction**
```bash
# Run verification tests
make e2e-verify

# Or run directly
go run cmd/e2e-verify/main.go
```

#### E2E Test Coverage

The E2E verification tests validate:
- ✅ Database connectivity and schema
- ✅ Transaction extraction from blocks
- ✅ ECDSA signature recovery for from_addr
- ✅ Contract creation transaction handling (nil to_addr)
- ✅ Fee calculation (gas_used × gas_price)
- ✅ Block-transaction foreign key integrity
- ✅ Store query methods (GetTransaction, GetBlockTransactions)
- ✅ API endpoints return correct data
- ✅ Frontend uses real API data (no mock data)
- ✅ Data consistency and no orphaned records

#### Viewing Worker Logs

```bash
# View all logs
make logs

# View only worker logs
make logs-worker

# Or directly
tail -f logs/worker-e2e.log
```

#### Worker Configuration for E2E Tests

The worker can be configured with environment variables:

```bash
# Backfill configuration
export BACKFILL_START_HEIGHT=0          # Starting block height
export BACKFILL_END_HEIGHT=100          # Ending block height
export BACKFILL_BATCH_SIZE=10           # Blocks per batch
export BACKFILL_CONCURRENCY=4           # Concurrent batches

# Live-tail configuration
export LIVETAIL_ENABLED=true            # Enable real-time block indexing
export LIVETAIL_START_FROM_TIP=true     # Start from latest block

# Database configuration
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=blockchain_explorer
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_MAX_CONNS=20

# RPC configuration
export RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
export RPC_TIMEOUT=10s

# Logging
export LOG_LEVEL=info                   # debug, info, warn, error
```

#### Performance Benchmarks

Typical performance on Ethereum Sepolia testnet:
- **Backfill Speed**: ~100-200 blocks/second (depends on RPC provider)
- **Transaction Extraction**: ~1000 transactions/second
- **Database Insertion**: ~5000 rows/second (blocks + transactions)
- **API Response Time**: <100ms for paginated queries

## Architecture

### RPC Client (Story 1.1)

The RPC client provides a robust interface to Ethereum nodes with:

- **Automatic Retry Logic**: Exponential backoff for transient failures (1s, 2s, 4s, 8s, 16s)
- **Error Classification**: Distinguishes transient vs permanent errors
- **Structured Logging**: JSON-formatted logs with operation context
- **Timeout Management**: 10s connection timeout, 30s request timeout
- **Rate Limit Handling**: Automatic backoff for HTTP 429 responses

### Error Handling

Errors are classified into three types:

1. **Transient** (retry): Network timeouts, connection refused, DNS errors
2. **Permanent** (fail immediately): Invalid parameters, method not found
3. **Rate Limit** (backoff + retry): HTTP 429, quota exceeded

## Development Status

- ✅ **Story 1.1**: RPC Client with Retry Logic (Complete)
- 📋 **Story 1.2**: PostgreSQL Schema and Migrations (Next)
- 📋 **Story 1.3**: Parallel Backfill Worker Pool
- 📋 **Story 1.4**: Live-Tail Mechanism
- 📋 **Story 1.5**: Chain Reorganization Handling
- 📋 **Stories 1.6-2.6**: Additional features

See `docs/sprint-status.yaml` for detailed progress tracking.

## API Documentation

### REST API Endpoints

All list endpoints support pagination using `limit` and `offset` query parameters.

#### Pagination Parameters

- **limit**: Number of results to return
  - Blocks endpoint: Default 25, Max 100
  - Transactions endpoint: Default 50, Max 100
  - Logs endpoint: Default 100, Max 1000
- **offset**: Number of results to skip (default: 0)

#### Pagination Response Format

All paginated responses include metadata:

```json
{
  "data": [...],
  "total": 5001,
  "limit": 25,
  "offset": 0
}
```

- **data**: Array of results (blocks, transactions, or logs)
- **total**: Total count of results matching the query
- **limit**: Actual limit applied in this response
- **offset**: Actual offset applied in this response

### Example API Requests

#### List Recent Blocks

```bash
# Get first page (25 blocks)
curl "http://localhost:8080/v1/blocks?limit=25&offset=0"

# Get second page
curl "http://localhost:8080/v1/blocks?limit=25&offset=25"

# Get specific number of results
curl "http://localhost:8080/v1/blocks?limit=10&offset=0"
```

#### Get Block by Height or Hash

```bash
# By height
curl "http://localhost:8080/v1/blocks/1000000"

# By hash
curl "http://localhost:8080/v1/blocks/0x1234...5678"
```

#### Get Transaction by Hash

```bash
curl "http://localhost:8080/v1/txs/0xabcd...ef01"
```

#### Get Address Transaction History

```bash
# Get first 50 transactions for an address
curl "http://localhost:8080/v1/address/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0/txs?limit=50&offset=0"

# Get next 50 transactions
curl "http://localhost:8080/v1/address/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0/txs?limit=50&offset=50"
```

#### Query Event Logs

```bash
# Get logs for a specific contract address
curl "http://localhost:8080/v1/logs?address=0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0&limit=100&offset=0"

# Filter by contract address and topic0
curl "http://localhost:8080/v1/logs?address=0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0&topic0=0xddf...abc&limit=100&offset=0"
```

#### Chain Statistics

```bash
curl "http://localhost:8080/v1/stats/chain"
```

Response:
```json
{
  "latest_block": 5001,
  "total_blocks": 5001,
  "total_transactions": 12543,
  "indexer_lag_blocks": 0,
  "indexer_lag_seconds": 2,
  "last_updated": "2025-10-31T10:30:00Z"
}
```

#### Health Check

```bash
curl "http://localhost:8080/health"
```

### Pagination Edge Cases

- **Offset beyond total**: Returns empty array with total count (not an error)
- **Invalid limit/offset**: Silently uses default values for lenient user experience
- **Limit exceeds maximum**: Automatically clamped to configured maximum

### Performance Notes

- **Blocks pagination**: <50ms (indexed by height)
- **Address transactions**: <150ms (indexed by address and block height)
- **Event logs**: <100ms (indexed by address and topic0)
- All queries use database indexes for efficient pagination

## Contributing

This is a portfolio project demonstrating production-grade Go development. Contributions are welcome!

## License

MIT License - See LICENSE file for details

## Author

Hieu - [GitHub](https://github.com/hieutt50)

## Acknowledgments

- Built with [go-ethereum](https://github.com/ethereum/go-ethereum) v1.16.5
- Uses [Sepolia testnet](https://sepolia.etherscan.io) for testing
- Follows BMad Method for structured development
