# Task 4: Add Block Transactions API Endpoint

**Status:** Ready for Implementation
**Priority:** MEDIUM
**Estimated Time:** 30 minutes
**Dependencies:** Task 3 (Transactions in database)
**Blocks:** Task 5

---

## Objective

Add a new REST API endpoint to fetch all transactions for a specific block, enabling the frontend to display transaction details without mock data.

---

## Endpoint Specification

### Route
```
GET /v1/blocks/{heightOrHash}/transactions
```

### Parameters
- `heightOrHash` (path) - Block height (number) or block hash (0x + 64 hex)
- `limit` (query) - Max results (default: 100, max: 1000)
- `offset` (query) - Pagination offset (default: 0)

### Response Format
```json
{
  "transactions": [
    {
      "hash": "0xabc123...",
      "block_height": 1234,
      "tx_index": 0,
      "from_addr": "0xdef456...",
      "to_addr": "0x789abc...",
      "value_wei": "1000000000000000000",
      "fee_wei": "21000000000000",
      "gas_used": "21000",
      "gas_price": "1000000000",
      "nonce": 5,
      "success": true,
      "created_at": "2025-11-01T10:30:00Z"
    }
  ],
  "total": 47,
  "limit": 100,
  "offset": 0
}
```

### Error Responses
- `400 Bad Request` - Invalid block height or hash format
- `404 Not Found` - Block doesn't exist
- `500 Internal Server Error` - Database error

---

## Implementation Steps

### Step 1: Add Store Method

**File:** `internal/store/queries.go` (add new method)

```go
// GetBlockTransactions returns paginated transactions for a specific block
func (s *Store) GetBlockTransactions(ctx context.Context, blockHeight int64, limit, offset int) ([]Transaction, int64, error) {
    // Get total count of transactions in this block
    var total int64
    err := s.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM transactions
        WHERE block_height = $1
    `, blockHeight).Scan(&total)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to count block transactions: %w", err)
    }

    // Get paginated transactions
    rows, err := s.pool.Query(ctx, `
        SELECT hash, block_height, tx_index, from_addr, to_addr, value_wei,
               fee_wei, gas_used, gas_price, nonce, success, created_at
        FROM transactions
        WHERE block_height = $1
        ORDER BY tx_index ASC
        LIMIT $2 OFFSET $3
    `, blockHeight, limit, offset)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to query block transactions: %w", err)
    }
    defer rows.Close()

    txs := make([]Transaction, 0, limit)
    for rows.Next() {
        var tx Transaction
        var hashBytes, fromBytes []byte
        var toAddr *[]byte

        err := rows.Scan(&hashBytes, &tx.BlockHeight, &tx.TxIndex, &fromBytes, &toAddr,
            &tx.ValueWei, &tx.FeeWei, &tx.GasUsed, &tx.GasPrice, &tx.Nonce, &tx.Success, &tx.CreatedAt)
        if err != nil {
            return nil, 0, fmt.Errorf("failed to scan transaction: %w", err)
        }

        tx.Hash = "0x" + hex.EncodeToString(hashBytes)
        tx.FromAddr = "0x" + hex.EncodeToString(fromBytes)

        if toAddr != nil {
            toAddrStr := "0x" + hex.EncodeToString(*toAddr)
            tx.ToAddr = &toAddrStr
        }

        txs = append(txs, tx)
    }

    if err := rows.Err(); err != nil {
        return nil, 0, fmt.Errorf("error iterating transactions: %w", err)
    }

    return txs, total, nil
}
```

### Step 2: Add API Handler

**File:** `internal/api/handlers.go` (add new handler function)

```go
// handleGetBlockTransactions handles GET /v1/blocks/{heightOrHash}/transactions
// Returns all transactions for a specific block with pagination
func (s *Server) handleGetBlockTransactions(w http.ResponseWriter, r *http.Request) {
    heightOrHash := chi.URLParam(r, "heightOrHash")

    // Parse pagination (default limit=100, max=1000)
    limit, offset := parsePagination(r, 100, 1000)

    // Try parsing as block height (number)
    if height, err := strconv.ParseInt(heightOrHash, 10, 64); err == nil {
        st := store.NewStore(s.pool.Pool)

        txs, total, err := st.GetBlockTransactions(r.Context(), height, limit, offset)
        if err != nil {
            writeInternalError(w, err)
            return
        }

        response := map[string]interface{}{
            "transactions": txs,
            "total":        total,
            "limit":        limit,
            "offset":       offset,
        }

        writeJSON(w, http.StatusOK, response)
        return
    }

    // Could also support block hash lookup (future enhancement)
    writeBadRequest(w, "invalid block height")
}
```

### Step 3: Register Route

**File:** `internal/api/server.go` (update router configuration)

Find the route registration section and add:

```go
// Block routes
r.Get("/v1/blocks", s.handleListBlocks)
r.Get("/v1/blocks/{heightOrHash}", s.handleGetBlock)
r.Get("/v1/blocks/{heightOrHash}/transactions", s.handleGetBlockTransactions)  // NEW
```

---

## Alternative: Recent Transactions Endpoint

**If frontend prefers a simpler API:**

### Route
```
GET /v1/transactions/recent
```

### Implementation

**File:** `internal/store/queries.go`

```go
// GetRecentTransactions returns the N most recent transactions across all blocks
func (s *Store) GetRecentTransactions(ctx context.Context, limit int) ([]Transaction, error) {
    rows, err := s.pool.Query(ctx, `
        SELECT t.hash, t.block_height, t.tx_index, t.from_addr, t.to_addr,
               t.value_wei, t.fee_wei, t.gas_used, t.gas_price, t.nonce, t.success, t.created_at
        FROM transactions t
        INNER JOIN blocks b ON t.block_height = b.height
        WHERE b.orphaned = FALSE
        ORDER BY t.block_height DESC, t.tx_index DESC
        LIMIT $1
    `, limit)
    // ... scan and return
}
```

**File:** `internal/api/handlers.go`

```go
func (s *Server) handleRecentTransactions(w http.ResponseWriter, r *http.Request) {
    limitParam := r.URL.Query().Get("limit")
    limit := 25
    if limitParam != "" {
        if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
            limit = l
        }
    }

    st := store.NewStore(s.pool.Pool)
    txs, err := st.GetRecentTransactions(r.Context(), limit)
    if err != nil {
        writeInternalError(w, err)
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "transactions": txs,
        "total":        len(txs),
    })
}
```

Register route:
```go
r.Get("/v1/transactions/recent", s.handleRecentTransactions)
```

---

## Testing

### Unit Tests

**File:** `internal/api/handlers_test.go`

```go
func TestHandleGetBlockTransactions_Success(t *testing.T) {
    // Mock store with block that has 3 transactions
    // Call handler
    // Verify response has 3 transactions
    // Verify pagination metadata correct
}

func TestHandleGetBlockTransactions_InvalidHeight(t *testing.T) {
    // Call with invalid height "abc"
    // Verify 400 Bad Request
}

func TestHandleGetBlockTransactions_BlockNotFound(t *testing.T) {
    // Call with height 999999 (doesn't exist)
    // Verify 404 Not Found or empty array (design choice)
}
```

### Integration Test

**File:** `internal/api/handlers_integration_test.go` (create if needed)

```go
func TestGetBlockTransactions_Integration(t *testing.T) {
    // Setup test database
    // Insert block with 10 transactions
    // HTTP GET /v1/blocks/123/transactions
    // Verify response contains all 10 transactions
    // Verify transactions ordered by tx_index
}
```

### Manual Testing

```bash
# Start API server
go run cmd/api/main.go

# Test endpoint
curl http://localhost:8080/v1/blocks/1234/transactions?limit=10

# Expected response:
{
  "transactions": [...],
  "total": 47,
  "limit": 10,
  "offset": 0
}
```

---

## Acceptance Criteria

- [ ] `GetBlockTransactions()` method added to Store
- [ ] `handleGetBlockTransactions()` handler implemented
- [ ] Route registered in server.go
- [ ] Pagination support (limit, offset)
- [ ] Returns transactions ordered by tx_index
- [ ] Handles invalid block height gracefully
- [ ] Returns empty array for blocks with 0 transactions
- [ ] Unit tests added and passing
- [ ] Integration test verifies database query
- [ ] Manual curl test succeeds

---

## Frontend Integration

**How frontend will use this:**

```javascript
// web/app.js
async function fetchRecentTransactions() {
    try {
        // Fetch latest blocks
        const blocksResp = await fetch(`${API_BASE}/v1/blocks?limit=3`);
        const blocksData = await blocksResp.json();

        const allTxs = [];

        // Fetch transactions from each block
        for (const block of blocksData.blocks) {
            if (block.tx_count > 0) {
                const txResp = await fetch(
                    `${API_BASE}/v1/blocks/${block.height}/transactions?limit=25`
                );
                const txData = await txResp.json();
                allTxs.push(...txData.transactions);

                if (allTxs.length >= 25) break;
            }
        }

        return allTxs.slice(0, 25);
    } catch (error) {
        console.error('Failed to fetch transactions:', error);
        return [];
    }
}
```

---

## Next Task

**Task 5:** Fix Frontend Mock Data
- Remove hardcoded transaction hashes
- Use new `/v1/blocks/{height}/transactions` endpoint
- Update `fetchRecentTransactions()` function
