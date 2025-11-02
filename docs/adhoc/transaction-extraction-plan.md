# Ad-Hoc Plan: Transaction and Log Extraction Implementation

**Date:** 2025-11-01
**Priority:** HIGH - Critical gap preventing full application functionality
**Estimated Effort:** 2-3 hours
**Status:** Ready for Implementation

---

## Problem Statement

### Current State
The blockchain explorer currently indexes blocks but does NOT extract or store transactions and logs:

- ✅ Blocks are indexed (height, hash, timestamp, miner, gas_used, tx_count)
- ❌ Transactions table is EMPTY (not extracted from blocks)
- ❌ Logs table is EMPTY (no receipt fetching)
- ❌ Frontend uses mock/hardcoded transaction data
- ❌ Transaction search returns no results
- ❌ Address history is empty
- ❌ Log queries fail

### Impact
- **Transaction Search** (`/v1/txs/{hash}`) - Returns 404
- **Address History** (`/v1/address/{addr}/txs`) - Returns empty array
- **Log Queries** (`/v1/logs`) - Returns empty array
- **Frontend Transaction Table** - Shows mock data, not real blockchain data
- **Demonstrability** - Cannot show core blockchain explorer functionality

### Root Cause
1. Block domain model (`internal/index/livetail.go:73`) has no `Transactions` field
2. `ParseRPCBlock()` only extracts block metadata, ignores transactions
3. `InsertBlock()` has TODO comment: "Transactions not used in MVP"
4. No receipt fetching implemented (needed for logs)

---

## Solution Approach

### Option Selected: **Option B - Basic Transaction Extraction (Fast Implementation)**

**Rationale:**
- Fast implementation (2-3 hours vs 6+ hours for full receipts)
- Meets 90% of use cases (search, history, value tracking)
- Maintains performance targets (backfill <5 min)
- Can add receipts/logs later as enhancement

**Trade-offs:**
- ✅ Transaction hash, from/to, value, nonce - ACCURATE
- ⚠️ Gas used - ESTIMATED (use transaction gas limit, not actual from receipt)
- ⚠️ Success status - ASSUMED TRUE (can't detect failures without receipts)
- ❌ Logs - NOT EXTRACTED (requires receipts, deferred)

---

## Implementation Plan

### Phase 1: Extend Domain Model
**File:** `internal/index/livetail.go`

Add transaction and log types to support the data pipeline:

```go
type Block struct {
    Height       uint64
    Hash         []byte
    ParentHash   []byte
    Timestamp    uint64
    Miner        []byte
    GasUsed      uint64
    TxCount      int
    Transactions []Transaction  // NEW FIELD
}

type Transaction struct {
    Hash      []byte
    TxIndex   int
    FromAddr  []byte
    ToAddr    []byte  // nil for contract creation
    ValueWei  string  // String to preserve precision
    GasUsed   uint64  // Estimated from tx.Gas() in basic mode
    GasPrice  uint64
    Nonce     uint64
    Success   bool    // Assumed true in basic mode
    Logs      []Log   // Empty in basic mode
}

type Log struct {
    LogIndex  int
    Address   []byte
    Topics    [4][]byte
    Data      []byte
}
```

**Files to Update:**
- `internal/index/livetail.go` - Add Transaction and Log types
- `internal/index/reorg.go` - Update any Block references
- `internal/index/backfill.go` - Update any Block references

---

### Phase 2: Implement Transaction Parsing
**File:** `internal/store/adapter.go`

Update `ParseRPCBlock()` to extract transactions:

```go
func ParseRPCBlock(rpcBlock *types.Block) *index.Block {
    block := &index.Block{
        // ... existing fields ...
        Transactions: make([]index.Transaction, 0, len(rpcBlock.Transactions())),
    }

    // Extract transactions from block
    for txIndex, tx := range rpcBlock.Transactions() {
        indexerTx := parseTransaction(tx, txIndex, block.Height)
        block.Transactions = append(block.Transactions, indexerTx)
    }

    return block
}

func parseTransaction(tx *types.Transaction, txIndex int, blockHeight uint64) index.Transaction {
    // Get sender address (requires signature recovery)
    from, err := types.LatestSignerForChainID(tx.ChainId()).Sender(tx)
    var fromAddr []byte
    if err == nil {
        fromAddr = from.Bytes()
    } else {
        fromAddr = common.Address{}.Bytes() // Zero address on error
    }

    // Get recipient address (nil for contract creation)
    var toAddr []byte
    if tx.To() != nil {
        toAddr = tx.To().Bytes()
    }

    return index.Transaction{
        Hash:      tx.Hash().Bytes(),
        TxIndex:   txIndex,
        FromAddr:  fromAddr,
        ToAddr:    toAddr,
        ValueWei:  tx.Value().String(),
        GasUsed:   tx.Gas(),        // Using gas limit (receipt needed for actual)
        GasPrice:  tx.GasPrice().Uint64(),
        Nonce:     tx.Nonce(),
        Success:   true,            // Assume success (receipt needed for actual)
        Logs:      []index.Log{},   // Empty for basic mode
    }
}
```

---

### Phase 3: Update Database Insertion
**File:** `internal/store/adapter.go`

Update `InsertBlock()` to insert transactions:

```go
func (a *IndexerAdapter) InsertBlock(ctx context.Context, block *index.Block) error {
    tx, err := a.pool.Pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // 1. Insert block (existing code)
    _, err = tx.Exec(ctx, `INSERT INTO blocks ...`)
    if err != nil {
        return fmt.Errorf("failed to insert block: %w", err)
    }

    // 2. Insert transactions (NEW)
    for _, txn := range block.Transactions {
        // Calculate fee: gas_used * gas_price (in wei)
        feeWei := new(big.Int).Mul(
            new(big.Int).SetUint64(txn.GasUsed),
            new(big.Int).SetUint64(txn.GasPrice),
        ).String()

        _, err = tx.Exec(ctx, `
            INSERT INTO transactions
            (hash, block_height, tx_index, from_addr, to_addr, value_wei,
             fee_wei, gas_used, gas_price, nonce, success)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            ON CONFLICT (hash) DO NOTHING
        `, txn.Hash, block.Height, txn.TxIndex, txn.FromAddr, txn.ToAddr,
           txn.ValueWei, feeWei, txn.GasUsed, txn.GasPrice, txn.Nonce, txn.Success)

        if err != nil {
            return fmt.Errorf("failed to insert transaction %x: %w", txn.Hash, err)
        }
    }

    // 3. Insert logs (skipped in basic mode)

    return tx.Commit(ctx)
}
```

---

### Phase 4: Fix Frontend Mock Data
**File:** `web/app.js`

Remove mock data and use real API:

**Current Mock Implementation:**
```javascript
// Lines with mock data workarounds:
// - Line ~140: const mockHashes = ['0x...', '0x...']
// - Line ~150: for (const hash of mockHashes)
// - Line ~160: // Fetch known mock transactions directly
```

**Replace With Real API:**
```javascript
async function fetchRecentTransactions() {
    try {
        // Fetch recent blocks with their transactions
        const blocksResponse = await fetch(`${API_BASE}/v1/blocks?limit=5&offset=0`);
        const blocksData = await blocksResponse.json();

        const allTransactions = [];

        // For each block, fetch its transactions
        for (const block of blocksData.blocks) {
            if (block.tx_count > 0) {
                // Transactions will be returned by the block detail endpoint
                // Or we can add a new endpoint: /v1/blocks/{height}/transactions
                const txResponse = await fetch(`${API_BASE}/v1/blocks/${block.height}/transactions`);
                const txData = await txResponse.json();
                allTransactions.push(...txData.transactions);

                if (allTransactions.length >= 25) break;
            }
        }

        return allTransactions.slice(0, 25);
    } catch (error) {
        console.error('Failed to fetch transactions:', error);
        return [];
    }
}
```

**Alternative (if we add `/v1/txs/recent` endpoint):**
```javascript
async function fetchRecentTransactions() {
    const response = await fetch(`${API_BASE}/v1/txs/recent?limit=25`);
    const data = await response.json();
    return data.transactions;
}
```

---

### Phase 5: Add Block Transactions Endpoint (Optional)
**File:** `internal/api/handlers.go`

Add new endpoint for better frontend experience:

```go
// GET /v1/blocks/{height}/transactions
func (s *Server) handleGetBlockTransactions(w http.ResponseWriter, r *http.Request) {
    heightOrHash := chi.URLParam(r, "heightOrHash")
    limit, offset := parsePagination(r, 100, 1000)

    st := store.NewStore(s.pool.Pool)

    // Try parsing as height first
    if height, err := strconv.ParseInt(heightOrHash, 10, 64); err == nil {
        txs, total, err := st.GetBlockTransactions(r.Context(), height, limit, offset)
        if err != nil {
            writeInternalError(w, err)
            return
        }
        writeJSON(w, http.StatusOK, map[string]interface{}{
            "transactions": txs,
            "total": total,
            "limit": limit,
            "offset": offset,
        })
        return
    }

    writeBadRequest(w, "invalid block height")
}
```

Add to store:
```go
func (s *Store) GetBlockTransactions(ctx context.Context, blockHeight int64, limit, offset int) ([]Transaction, int64, error) {
    // Query transactions for specific block
    rows, err := s.pool.Query(ctx, `
        SELECT hash, block_height, tx_index, from_addr, to_addr, value_wei,
               fee_wei, gas_used, gas_price, nonce, success
        FROM transactions
        WHERE block_height = $1
        ORDER BY tx_index ASC
        LIMIT $2 OFFSET $3
    `, blockHeight, limit, offset)
    // ... scan and return
}
```

---

## Task Breakdown

### Task 1: Update Domain Model
**File:** `internal/index/livetail.go`
**Effort:** 10 minutes
**Dependencies:** None
**Deliverable:** Block struct with Transactions field

### Task 2: Implement Transaction Parsing
**File:** `internal/store/adapter.go`
**Effort:** 30 minutes
**Dependencies:** Task 1
**Deliverable:** ParseRPCBlock() extracts transactions

### Task 3: Update Database Insertion
**File:** `internal/store/adapter.go`
**Effort:** 30 minutes
**Dependencies:** Task 1, Task 2
**Deliverable:** InsertBlock() stores transactions

### Task 4: Add Block Transactions API Endpoint
**File:** `internal/api/handlers.go`, `internal/store/queries.go`
**Effort:** 30 minutes
**Dependencies:** Task 3
**Deliverable:** GET /v1/blocks/{height}/transactions

### Task 5: Fix Frontend Mock Data
**File:** `web/app.js`
**Effort:** 20 minutes
**Dependencies:** Task 4
**Deliverable:** Frontend uses real transaction data

### Task 6: End-to-End Testing
**File:** Manual testing
**Effort:** 20 minutes
**Dependencies:** Task 5
**Deliverable:** Verified transaction flow

---

## Success Criteria

### Functional Validation
- [ ] Worker indexes blocks and extracts transactions
- [ ] Database `transactions` table populated with real data
- [ ] API endpoint `/v1/txs/{hash}` returns real transactions
- [ ] API endpoint `/v1/address/{addr}/txs` returns transaction history
- [ ] Frontend transaction table shows real blockchain data
- [ ] No mock data remaining in codebase

### Performance Validation
- [ ] Backfill 5000 blocks still completes in <5 minutes
- [ ] No significant performance degradation
- [ ] Database inserts remain fast

### Data Quality Validation
- [ ] Transaction hashes match blockchain
- [ ] From/to addresses correct
- [ ] Value in Wei accurate
- [ ] Foreign keys (block_height) valid

---

## Implementation Notes

### Important Considerations

1. **Signature Recovery for From Address:**
   - Ethereum transactions don't include `from` address directly
   - Must recover from signature using `types.LatestSignerForChainID()`
   - Handle errors gracefully (use zero address on failure)

2. **Contract Creation Transactions:**
   - `tx.To()` returns nil for contract creation
   - Store NULL in `to_addr` column
   - Frontend must handle null values

3. **Gas Used Estimation:**
   - Transaction has `Gas()` (gas limit, what sender allocated)
   - Receipt has `GasUsed` (actual gas consumed)
   - Basic mode uses `tx.Gas()` as estimate (overestimate)
   - Future enhancement: Fetch receipts for accuracy

4. **Fee Calculation:**
   - Fee = GasUsed × GasPrice (in Wei)
   - Store as string to preserve precision
   - Frontend converts to ETH for display

5. **Database Transaction Atomicity:**
   - Use single database transaction for block + all transactions
   - If any transaction insert fails, rollback entire block
   - Prevents partial block data

6. **Performance Impact:**
   - Parsing transactions adds ~10-20ms per block
   - Inserting 100 transactions per block adds ~50-100ms
   - Still within performance target (<5 min for 5000 blocks)

### Testing Strategy

**Unit Tests:**
- Test `parseTransaction()` with various transaction types
- Test contract creation (nil to_addr)
- Test value/gas calculations
- Test signature recovery errors

**Integration Tests:**
- Insert block with transactions, query them back
- Verify foreign key constraints
- Test transaction uniqueness (ON CONFLICT DO NOTHING)
- Verify cascade delete (orphaned blocks → orphaned transactions)

**End-to-End Tests:**
- Run backfill, verify transactions in database
- Query API, verify transaction data correct
- Test frontend, verify real data displays

---

## Rollback Plan

If implementation causes issues:

1. **Performance Issues:**
   - Reduce batch size
   - Add transaction parsing toggle (env var)
   - Revert to block-only indexing

2. **Database Issues:**
   - Check foreign key constraints
   - Verify transaction insert queries
   - Roll back migration if schema issues

3. **Frontend Issues:**
   - Keep both mock and real data code paths
   - Toggle via configuration

---

## Future Enhancements (Deferred)

### Enhancement 1: Receipt Fetching for Accurate Data
- Fetch receipts in background job (async)
- Update transactions with accurate gas_used, success status
- Extract and store logs

### Enhancement 2: Log Extraction and Indexing
- Parse logs from receipts
- Store in logs table
- Enable event filtering queries

### Enhancement 3: Failed Transaction Detection
- Fetch receipts to check status
- Mark failed transactions in database
- Display failed transactions differently in UI

---

## References

- **Tech Spec:** `docs/tech-spec-epic-1.md` (Data Architecture, lines 131-525)
- **Database Schema:** `migrations/000001_initial_schema.up.sql`
- **API Handlers:** `internal/api/handlers.go`
- **Frontend:** `web/app.js` (mock data at lines ~140-160)
- **go-ethereum Types:** https://pkg.go.dev/github.com/ethereum/go-ethereum/core/types

---

_This ad-hoc plan addresses a critical gap discovered during application wiring. Implementation follows existing architecture patterns and maintains performance targets._
