# Task 1: Update Block Domain Model with Transactions

**Status:** Ready for Implementation
**Priority:** HIGH
**Estimated Time:** 10 minutes
**Dependencies:** None
**Blocks:** Tasks 2, 3, 4, 5, 6

---

## Objective

Extend the `Block` domain model to include `Transactions` field, enabling the indexer pipeline to carry transaction data from RPC through to database storage.

---

## Current State

**File:** `internal/index/livetail.go:73-81`

```go
type Block struct {
    Height     uint64
    Hash       []byte
    ParentHash []byte
    Timestamp  uint64
    Miner      []byte
    GasUsed    uint64
    TxCount    int
    // Missing: Transactions field
}
```

**Problem:** Block only stores transaction count, not actual transaction data.

---

## Target State

**File:** `internal/index/livetail.go`

```go
type Block struct {
    Height       uint64
    Hash         []byte
    ParentHash   []byte
    Timestamp    uint64
    Miner        []byte
    GasUsed      uint64
    TxCount      int
    Transactions []Transaction  // NEW
}

type Transaction struct {
    Hash      []byte
    TxIndex   int
    FromAddr  []byte
    ToAddr    []byte  // nil for contract creation
    ValueWei  string  // String to preserve big.Int precision
    GasUsed   uint64  // Estimated in basic mode (tx.Gas())
    GasPrice  uint64
    Nonce     uint64
    Success   bool    // Assumed true in basic mode
    Logs      []Log   // Empty in basic mode, future enhancement
}

type Log struct {
    LogIndex  int
    Address   []byte
    Topics    [4][]byte  // Up to 4 indexed topics
    Data      []byte     // Non-indexed data
}
```

---

## Implementation Steps

### Step 1: Add Transaction struct
**File:** `internal/index/livetail.go` (after Block struct, around line 82)

```go
// Transaction represents a blockchain transaction (domain model for indexer)
type Transaction struct {
    Hash      []byte   // Transaction hash
    TxIndex   int      // Index within block
    FromAddr  []byte   // Sender address (20 bytes)
    ToAddr    []byte   // Recipient address (20 bytes, nil for contract creation)
    ValueWei  string   // Value transferred in Wei (as string to preserve precision)
    GasUsed   uint64   // Gas consumed (estimated from gas limit in basic mode)
    GasPrice  uint64   // Gas price in Wei
    Nonce     uint64   // Sender nonce
    Success   bool     // Transaction success status (assumed true in basic mode)
    Logs      []Log    // Event logs emitted (empty in basic mode)
}
```

### Step 2: Add Log struct
**File:** `internal/index/livetail.go` (after Transaction struct)

```go
// Log represents an event log emitted by a transaction (domain model for indexer)
type Log struct {
    LogIndex  int        // Index within transaction
    Address   []byte     // Contract address that emitted the log (20 bytes)
    Topics    [4][]byte  // Indexed topics (up to 4, each 32 bytes, nil if unused)
    Data      []byte     // Non-indexed data payload
}
```

### Step 3: Add Transactions field to Block
**File:** `internal/index/livetail.go` (update Block struct)

Add this field to the existing Block struct:
```go
type Block struct {
    Height       uint64
    Hash         []byte
    ParentHash   []byte
    Timestamp    uint64
    Miner        []byte
    GasUsed      uint64
    TxCount      int
    Transactions []Transaction  // NEW: Actual transaction data
}
```

---

## Verification

### Compilation Check
```bash
go build ./internal/index
```

Should compile without errors.

### Type Compatibility Check
- Block struct used in:
  - `backfill.go:348` (parseRPCBlockToDomain)
  - `livetail.go:200` (defaultParseRPCBlock)
  - `reorg.go:67` (HandleReorg parameter)
- All should compile after adding Transactions field

### Test Impact
- Unit tests may need updates if they assert on Block struct fields
- Run: `go test ./internal/index -run TestLiveTail`

---

## Acceptance Criteria

- [ ] Transaction struct defined with all required fields
- [ ] Log struct defined for future use
- [ ] Block struct includes Transactions field
- [ ] Code compiles without errors
- [ ] Existing tests still pass (or updated)
- [ ] No breaking changes to other components

---

## Rollback

If issues arise:
```bash
git checkout internal/index/livetail.go
```

Domain model changes are local to this file, easily reversible.

---

## Next Task

**Task 2:** Implement Transaction Parsing
- Update `ParseRPCBlock()` to extract transactions from blocks
- Implement `parseTransaction()` helper function
