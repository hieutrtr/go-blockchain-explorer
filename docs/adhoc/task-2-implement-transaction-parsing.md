# Task 2: Implement Transaction Parsing

**Status:** Ready for Implementation
**Priority:** HIGH
**Estimated Time:** 30 minutes
**Dependencies:** Task 1 (Domain model updated)
**Blocks:** Task 3, 4, 5, 6

---

## Objective

Implement transaction extraction from go-ethereum `types.Block` into our `index.Block` domain model, enabling the indexer to carry transaction data through the pipeline.

---

## Current State

**File:** `internal/store/adapter.go:138-156`

```go
func ParseRPCBlock(rpcBlock *types.Block) *index.Block {
    return &index.Block{
        Height:     rpcBlock.NumberU64(),
        Hash:       rpcBlock.Hash().Bytes(),
        // ... other fields ...
        TxCount:    len(rpcBlock.Transactions()),
        // Missing: Actually parse rpcBlock.Transactions()
    }
}
```

**Problem:** Function counts transactions but doesn't extract their data.

---

## Target State

**File:** `internal/store/adapter.go`

Full transaction parsing with signature recovery:

```go
func ParseRPCBlock(rpcBlock *types.Block) *index.Block {
    block := &index.Block{
        Height:       rpcBlock.NumberU64(),
        Hash:         rpcBlock.Hash().Bytes(),
        ParentHash:   rpcBlock.ParentHash().Bytes(),
        Timestamp:    rpcBlock.Time(),
        Miner:        rpcBlock.Coinbase().Bytes(),
        GasUsed:      rpcBlock.GasUsed(),
        TxCount:      len(rpcBlock.Transactions()),
        Transactions: make([]index.Transaction, 0, len(rpcBlock.Transactions())),
    }

    // Extract each transaction with proper parsing
    for txIndex, tx := range rpcBlock.Transactions() {
        indexerTx := parseTransaction(tx, txIndex)
        block.Transactions = append(block.Transactions, indexerTx)
    }

    return block
}
```

---

## Implementation Steps

### Step 1: Import Required Packages

**File:** `internal/store/adapter.go` (top of file)

Add missing import:
```go
import (
    // ... existing imports ...
    "math/big"
    "github.com/ethereum/go-ethereum/common"
)
```

### Step 2: Implement parseTransaction Helper

**File:** `internal/store/adapter.go` (add new function after ParseRPCBlock)

```go
// parseTransaction converts a go-ethereum transaction to index.Transaction domain model
// Uses basic mode: no receipt fetching, estimates gas_used, assumes success=true
func parseTransaction(tx *types.Transaction, txIndex int) index.Transaction {
    // Recover sender address from transaction signature
    // This is required because Ethereum transactions don't include from_addr directly
    from, err := types.LatestSignerForChainID(tx.ChainId()).Sender(tx)
    var fromAddr []byte
    if err != nil {
        // Signature recovery failed - use zero address as fallback
        // This can happen for invalid transactions
        fromAddr = common.Address{}.Bytes()
    } else {
        fromAddr = from.Bytes()
    }

    // Get recipient address (nil for contract creation)
    var toAddr []byte
    if tx.To() != nil {
        toAddr = tx.To().Bytes()
    }
    // If nil, toAddr remains nil (contract creation transaction)

    return index.Transaction{
        Hash:      tx.Hash().Bytes(),
        TxIndex:   txIndex,
        FromAddr:  fromAddr,
        ToAddr:    toAddr,
        ValueWei:  tx.Value().String(),  // Convert big.Int to string
        GasUsed:   tx.Gas(),              // Using gas limit as estimate
        GasPrice:  tx.GasPrice().Uint64(),
        Nonce:     tx.Nonce(),
        Success:   true,                  // Assume success (no receipt)
        Logs:      []index.Log{},         // Empty for basic mode
    }
}
```

### Step 3: Update ParseRPCBlock to Call Helper

**File:** `internal/store/adapter.go:138-156` (update existing function)

Replace the current implementation with:

```go
func ParseRPCBlock(rpcBlock *types.Block) *index.Block {
    if rpcBlock == nil {
        return nil
    }

    block := &index.Block{
        Height:       rpcBlock.NumberU64(),
        Hash:         rpcBlock.Hash().Bytes(),
        ParentHash:   rpcBlock.ParentHash().Bytes(),
        Timestamp:    rpcBlock.Time(),
        Miner:        rpcBlock.Coinbase().Bytes(),
        GasUsed:      rpcBlock.GasUsed(),
        TxCount:      len(rpcBlock.Transactions()),
        Transactions: make([]index.Transaction, 0, len(rpcBlock.Transactions())),
    }

    // Extract transactions from block
    for txIndex, tx := range rpcBlock.Transactions() {
        indexerTx := parseTransaction(tx, txIndex)
        block.Transactions = append(block.Transactions, indexerTx)
    }

    return block
}
```

---

## Edge Cases to Handle

### 1. Contract Creation Transactions
- `tx.To()` returns `nil`
- Store `NULL` in database `to_addr` column
- Frontend must handle null values in display

### 2. Signature Recovery Failure
- Can happen for malformed transactions
- Use zero address as fallback
- Log warning for debugging

### 3. Legacy Transactions (Pre-EIP-155)
- May not have ChainID
- `types.LatestSignerForChainID()` handles this
- Sepolia uses EIP-155, so unlikely

### 4. Empty Blocks
- Block with 0 transactions
- `rpcBlock.Transactions()` returns empty slice
- Loop doesn't execute, Transactions field remains empty array

### 5. Large Blocks
- Some blocks have 200+ transactions
- Pre-allocate slice capacity: `make([]Transaction, 0, len(...))`
- No performance issue (array growth handled)

---

## Testing

### Unit Tests to Add

**File:** `internal/store/adapter_test.go` (create new file)

```go
func TestParseRPCBlock_WithTransactions(t *testing.T) {
    // Create mock RPC block with 3 transactions
    // Verify all 3 transactions parsed correctly
    // Check from/to addresses, value, gas, nonce
}

func TestParseRPCBlock_ContractCreation(t *testing.T) {
    // Create transaction with tx.To() = nil
    // Verify ToAddr is nil in parsed result
}

func TestParseRPCBlock_EmptyBlock(t *testing.T) {
    // Block with 0 transactions
    // Verify Transactions field is empty array
}

func TestParseTransaction_SignatureRecovery(t *testing.T) {
    // Test with valid transaction
    // Verify from address recovered correctly
}
```

### Integration Test

```go
func TestParseRPCBlock_Integration(t *testing.T) {
    // Fetch real block from Sepolia testnet
    // Parse it
    // Verify transaction count matches
    // Verify first transaction hash matches
}
```

---

## Verification Checklist

- [ ] `parseTransaction()` function added
- [ ] `ParseRPCBlock()` updated to extract transactions
- [ ] Nil `to_addr` handled for contract creation
- [ ] Signature recovery implemented with error handling
- [ ] Value stored as string (preserves precision)
- [ ] Code compiles: `go build ./internal/store`
- [ ] Unit tests added and passing
- [ ] No performance regression (parsing adds <20ms per block)

---

## Performance Impact

**Before:** ParseRPCBlock processes ~100 blocks/second (mock)
**After:** ParseRPCBlock processes ~80-90 blocks/second (with signature recovery)

**Impact on Backfill:**
- 5000 blocks with avg 50 transactions each = 250,000 transactions
- Signature recovery: ~0.1ms per transaction = 25 seconds total
- Still well within <5 minute target

---

## Common Issues and Solutions

### Issue 1: "ChainId() nil pointer"
**Solution:** Check `tx.ChainId()` for nil before calling `LatestSignerForChainID()`

### Issue 2: "Value().String() panic"
**Solution:** Check `tx.Value()` for nil before calling `String()`

### Issue 3: "Invalid signature"
**Solution:** Use zero address fallback, log warning for debugging

---

## Next Task

**Task 3:** Update Database Insertion
- Modify `InsertBlock()` to insert transactions
- Add transaction SQL insert statements
- Calculate and store fee_wei
