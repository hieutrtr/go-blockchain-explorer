# Ad-Hoc Plans - Transaction Extraction Implementation

**Created:** 2025-11-01
**Purpose:** Critical gap fix for transaction and log extraction
**Status:** Ready for Implementation

---

## Overview

This directory contains the ad-hoc plan for implementing transaction extraction from Ethereum blocks. This was identified as a critical gap during application wiring - while blocks are being indexed, transactions and logs are not being extracted or stored.

---

## Plan Documents

### Main Plan
📄 **[transaction-extraction-plan.md](transaction-extraction-plan.md)**
- Complete problem analysis
- Solution approach (Option B - Basic Transaction Extraction)
- High-level implementation plan
- Success criteria and rollback strategy

---

## Task Breakdown

### Critical Path (Required)

1. 📋 **[task-1-update-domain-model.md](task-1-update-domain-model.md)**
   - **Effort:** 10 minutes
   - **Dependencies:** None
   - **Deliverable:** Block struct with Transactions field
   - **Status:** Ready

2. 📋 **[task-2-implement-transaction-parsing.md](task-2-implement-transaction-parsing.md)**
   - **Effort:** 30 minutes
   - **Dependencies:** Task 1
   - **Deliverable:** ParseRPCBlock() extracts transactions
   - **Status:** Ready

3. 📋 **[task-3-update-database-insertion.md](task-3-update-database-insertion.md)**
   - **Effort:** 30 minutes
   - **Dependencies:** Task 1, Task 2
   - **Deliverable:** InsertBlock() stores transactions
   - **Status:** Ready

5. 📋 **[task-5-fix-frontend-mock-data.md](task-5-fix-frontend-mock-data.md)**
   - **Effort:** 20 minutes
   - **Dependencies:** Task 3
   - **Deliverable:** Frontend uses real API data
   - **Status:** Ready

6. 📋 **[task-6-end-to-end-testing.md](task-6-end-to-end-testing.md)**
   - **Effort:** 20 minutes
   - **Dependencies:** Tasks 1-5
   - **Deliverable:** Complete pipeline verification
   - **Status:** Ready

### Optional Enhancement

4. 📋 **[task-4-add-block-transactions-endpoint.md](task-4-add-block-transactions-endpoint.md)**
   - **Effort:** 30 minutes
   - **Dependencies:** Task 3
   - **Deliverable:** GET /v1/blocks/{id}/transactions endpoint
   - **Status:** Optional (improves frontend UX)

---

## Execution Guide

### Quick Start (Minimum Viable)

**Execute critical path only:**
```
Task 1 (10 min) → Task 2 (30 min) → Task 3 (30 min) → Task 5 (20 min) → Task 6 (20 min)
```

**Total Time:** ~2 hours

**Result:** Transactions extracted and displayed, frontend works with real data

### Full Implementation (Recommended)

**Execute all tasks:**
```
Task 1 → Task 2 → Task 3 → Task 4 → Task 5 → Task 6
```

**Total Time:** ~2.5 hours

**Result:** Optimized API endpoint + complete functionality

---

## Dependency Visualization

See **[task-dependency-graph.md](task-dependency-graph.md)** for detailed dependency graph and execution strategies.

---

## Current Status

| Task | Status | Assignee | Notes |
|------|--------|----------|-------|
| Task 1 | ⏳ Ready | - | No blockers |
| Task 2 | ⏳ Ready | - | Depends on Task 1 |
| Task 3 | ⏳ Ready | - | Depends on Tasks 1, 2 |
| Task 4 | ⏳ Optional | - | Can be skipped for MVP |
| Task 5 | ⏳ Ready | - | Depends on Task 3 |
| Task 6 | ⏳ Ready | - | Final validation |

---

## Impact Assessment

### What Gets Fixed

**Backend:**
- ✅ Transactions extracted from blocks
- ✅ Transactions stored in database
- ✅ Transaction search works (`/v1/txs/{hash}`)
- ✅ Address history works (`/v1/address/{addr}/txs`)

**Frontend:**
- ✅ Transaction table shows real data (not mock)
- ✅ Transaction search functional
- ✅ Address history functional
- ✅ Real-time transaction updates

**Demo-ability:**
- ✅ Can show complete blockchain explorer
- ✅ Transaction search demonstrates functionality
- ✅ Address tracking shows real activity
- ✅ No "fake data" disclaimer needed

### What Remains Deferred

**Still Missing (Future Enhancements):**
- ❌ Accurate gas_used (requires receipt fetching)
- ❌ Failed transaction detection (requires receipts)
- ❌ Event logs extraction (requires receipts)
- ❌ Log filtering queries

**Impact:** 90% functionality achieved, 10% deferred for performance

---

## Files Modified Summary

### New Files Created
- `internal/store/adapter_test.go` - Transaction parsing tests

### Files Modified
- `internal/index/livetail.go` - Add Transaction, Log structs
- `internal/store/adapter.go` - Add parseTransaction(), update InsertBlock()
- `internal/store/queries.go` - Add GetBlockTransactions() (optional)
- `internal/api/handlers.go` - Add handleGetBlockTransactions() (optional)
- `internal/api/server.go` - Register new route (optional)
- `web/app.js` - Remove mock data, use real API

**Total Files:** 6-7 files

---

## Testing Checklist

After implementation, verify:

- [ ] Worker compiles: `go build ./cmd/worker`
- [ ] API compiles: `go build ./cmd/api`
- [ ] Unit tests pass: `go test ./internal/...`
- [ ] Worker starts without errors
- [ ] Database has transactions: `SELECT COUNT(*) FROM transactions;`
- [ ] API returns transactions: `curl http://localhost:8080/v1/txs/{hash}`
- [ ] Frontend shows real data (open browser)
- [ ] No mock data comments remain: `grep -r "mock" web/`
- [ ] Performance acceptable: Backfill <10 minutes

---

## Success Metrics

**Before Implementation:**
```
Blocks indexed:        5000
Transactions indexed:  0        ❌
Frontend transactions: Mock data ❌
Transaction search:    404 errors ❌
Address history:       Empty     ❌
```

**After Implementation:**
```
Blocks indexed:        5000      ✅
Transactions indexed:  250,000+  ✅
Frontend transactions: Real data ✅
Transaction search:    Working   ✅
Address history:       Populated ✅
```

---

## Next Steps

1. **Review this plan** - Ensure approach makes sense
2. **Start Task 1** - Update domain model (10 min)
3. **Execute sequentially** - Follow critical path
4. **Test after each task** - Verify before moving forward
5. **Complete Task 6** - Full validation

---

## References

- **Main Plan:** transaction-extraction-plan.md
- **Tech Spec:** docs/tech-spec-epic-1.md (Data Architecture)
- **Database Schema:** migrations/000001_initial_schema.up.sql
- **Current Implementation:** internal/store/adapter.go:102 (TODO comment)

---

_This ad-hoc plan addresses the transaction extraction gap discovered during worker wiring. Implementation follows Option B (Basic Transaction Data) for fast delivery while maintaining performance targets._
