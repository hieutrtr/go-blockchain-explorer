# Blockchain Explorer API Documentation

Complete REST API reference for the Blockchain Explorer.

**Base URL:** `http://localhost:8080`

**Version:** v1

---

## Table of Contents

- [Authentication](#authentication)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Pagination](#pagination)
- [Endpoints](#endpoints)
  - [Health Check](#health-check)
  - [Blocks](#blocks)
  - [Transactions](#transactions)
  - [Addresses](#addresses)
  - [Event Logs](#event-logs)
  - [Chain Statistics](#chain-statistics)
  - [WebSocket Streaming](#websocket-streaming)
  - [Metrics](#metrics)

---

## Authentication

Currently, the API is **public and does not require authentication**. This may change in future versions.

## Response Format

All API responses return JSON with the following structure:

### Success Response
```json
{
  "data": { ... },
  "timestamp": "2025-10-31T10:00:00Z"
}
```

### Paginated Response
```json
{
  "data": [ ... ],
  "total": 5001,
  "limit": 25,
  "offset": 0
}
```

## Error Handling

Error responses follow this format:

```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "timestamp": "2025-10-31T10:00:00Z"
}
```

### HTTP Status Codes

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 404 | Not Found - Resource doesn't exist |
| 500 | Internal Server Error |
| 503 | Service Unavailable - Database or service issue |

## Rate Limiting

Currently, there are **no rate limits** enforced. This may change in production environments.

## Pagination

List endpoints support pagination using query parameters:

| Parameter | Description | Default | Maximum |
|-----------|-------------|---------|---------|
| `limit` | Number of results to return | Varies by endpoint | Varies by endpoint |
| `offset` | Number of results to skip | 0 | No limit |

**Example:**
```bash
GET /v1/blocks?limit=10&offset=20
```

---

## Endpoints

### Health Check

Check the API server and database health.

#### Request
```http
GET /health
```

#### Response
```json
{
  "status": "ok",
  "database": "healthy",
  "uptime_seconds": 3600,
  "timestamp": "2025-10-31T10:00:00Z"
}
```

#### Status Codes
- `200` - Service healthy
- `503` - Service unhealthy

#### Example
```bash
curl http://localhost:8080/health
```

---

## Blocks

### List Recent Blocks

Get a paginated list of recent blocks, ordered by height descending.

#### Request
```http
GET /v1/blocks?limit={limit}&offset={offset}
```

#### Parameters
| Parameter | Type | Required | Default | Max | Description |
|-----------|------|----------|---------|-----|-------------|
| `limit` | integer | No | 25 | 100 | Number of blocks to return |
| `offset` | integer | No | 0 | - | Number of blocks to skip |

#### Response
```json
{
  "blocks": [
    {
      "height": 18500000,
      "hash": "0x1234567890abcdef...",
      "parent_hash": "0xabcdef1234567890...",
      "miner": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
      "gas_used": "15000000",
      "gas_limit": "30000000",
      "timestamp": 1698768000,
      "tx_count": 150,
      "orphaned": false,
      "created_at": "2025-10-31T10:00:00Z"
    }
  ],
  "total": 18500000,
  "limit": 25,
  "offset": 0
}
```

#### Example
```bash
# Get first page of blocks
curl "http://localhost:8080/v1/blocks?limit=10&offset=0"

# Get next page
curl "http://localhost:8080/v1/blocks?limit=10&offset=10"
```

---

### Get Block by Height or Hash

Get detailed information about a specific block.

#### Request
```http
GET /v1/blocks/{heightOrHash}
```

#### Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `heightOrHash` | string/integer | Yes | Block height (e.g., `18500000`) or block hash (e.g., `0x1234...`) |

#### Response
```json
{
  "height": 18500000,
  "hash": "0x1234567890abcdef...",
  "parent_hash": "0xabcdef1234567890...",
  "miner": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "gas_used": "15000000",
  "gas_limit": "30000000",
  "timestamp": 1698768000,
  "tx_count": 150,
  "orphaned": false,
  "created_at": "2025-10-31T10:00:00Z"
}
```

#### Status Codes
- `200` - Block found
- `400` - Invalid block height or hash format
- `404` - Block not found

#### Examples
```bash
# Get block by height
curl "http://localhost:8080/v1/blocks/18500000"

# Get block by hash
curl "http://localhost:8080/v1/blocks/0x1234567890abcdef..."
```

---

## Transactions

### Get Transaction by Hash

Get detailed information about a specific transaction.

#### Request
```http
GET /v1/txs/{hash}
```

#### Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `hash` | string | Yes | Transaction hash (0x + 64 hex characters) |

#### Response
```json
{
  "hash": "0xabcdef1234567890...",
  "block_height": 18500000,
  "tx_index": 42,
  "from_addr": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "to_addr": "0x1234567890abcdef...",
  "value_wei": "1000000000000000000",
  "fee_wei": "21000000000000",
  "gas_used": "21000",
  "gas_price": "1000000000",
  "nonce": 10,
  "success": true,
  "created_at": "2025-10-31T10:00:00Z"
}
```

#### Status Codes
- `200` - Transaction found
- `400` - Invalid transaction hash format
- `404` - Transaction not found

#### Example
```bash
curl "http://localhost:8080/v1/txs/0xabcdef1234567890..."
```

---

## Addresses

### Get Address Transaction History

Get all transactions involving a specific address (as sender or receiver).

#### Request
```http
GET /v1/address/{addr}/txs?limit={limit}&offset={offset}
```

#### Parameters
| Parameter | Type | Required | Default | Max | Description |
|-----------|------|----------|---------|-----|-------------|
| `addr` | string | Yes | - | - | Ethereum address (0x + 40 hex characters) |
| `limit` | integer | No | 50 | 100 | Number of transactions to return |
| `offset` | integer | No | 0 | - | Number of transactions to skip |

#### Response
```json
{
  "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
  "transactions": [
    {
      "hash": "0xabcdef1234567890...",
      "block_height": 18500000,
      "tx_index": 42,
      "from_addr": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
      "to_addr": "0x1234567890abcdef...",
      "value_wei": "1000000000000000000",
      "fee_wei": "21000000000000",
      "gas_used": "21000",
      "gas_price": "1000000000",
      "nonce": 10,
      "success": true,
      "created_at": "2025-10-31T10:00:00Z"
    }
  ],
  "total": 1234,
  "limit": 50,
  "offset": 0
}
```

#### Status Codes
- `200` - Success
- `400` - Invalid address format

#### Example
```bash
# Get first 50 transactions
curl "http://localhost:8080/v1/address/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0/txs?limit=50"

# Get next page
curl "http://localhost:8080/v1/address/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0/txs?limit=50&offset=50"
```

---

## Event Logs

### Query Event Logs

Query smart contract event logs with optional filters.

#### Request
```http
GET /v1/logs?address={address}&topic0={topic0}&limit={limit}&offset={offset}
```

#### Parameters
| Parameter | Type | Required | Default | Max | Description |
|-----------|------|----------|---------|-----|-------------|
| `address` | string | No | - | - | Contract address to filter by |
| `topic0` | string | No | - | - | Event signature hash (topic0) |
| `limit` | integer | No | 100 | 1000 | Number of logs to return |
| `offset` | integer | No | 0 | - | Number of logs to skip |

#### Response
```json
{
  "logs": [
    {
      "id": 12345,
      "tx_hash": "0xabcdef1234567890...",
      "log_index": 0,
      "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
      "topic0": "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
      "topic1": "0x000000000000000000000000742d35cc6634c0532925a3b844bc9e7595f0beb0",
      "topic2": "0x0000000000000000000000001234567890abcdef...",
      "topic3": null,
      "data": "0x0000000000000000000000000000000000000000000000000de0b6b3a7640000",
      "created_at": "2025-10-31T10:00:00Z"
    }
  ],
  "total": 5678,
  "limit": 100,
  "offset": 0
}
```

#### Status Codes
- `200` - Success
- `400` - Invalid address or topic0 format

#### Examples
```bash
# Get all logs for a contract
curl "http://localhost:8080/v1/logs?address=0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"

# Filter by event signature (Transfer event)
curl "http://localhost:8080/v1/logs?address=0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0&topic0=0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

# Paginate results
curl "http://localhost:8080/v1/logs?address=0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0&limit=100&offset=0"
```

---

## Chain Statistics

### Get Chain Statistics

Get overall blockchain indexing statistics.

#### Request
```http
GET /v1/stats/chain
```

#### Response
```json
{
  "latest_block": 18500000,
  "total_blocks": 18500000,
  "total_transactions": 125430000,
  "indexer_lag_blocks": 0,
  "indexer_lag_seconds": 2,
  "last_updated": "2025-10-31T10:00:00Z"
}
```

#### Example
```bash
curl "http://localhost:8080/v1/stats/chain"
```

---

## WebSocket Streaming

Real-time updates for blocks and transactions via WebSocket.

### Connect to WebSocket

#### Connection
```
ws://localhost:8080/v1/stream
```

### Subscribe to Channel

Send a JSON message to subscribe to a channel:

```json
{
  "action": "subscribe",
  "channel": "blocks"
}
```

**Available Channels:**
- `blocks` - New blocks as they're mined
- `transactions` - New transactions as they're confirmed

### Unsubscribe from Channel

```json
{
  "action": "unsubscribe",
  "channel": "blocks"
}
```

### Message Format

#### Block Update
```json
{
  "type": "block",
  "data": {
    "height": 18500000,
    "hash": "0x1234567890abcdef...",
    "timestamp": 1698768000,
    "tx_count": 150,
    "miner": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
  }
}
```

#### Transaction Update
```json
{
  "type": "transaction",
  "data": {
    "hash": "0xabcdef1234567890...",
    "block_height": 18500000,
    "from_addr": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
    "to_addr": "0x1234567890abcdef...",
    "value_wei": "1000000000000000000"
  }
}
```

#### Error Message
```json
{
  "type": "error",
  "message": "Invalid action or channel"
}
```

### WebSocket Example (JavaScript)

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/v1/stream');

// Handle connection open
ws.onopen = () => {
  console.log('Connected to WebSocket');

  // Subscribe to new blocks
  ws.send(JSON.stringify({
    action: 'subscribe',
    channel: 'blocks'
  }));

  // Subscribe to new transactions
  ws.send(JSON.stringify({
    action: 'subscribe',
    channel: 'transactions'
  }));
};

// Handle incoming messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  if (data.type === 'block') {
    console.log('New block:', data.data);
  } else if (data.type === 'transaction') {
    console.log('New transaction:', data.data);
  } else if (data.type === 'error') {
    console.error('Error:', data.message);
  }
};

// Handle connection close
ws.onclose = () => {
  console.log('Disconnected from WebSocket');
};

// Handle errors
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

// Unsubscribe from a channel
function unsubscribe(channel) {
  ws.send(JSON.stringify({
    action: 'unsubscribe',
    channel: channel
  }));
}

// Close connection when done
function disconnect() {
  ws.close();
}
```

### WebSocket Example (cURL with websocat)

```bash
# Install websocat: https://github.com/vi/websocat
# Or: brew install websocat

# Connect and subscribe
echo '{"action":"subscribe","channel":"blocks"}' | websocat ws://localhost:8080/v1/stream
```

---

## Metrics

### Prometheus Metrics

Get Prometheus-formatted metrics for monitoring.

#### Request
```http
GET /metrics
```

#### Response
```
# HELP blockchain_api_requests_total Total number of API requests
# TYPE blockchain_api_requests_total counter
blockchain_api_requests_total{method="GET",endpoint="/v1/blocks"} 1234

# HELP blockchain_api_request_duration_seconds Request duration in seconds
# TYPE blockchain_api_request_duration_seconds histogram
blockchain_api_request_duration_seconds_bucket{method="GET",endpoint="/v1/blocks",le="0.1"} 1000
blockchain_api_request_duration_seconds_bucket{method="GET",endpoint="/v1/blocks",le="0.5"} 1200
...
```

#### Example
```bash
curl "http://localhost:8080/metrics"
```

---

## Common Patterns

### Pagination Best Practices

```bash
# Start with first page
curl "http://localhost:8080/v1/blocks?limit=25&offset=0"

# Calculate next page: offset = offset + limit
curl "http://localhost:8080/v1/blocks?limit=25&offset=25"

# Continue until offset >= total
```

### Error Handling

```javascript
async function getBlock(height) {
  try {
    const response = await fetch(`http://localhost:8080/v1/blocks/${height}`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error);
    }

    return await response.json();
  } catch (error) {
    console.error('Failed to fetch block:', error);
    throw error;
  }
}
```

### Rate Limiting (Future)

When rate limiting is implemented, you'll receive:

```json
{
  "error": "Rate limit exceeded",
  "retry_after": 60
}
```

---

## Data Types

### Address Format
- Must start with `0x`
- Followed by exactly 40 hexadecimal characters
- Example: `0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0`

### Hash Format
- Must start with `0x`
- Followed by exactly 64 hexadecimal characters
- Example: `0x1234567890abcdef...` (64 chars after 0x)

### Wei to Ether Conversion

Wei is the smallest unit of Ether. To convert:

```javascript
// Wei to Ether
const ether = weiValue / 1e18;

// Ether to Wei
const wei = etherValue * 1e18;
```

---

## Support & Feedback

- **GitHub Issues**: [Report bugs or request features](https://github.com/hieutt50/go-blockchain-explorer/issues)
- **Documentation**: [Full README](README.md)
- **Docker Guide**: [DOCKER.md](DOCKER.md)
- **Quick Start**: [QUICKSTART.md](QUICKSTART.md)

---

## Changelog

### Version 1.0
- Initial API release
- All REST endpoints implemented
- WebSocket streaming support
- Pagination support for all list endpoints
- Health check and metrics endpoints
