# Data Flow Guide: From Blockchain to Your Browser

A visual guide showing exactly how data flows through the system.

## High-Level Overview

```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│  ETHEREUM BLOCKCHAIN (Sepolia Testnet)                      │
│  ├─ Block 7000010: Hash=0x123..., Txs=45                   │
│  ├─ Block 7000011: Hash=0x456..., Txs=52                   │
│  └─ Block 7000012: Hash=0x789..., Txs=38                   │
│                                                              │
└──────────────────────────┬─────────────────────────────────┘
                           │
                           │ RPC Protocol
                           │ (eth_getBlockByNumber, etc.)
                           ▼
        ┌──────────────────────────────────────┐
        │  WORKER (Indexer)                    │
        │  ├─ Fetch block 7000010              │
        │  ├─ Extract 45 transactions          │
        │  ├─ Recover sender addresses (ECDSA) │
        │  ├─ Calculate fees                   │
        │  └─ Store in database                │
        └──────────────────────────────────────┘
                           │
                           │ INSERT
                           ▼
        ┌──────────────────────────────────────┐
        │  POSTGRESQL DATABASE (Docker)        │
        │  ├─ blocks table                     │
        │  │  ├─ height: 7000010              │
        │  │  ├─ hash: 0x123...               │
        │  │  └─ tx_count: 45                 │
        │  ├─ transactions table               │
        │  │  ├─ hash: 0xabc...               │
        │  │  ├─ from_addr: 0x123...          │
        │  │  ├─ to_addr: 0x456...            │
        │  │  └─ value: 1500000000000000000   │
        │  └─ logs table                       │
        └──────────────────────────────────────┘
                           │
                           │ SELECT
                           ▼
        ┌──────────────────────────────────────┐
        │  API SERVER (Go HTTP + WebSocket)    │
        │  ├─ GET  /v1/blocks                 │
        │  ├─ GET  /v1/txs                    │
        │  ├─ WS   /v1/ws                     │
        │  └─ GET  /v1/stats/chain            │
        └──────────────────────────────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         JSON via HTTP  WebSocket   Static Files
              │            │            │
              ▼            ▼            ▼
    ┌─────────────────┬──────────────┬──────────────┐
    │  curl/Browser   │  WebSocket   │   Browser    │
    │  REST API       │  Streaming   │   UI (HTML)  │
    │                 │  Updates     │              │
    └─────────────────┴──────────────┴──────────────┘
```

---

## Detailed Data Flow: Step by Step

### Phase 1: Worker Fetches Blockchain Data

```
STEP 1: Configuration
    .env sets:
    ├─ RPC_URL: https://eth-sepolia.g.alchemy.com/v2/KEY
    ├─ BACKFILL_START_HEIGHT: 7000010
    ├─ BACKFILL_END_HEIGHT: 7001000
    ├─ BACKFILL_CONCURRENCY: 4
    └─ LIVETAIL_ENABLED: true

STEP 2: Worker Startup
    go run cmd/worker/main.go
    └─ Loads config
    ├─ Connects to PostgreSQL
    ├─ Checks latest block in database
    ├─ Starts backfill coordinator
    └─ Starts live-tail monitor

STEP 3: Backfill Phase - Parallel Fetching

    Concurrent Batches (BACKFILL_CONCURRENCY=4):

    Batch 1          Batch 2          Batch 3          Batch 4
    ├─ Block 7000010 ├─ Block 7000020 ├─ Block 7000030 ├─ Block 7000040
    ├─ Block 7000011 ├─ Block 7000021 ├─ Block 7000031 ├─ Block 7000041
    ├─ Block 7000012 ├─ Block 7000022 ├─ Block 7000032 ├─ Block 7000042
    └─ ...           └─ ...           └─ ...           └─ ...
         │                │                │                │
         │ (concurrent)   │                │                │
         └────────┬───────┴────────┬───────┴────────┬────────┘
                  │               │                │
                  └───────────────┼────────────────┘
                                  ▼
                    POST to Ethereum RPC:
                    {
                      "jsonrpc": "2.0",
                      "method": "eth_getBlockByNumber",
                      "params": ["0x6a0010", true],
                      "id": 1
                    }

                    Response includes:
                    ├─ Block data (hash, timestamp, etc.)
                    └─ ALL transactions in block
                       ├─ Transaction 1
                       ├─ Transaction 2
                       └─ Transaction N

STEP 4: Transaction Extraction

    For each transaction in block:
    {
      "from": "0x1234...",       ← Sender (recovered via ECDSA)
      "to": "0x5678...",
      "value": "1500000000000000000",  ← 1.5 ETH in Wei
      "gasPrice": "20000000000",       ← 20 Gwei
      "gas": "21000",
      "input": "0x",
      "nonce": 42
    }

    Worker calculates:
    ├─ Fee = gas_used × gas_price
    │        = 21000 × 20000000000
    │        = 420000000000000 Wei
    │        = 0.00042 ETH
    └─ Sender = ECDSA signature recovery on (v, r, s)
```

### Phase 2: Data Storage

```
STEP 5: Insert into PostgreSQL (Atomic Transaction)

    BEGIN TRANSACTION
    ├─ INSERT INTO blocks (height, hash, parent_hash, ...)
    │  VALUES (7000010, 0x123abc, 0xdef456, ...)
    │  ✓ 1 block inserted
    │
    ├─ INSERT INTO transactions (hash, from_addr, to_addr, ...)
    │  VALUES (tx_hash_1, from_1, to_1, ...)
    │  VALUES (tx_hash_2, from_2, to_2, ...)
    │  VALUES (tx_hash_3, from_3, to_3, ...)
    │  ... (45 transactions)
    │  ✓ 45 transactions inserted
    │
    ├─ INSERT INTO logs (address, topic0, data, ...)
    │  VALUES (contract_1, topic_1, data_1, ...)
    │  ... (N event logs)
    │  ✓ N logs inserted
    │
    COMMIT

    Result: Database now has block 7000010 with all data

STEP 6: Repeat for Next Block
    Block 7000011:
    ├─ Fetch from RPC
    ├─ Extract 52 transactions
    ├─ Insert into database
    └─ Continue...
```

### Phase 3: Query Data via API

```
STEP 7: API Server Receives Request

    User Request:
    GET /v1/blocks?limit=10&offset=0

    API Handler:
    ├─ Validate parameters (limit ≤ 100, offset ≥ 0)
    ├─ Query database:
    │  SELECT height, hash, tx_count, timestamp, ...
    │  FROM blocks
    │  ORDER BY height DESC
    │  LIMIT 10 OFFSET 0
    │
    ├─ Get 10 blocks from database
    ├─ Format as JSON
    └─ Return response

STEP 8: Response Structure

    {
      "data": [
        {
          "height": 7001000,
          "hash": "0x1234567890abcdef...",
          "parent_hash": "0xfedcba0987654321...",
          "timestamp": 1698765432,
          "gas_used": 15234567,
          "gas_limit": 30000000,
          "tx_count": 142,
          "miner": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
          "orphaned": false
        },
        ... (9 more blocks)
      ],
      "total": 991,
      "limit": 10,
      "offset": 0
    }
```

### Phase 4: Real-time WebSocket Streaming

```
STEP 9: WebSocket Connection Setup

    Client connects: ws://localhost:8080/v1/ws

    Client sends:
    {
      "action": "subscribe",
      "channel": "blocks"
    }

    Server:
    ├─ Registers client in WebSocket hub
    └─ Now tracks this connection

STEP 10: Live Block Arrives (Real-time)

    [A new block is produced on Sepolia]

    Worker's live-tail detects:
    ├─ Polls RPC every 12 seconds
    ├─ Finds new block 7001001
    ├─ Extracts transactions
    ├─ Inserts into database
    └─ Broadcasts to WebSocket hub

    WebSocket Hub:
    ├─ Receives new block event
    ├─ Finds all subscribed clients
    └─ Sends to each client:

    {
      "type": "newBlock",
      "data": {
        "height": 7001001,
        "hash": "0xabc123def456...",
        "tx_count": 145,
        "timestamp": 1698765444,
        "gas_used": 15234890,
        ...
      }
    }

    Client (Browser):
    ├─ Receives message
    ├─ Parses JSON
    ├─ Updates display
    └─ User sees new block appear!

STEP 11: Live Updates Flow

    Ethereum Block Produced
         ↓
    Worker detects (every 12s poll)
         ↓
    Fetches and indexes
         ↓
    Stores in database
         ↓
    Broadcasts to WebSocket
         ↓
    Browser receives
         ↓
    UI updates instantly

    (All in ~15 seconds)
```

---

## Data Transformation Examples

### Example 1: Block Ingestion

**From RPC:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "number": "0x6a0010",
    "hash": "0x1234567890...",
    "parentHash": "0xfedcba...",
    "timestamp": "0x6540ab01",
    "gasUsed": "0xe8ac07",
    "gasLimit": "0x1c9c380",
    "miner": "0x742d...",
    "transactions": [ ... ]
  }
}
```

**Converted to Database:**
```sql
INSERT INTO blocks (height, hash, parent_hash, timestamp, gas_used, gas_limit, miner, tx_count)
VALUES (
  7000010,                                  -- 0x6a0010 converted
  '0x1234567890...',
  '0xfedcba...',
  1698764545,                               -- 0x6540ab01 converted
  241927687,                                -- 0xe8ac07 converted
  30000000,                                 -- 0x1c9c380 converted
  '0x742d...',
  45                                        -- Counted from transactions array
);
```

### Example 2: Transaction Extraction

**From Block RPC Response:**
```json
{
  "hash": "0xabc123def456...",
  "from": "0x1234...",          -- Actually reconstructed!
  "to": "0x5678...",
  "value": "1500000000000000000",
  "gasPrice": "20000000000",
  "gas": "21000",
  "input": "0x",
  "v": 38,                       -- Signature component
  "r": "0xabcd...",              -- Signature component
  "s": "0xef12..."               -- Signature component
}
```

**Worker Processing:**
```
1. Extract (v, r, s) from transaction
2. Apply ECDSA recovery algorithm
   └─ Recovers public key → recovers sender address
3. Verify: recovered sender = "0x1234..."
4. Calculate fee: 21000 * 20000000000 = 420000000000000 Wei
```

**Inserted to Database:**
```sql
INSERT INTO transactions (hash, from_addr, to_addr, value, gas_price, gas, gas_used)
VALUES (
  '0xabc123def456...',
  '0x1234...',              -- Recovered by ECDSA
  '0x5678...',
  1500000000000000000,
  20000000000,
  21000,
  21000
);
```

### Example 3: Contract Creation (Special Case)

**From RPC (contract creation):**
```json
{
  "hash": "0x987654321abc...",
  "from": "0x9999...",
  "to": null,                    -- NULL means contract creation!
  "input": "0x6060604052...",    -- Contract bytecode
  "contractAddress": "0xnewcontract..."
}
```

**Handled by Worker:**
```
1. Detects to_addr is null
2. Sets to_addr = NULL in database (intentional)
3. Stores contractAddress separately if needed
4. Marks as contract creation transaction
```

**In Database:**
```sql
INSERT INTO transactions (hash, from_addr, to_addr, is_contract_creation)
VALUES (
  '0x987654321abc...',
  '0x9999...',
  NULL,                         -- Contract creation
  true
);
```

---

## Performance Metrics

### Typical Flow Times

```
WITH ALCHEMY RPC:

1. Fetch Block from RPC:          50-100ms
2. Extract Transactions:          1-5ms (CPU bound)
3. ECDSA Recovery (per tx):       0.1-0.5ms
4. Calculate Fees:                <0.1ms
5. Insert to Database:            10-20ms per batch
6. Broadcast to WebSocket:        <1ms

Total per block:
├─ 45 transactions per block
├─ Time: ~200-400ms
└─ Speed: 2.5-5 blocks/second
   or: 35,000+ transactions/second
```

### Scaling

```
For 1,000,000 blocks:
├─ Processing: ~200,000-400,000ms = 55-110 hours
├─ Storage: ~50-100GB (blocks + transactions)
└─ Real-time after initial indexing

With CONCURRENCY=8 (parallel):
├─ Processing: ~25-55 hours
└─ CPU usage: ~400-500%
```

---

## Data Model

### Blocks Table

```sql
CREATE TABLE blocks (
    id SERIAL PRIMARY KEY,
    height BIGINT UNIQUE NOT NULL,           -- Block number
    hash VARCHAR(66) NOT NULL,                -- Block hash (0x prefix)
    parent_hash VARCHAR(66) NOT NULL,        -- Previous block hash
    timestamp BIGINT NOT NULL,                -- Unix timestamp
    gas_used BIGINT NOT NULL,                 -- Total gas used
    gas_limit BIGINT NOT NULL,                -- Total gas limit
    tx_count INT NOT NULL,                    -- Number of transactions
    miner VARCHAR(42),                        -- Miner/validator address
    orphaned BOOLEAN DEFAULT FALSE,           -- Reorg detected?
    created_at TIMESTAMP DEFAULT NOW()
);

Index: CREATE INDEX idx_blocks_height ON blocks(height);
       CREATE INDEX idx_blocks_timestamp ON blocks(timestamp);
```

### Transactions Table

```sql
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(66) UNIQUE NOT NULL,        -- Transaction hash
    block_height BIGINT NOT NULL,            -- Block containing tx
    from_addr VARCHAR(42) NOT NULL,          -- Sender (recovered)
    to_addr VARCHAR(42),                     -- Recipient (NULL for contract creation)
    value NUMERIC NOT NULL,                  -- ETH amount (in Wei)
    gas_price NUMERIC NOT NULL,              -- Gas price (in Wei/gas)
    gas BIGINT NOT NULL,                     -- Gas limit
    gas_used BIGINT NOT NULL,                -- Actual gas used
    nonce BIGINT NOT NULL,                   -- Sender's transaction count
    data TEXT NOT NULL,                      -- Input data (for contract calls)
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (block_height) REFERENCES blocks(height)
);

Indexes:
  CREATE INDEX idx_txs_block_height ON transactions(block_height);
  CREATE INDEX idx_txs_from_addr ON transactions(from_addr);
  CREATE INDEX idx_txs_to_addr ON transactions(to_addr);
  CREATE INDEX idx_txs_hash ON transactions(hash);
```

### Logs Table

```sql
CREATE TABLE logs (
    id SERIAL PRIMARY KEY,
    tx_hash VARCHAR(66),                     -- Transaction containing log
    block_height BIGINT NOT NULL,            -- Block containing log
    address VARCHAR(42) NOT NULL,            -- Contract emitting event
    topic0 VARCHAR(66),                      -- Event signature hash
    topic1 VARCHAR(66),                      -- Indexed parameter 1
    topic2 VARCHAR(66),                      -- Indexed parameter 2
    topic3 VARCHAR(66),                      -- Indexed parameter 3
    data TEXT NOT NULL,                      -- Non-indexed data
    log_index INT NOT NULL,                  -- Log position in block
    created_at TIMESTAMP DEFAULT NOW()
);

Indexes:
  CREATE INDEX idx_logs_block_height ON logs(block_height);
  CREATE INDEX idx_logs_address ON logs(address);
  CREATE INDEX idx_logs_topic0 ON logs(topic0);
```

---

## Common Queries

### Users Want This

```bash
# "Show me the latest 10 blocks"
GET /v1/blocks?limit=10&offset=0
```

### API Does This

```sql
SELECT height, hash, tx_count, timestamp
FROM blocks
ORDER BY height DESC
LIMIT 10 OFFSET 0;
```

### What Flows

```
User → Browser → curl → HTTP → API Handler → PostgreSQL
                                     ↓
                          SELECT query runs
                                     ↓
                          Database returns 10 rows
                                     ↓
                          API formats as JSON
                                     ↓
                          HTTP response sent
                                     ↓
Browser receives JSON → Parses → Displays in table
```

---

## Error Handling

### What Goes Wrong

```
RPC Timeout
├─ Worker tries to fetch block
├─ RPC endpoint doesn't respond (> 10s)
├─ Error: "context deadline exceeded"
└─ Worker retries with exponential backoff
   ├─ Retry 1: Wait 1 second, try again
   ├─ Retry 2: Wait 2 seconds, try again
   ├─ Retry 3: Wait 4 seconds, try again
   └─ After 5 retries: Skip block and continue

Database Error
├─ INSERT fails (duplicate block?)
├─ Error logged with full context
├─ Transaction rolled back
└─ Worker continues (doesn't crash)

Chain Reorganization
├─ Sepolia reorgs (chain reorganizes)
├─ Worker detects: New block height < database height
├─ Reorg handler kicks in
├─ Removes orphaned blocks from database
└─ Continues with correct chain
```

---

## Monitoring the Flow

### Health Check Points

```bash
# 1. Is RPC responding?
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  $RPC_URL

# 2. Is PostgreSQL accessible?
docker exec blockchain-explorer-db pg_isready -U postgres

# 3. Is API server running?
curl http://localhost:8080/health

# 4. How many blocks indexed?
curl http://localhost:8080/v1/stats/chain | jq '.total_blocks'

# 5. Is worker fetching?
tail -f logs/worker.log | grep "indexing"

# 6. WebSocket working?
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
  http://localhost:8080/v1/ws
```

---

## Summary: Data Path

```
RPC Network
    ↓ (JSON-RPC Protocol)
Worker Process
    ├─ Fetch
    ├─ Parse
    ├─ Extract
    ├─ Calculate
    └─ Transform
    ↓ (INSERT SQL)
PostgreSQL Database
    ├─ blocks table
    ├─ transactions table
    └─ logs table
    ↓ (SELECT SQL)
API Server
    ├─ REST Endpoints
    ├─ WebSocket Hub
    └─ Static Files
    ↓ (HTTP/WebSocket)
Client Browser
    ├─ Web UI
    ├─ API Calls
    └─ Real-time Updates
```

---

**Understanding this flow helps you:**
- ✅ Debug issues
- ✅ Optimize performance
- ✅ Monitor indexing
- ✅ Scale the system
- ✅ Handle errors gracefully

