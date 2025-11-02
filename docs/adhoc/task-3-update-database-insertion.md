# Task 3: Update Database Insertion for Transactions

**Status:** Ready for Implementation
**Priority:** HIGH
**Estimated Time:** 30 minutes
**Dependencies:** Task 1 (Domain model), Task 2 (Parsing)
**Blocks:** Task 4, 5, 6

---

## Objective

Update `InsertBlock()` method to store transactions in the database along with blocks, maintaining referential integrity and transaction atomicity.

---

## Current State

**File:** `internal/store/adapter.go:74-110`

```go
func (a *IndexerAdapter) InsertBlock(ctx context.Context, block *index.Block) error {
    tx, _ := a.pool.Pool.Begin(ctx)
    defer tx.Rollback(ctx)

    // Insert block
    _, err = tx.Exec(ctx, `INSERT INTO blocks ...`)

    // Note: Transactions are in block.Transactions field, but not used in MVP
    // Full transaction processing would happen here

    return tx.Commit(ctx)
}
```

**Problem:** Comment says "not used in MVP" - transactions are parsed but discarded.

---

## Target State

**File:** `internal/store/adapter.go:74-110`

Full transaction insertion with proper fee calculation:

```go
func (a *IndexerAdapter) InsertBlock(ctx context.Context, block *index.Block) error {
    tx, err := a.pool.Pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // 1. Insert block (existing)
    _, err = tx.Exec(ctx, `
        INSERT INTO blocks (height, hash, parent_hash, miner, gas_used, gas_limit, timestamp, tx_count, orphaned)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (height) DO UPDATE SET
            hash = EXCLUDED.hash,
            parent_hash = EXCLUDED.parent_hash,
            miner = EXCLUDED.miner,
            gas_used = EXCLUDED.gas_used,
            gas_limit = EXCLUDED.gas_limit,
            timestamp = EXCLUDED.timestamp,
            tx_count = EXCLUDED.tx_count,
            orphaned = EXCLUDED.orphaned,
            updated_at = NOW()
    `, block.Height, block.Hash, block.ParentHash, block.Miner,
        block.GasUsed, 0, block.Timestamp, block.TxCount, false)

    if err != nil {
        return fmt.Errorf("failed to insert block %d: %w", block.Height, err)
    }

    // 2. Insert transactions (NEW)
    for _, txn := range block.Transactions {
        if err := insertTransaction(ctx, tx, txn, block.Height); err != nil {
            return fmt.Errorf("failed to insert transaction in block %d: %w", block.Height, err)
        }
    }

    // 3. Commit all changes atomically
    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("failed to commit block %d: %w", block.Height, err)
    }

    return nil
}
```

---

## Implementation Steps

### Step 1: Add math/big Import

**File:** `internal/store/adapter.go` (top of file)

```go
import (
    // ... existing imports ...
    "math/big"
)
```

### Step 2: Create insertTransaction Helper

**File:** `internal/store/adapter.go` (add new function)

```go
// insertTransaction inserts a single transaction into the database
// Uses the provided database transaction for atomicity with block insertion
func insertTransaction(ctx context.Context, dbTx pgx.Tx, txn index.Transaction, blockHeight uint64) error {
    // Calculate fee: gas_used * gas_price (in Wei)
    // Convert to big.Int for calculation, then back to string for storage
    gasUsedBig := new(big.Int).SetUint64(txn.GasUsed)
    gasPriceBig := new(big.Int).SetUint64(txn.GasPrice)
    feeWei := new(big.Int).Mul(gasUsedBig, gasPriceBig).String()

    _, err := dbTx.Exec(ctx, `
        INSERT INTO transactions
        (hash, block_height, tx_index, from_addr, to_addr, value_wei,
         fee_wei, gas_used, gas_price, nonce, success)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (hash) DO NOTHING
    `, txn.Hash, blockHeight, txn.TxIndex, txn.FromAddr, txn.ToAddr,
       txn.ValueWei, feeWei, txn.GasUsed, txn.GasPrice, txn.Nonce, txn.Success)

    return err
}
```

### Step 3: Update InsertBlock

**File:** `internal/store/adapter.go:74-110` (replace existing implementation)

Update the section after block insert:

```go
    // Insert block (existing code)
    _, err = tx.Exec(ctx, `INSERT INTO blocks ...`)
    if err != nil {
        return fmt.Errorf("failed to insert block %d: %w", block.Height, err)
    }

    // Insert transactions (NEW - replace TODO comment)
    for _, txn := range block.Transactions {
        if err := insertTransaction(ctx, tx, txn, block.Height); err != nil {
            return fmt.Errorf("failed to insert transaction %x in block %d: %w",
                txn.Hash, block.Height, err)
        }
    }

    // Commit transaction
    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("failed to commit block %d: %w", block.Height, err)
    }

    return nil
```

---

## Edge Cases to Handle

### 1. Contract Creation (to_addr = nil)
**Scenario:** Transaction creates new contract, `tx.To()` is nil

**Handling:**
```go
var toAddr []byte
if tx.To() != nil {
    toAddr = tx.To().Bytes()
}
// toAddr will be nil, database column allows NULL
```

**Database:** Column `to_addr BYTEA` (nullable)

### 2. Zero Value Transfer
**Scenario:** Transaction transfers 0 ETH (common for contract calls)

**Handling:**
```go
ValueWei: tx.Value().String()  // "0" is valid
```

**Database:** Stores "0", frontend displays "0.0000 ETH"

### 3. High Nonce Values
**Scenario:** Account has sent 1M+ transactions

**Handling:**
```go
Nonce: tx.Nonce()  // uint64 supports up to 2^64-1
```

**Database:** Column `nonce BIGINT` supports large values

### 4. Duplicate Transaction Hash
**Scenario:** Transaction already exists (replay during reorg recovery)

**Handling:**
```sql
ON CONFLICT (hash) DO NOTHING
```

**Result:** Silently skips duplicate, no error

### 5. Transaction Insertion Failure
**Scenario:** Database error during transaction insert

**Handling:**
```go
if err := insertTransaction(...); err != nil {
    return err  // Triggers tx.Rollback() in defer
}
```

**Result:** Entire block rolled back, maintains consistency

---

## Database Transaction Flow

```
BEGIN;
  INSERT INTO blocks (...) VALUES (...);          -- 1 row
  INSERT INTO transactions (...) VALUES (...);    -- N rows (1 per tx)
  INSERT INTO transactions (...) VALUES (...);
  ...
COMMIT;  -- Atomic: all or nothing
```

**If any INSERT fails:**
- ROLLBACK triggered by `defer tx.Rollback(ctx)`
- No partial data in database
- Error propagated to caller
- Backfill or live-tail can retry

---

## Performance Considerations

### Insertion Speed

**Before (blocks only):**
- 1 INSERT per block
- ~10ms per block insertion

**After (blocks + transactions):**
- 1 + N INSERTs per block (N = transaction count)
- Avg 50 transactions per block on Ethereum
- ~50-100ms per block insertion (still fast)

**Impact on Backfill:**
- 5000 blocks × 100ms = 500 seconds = 8.3 minutes
- Still reasonable, though above 5-minute target
- **Mitigation:** Use batch insert optimization (future)

### Optimization Opportunities (Future)

1. **Batch Insert with COPY:**
   ```go
   tx.CopyFrom(ctx, pgx.Identifier{"transactions"}, columns, rows)
   ```
   - Much faster than individual INSERTs
   - Requires refactoring to collect all transactions first

2. **Prepared Statements:**
   ```go
   stmt := tx.Prepare(ctx, "insert_tx", "INSERT INTO transactions ...")
   stmt.Exec(...)
   ```
   - Reduces query parsing overhead
   - Useful for high-volume inserts

---

## Testing

### Unit Tests

**File:** `internal/store/adapter_test.go` (create new file)

```go
func TestInsertBlock_WithTransactions(t *testing.T) {
    // Mock database connection
    // Create block with 3 transactions
    // Call InsertBlock
    // Verify 1 block + 3 transactions inserted
}

func TestInsertBlock_ContractCreation(t *testing.T) {
    // Transaction with ToAddr = nil
    // Verify NULL stored in to_addr column
}

func TestInsertBlock_DuplicateTransaction(t *testing.T) {
    // Insert block twice
    // Verify ON CONFLICT DO NOTHING works
}

func TestInsertBlock_TransactionFailure(t *testing.T) {
    // Mock transaction insert error
    // Verify entire block rolled back
    // Verify block NOT in database
}
```

### Integration Test

**File:** `internal/store/postgres_integration_test.go` (add test)

```go
func TestInsertBlock_WithTransactions_Integration(t *testing.T) {
    // Use real test database
    // Create block with transactions
    // Insert block
    // Query transactions table
    // Verify all transactions present
    // Verify foreign keys correct
}
```

---

## Acceptance Criteria

- [ ] `insertTransaction()` helper function implemented
- [ ] Fee calculation correct (gas_used × gas_price)
- [ ] `InsertBlock()` inserts all transactions
- [ ] Contract creation (nil to_addr) handled
- [ ] Database transaction used for atomicity
- [ ] ON CONFLICT DO NOTHING prevents duplicate errors
- [ ] Error handling rolls back entire block on failure
- [ ] Code compiles without errors
- [ ] Unit tests added and passing
- [ ] Integration test verifies database insertion

---

## Verification Commands

```bash
# Compile
go build ./internal/store

# Test
go test ./internal/store -v

# Integration test (requires test database)
go test ./internal/store -tags=integration -v

# Check transaction count after backfill
psql -d blockchain_explorer -c "SELECT COUNT(*) FROM transactions;"
```

---

## Rollback Plan

If issues arise:

1. **Revert code changes:**
   ```bash
   git checkout internal/store/adapter.go
   ```

2. **Clear database:**
   ```sql
   TRUNCATE transactions CASCADE;
   ```

3. **Re-run backfill** without transaction extraction

---

## Next Task

**Task 4:** Add Block Transactions API Endpoint
- Create `/v1/blocks/{height}/transactions` endpoint
- Enable frontend to fetch transactions for specific blocks
