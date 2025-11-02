# Task 6: End-to-End Transaction Flow Testing

**Status:** Ready for Execution
**Priority:** HIGH
**Estimated Time:** 20 minutes
**Dependencies:** Tasks 1-5 complete
**Blocks:** None (final verification task)

---

## Objective

Verify the complete transaction extraction and display pipeline works end-to-end, from Ethereum RPC through to frontend display, with real blockchain data.

---

## Test Scenario

### Full Pipeline Test Flow

```
Ethereum Sepolia RPC
    ↓ (fetch block with transactions)
RPC Client (internal/rpc)
    ↓ (return types.Block)
ParseRPCBlock (internal/store/adapter.go)
    ↓ (extract transactions)
InsertBlock (internal/store/adapter.go)
    ↓ (store in database)
PostgreSQL Database
    ↓ (query via API)
REST API Handler (internal/api/handlers.go)
    ↓ (return JSON)
Frontend (web/app.js)
    ↓ (render in browser)
User sees real blockchain transactions
```

---

## Test Cases

### Test Case 1: Worker Indexes Transactions

**Objective:** Verify worker extracts and stores transactions

**Steps:**
1. Clear database: `TRUNCATE blocks, transactions CASCADE;`
2. Configure small backfill: `BACKFILL_END_HEIGHT=10`
3. Start worker: `go run cmd/worker/main.go`
4. Wait for backfill to complete (~10 seconds)
5. Query database:
   ```sql
   SELECT COUNT(*) FROM blocks;      -- Should be 11 (blocks 0-10)
   SELECT COUNT(*) FROM transactions; -- Should be >0 (extracted txs)
   SELECT COUNT(*) FROM logs;         -- Should be 0 (not extracted yet)
   ```

**Expected Results:**
- ✅ Blocks table has 11 rows
- ✅ Transactions table has 50-500 rows (depends on block activity)
- ✅ Worker logs show "batch inserted successfully"
- ✅ No errors in worker logs

**Failure Cases:**
- ❌ Transactions table empty → Task 2 or 3 broken
- ❌ Worker crashes → Check error logs
- ❌ Foreign key violations → Check block_height references

---

### Test Case 2: API Returns Transaction Data

**Objective:** Verify REST API endpoints return real transactions

**Steps:**
1. Ensure worker has indexed blocks (Test Case 1)
2. Start API: `go run cmd/api/main.go`
3. Test endpoints:

```bash
# Test 1: Get specific transaction
curl http://localhost:8080/v1/txs/0x{actual_hash_from_db}

# Expected: 200 OK with transaction details
{
  "hash": "0x...",
  "block_height": 5,
  "from_addr": "0x...",
  "to_addr": "0x...",
  "value_wei": "1000000000000000000",
  "success": true
}

# Test 2: Get block transactions
curl http://localhost:8080/v1/blocks/5/transactions

# Expected: 200 OK with array of transactions
{
  "transactions": [...],
  "total": 47,
  "limit": 100,
  "offset": 0
}

# Test 3: Get address history
curl http://localhost:8080/v1/address/0x{address_from_block}/txs?limit=10

# Expected: 200 OK with transactions for that address
{
  "transactions": [...],
  "total": 23,
  "limit": 10,
  "offset": 0
}

# Test 4: Check health
curl http://localhost:8080/health

# Expected: 200 OK with indexer status
{
  "status": "healthy",
  "database": "connected",
  "indexer_last_block": 10,
  "indexer_lag_seconds": 5
}
```

**Expected Results:**
- ✅ `/v1/txs/{hash}` returns real transaction
- ✅ `/v1/blocks/{height}/transactions` returns transaction array
- ✅ `/v1/address/{addr}/txs` returns transaction history
- ✅ Transaction hashes match database
- ✅ All fields populated correctly

**Failure Cases:**
- ❌ 404 Not Found → Transactions not in database
- ❌ Empty arrays → Querying wrong table or orphaned blocks
- ❌ 500 Error → Database query issues

---

### Test Case 3: Frontend Displays Real Data

**Objective:** Verify frontend shows actual blockchain transactions

**Steps:**
1. Ensure worker and API running (Test Cases 1-2)
2. Open browser: `http://localhost:8080`
3. Open browser DevTools console (F12)
4. Observe page load

**Expected Results:**
- ✅ Live blocks ticker shows 10 recent blocks
- ✅ Transaction table shows 25 real transactions (not mock data)
- ✅ Transaction hashes start with `0x` and are 66 chars long
- ✅ From/to addresses start with `0x` and are 42 chars long
- ✅ Values display in ETH format (e.g., "0.1234 ETH")
- ✅ WebSocket status indicator shows "Connected"
- ✅ Console shows no errors

**Visual Verification:**
```
Before (Mock Data):
- Transaction hashes: 0x123...abc (fake, always same)
- Addresses: 0xabc...def (hardcoded)
- Values: Random mock values

After (Real Data):
- Transaction hashes: 0x7a8b... (real, from Sepolia)
- Addresses: 0x742d... (real Ethereum addresses)
- Values: Actual transferred amounts
```

**Failure Cases:**
- ❌ Transaction table empty → fetchRecentTransactions() failing
- ❌ Shows mock data → Frontend changes not applied
- ❌ Console errors → API endpoint issues

---

### Test Case 4: Real-Time Transaction Updates

**Objective:** Verify new transactions appear as blocks are indexed

**Steps:**
1. Frontend open at `http://localhost:8080`
2. Worker running and indexing new blocks
3. Wait for new block to be indexed
4. Observe transaction table

**Expected Results:**
- ✅ WebSocket receives `newBlock` message
- ✅ Block ticker updates with new block
- ✅ Transaction table refreshes automatically
- ✅ New transactions appear in the list
- ✅ No page refresh required

**Verification in Console:**
```javascript
// Should see WebSocket messages
WebSocket message received: {type: "newBlock", data: {...}}
Fetched 25 recent transactions
Transaction table updated
```

---

### Test Case 5: Address History

**Objective:** Verify address transaction history works

**Steps:**
1. Pick an address from a transaction in the UI
2. Navigate to address history (click address link)
3. Verify transactions load

**Expected Results:**
- ✅ Address history page shows transactions
- ✅ Both sent and received transactions appear
- ✅ Pagination works (if >50 transactions)
- ✅ Clicking transaction navigates to detail page

---

### Test Case 6: Transaction Detail Page

**Objective:** Verify transaction detail displays all fields

**Steps:**
1. Click a transaction hash from any list
2. Verify detail page loads

**Expected Results:**
- ✅ Transaction hash displayed
- ✅ Block height shown (clickable)
- ✅ From address shown (clickable)
- ✅ To address shown (clickable) or "Contract Creation"
- ✅ Value displayed in ETH
- ✅ Gas used, gas price, nonce shown
- ✅ Success badge shown (green checkmark)

---

## Performance Validation

### Backfill Performance

**Test:**
```bash
time go run cmd/worker/main.go
# With BACKFILL_END_HEIGHT=5000
```

**Expected:**
- ✅ Completes in <8 minutes (acceptable with transaction extraction)
- ✅ No memory leaks (monitor with `top` or Activity Monitor)
- ✅ Database size reasonable (~500MB for 5000 blocks + transactions)

**Metrics to Check:**
```bash
# Database size
psql -d blockchain_explorer -c "
    SELECT
        pg_size_pretty(pg_total_relation_size('blocks')) as blocks_size,
        pg_size_pretty(pg_total_relation_size('transactions')) as tx_size;
"

# Row counts
psql -d blockchain_explorer -c "
    SELECT
        (SELECT COUNT(*) FROM blocks) as blocks,
        (SELECT COUNT(*) FROM transactions) as transactions;
"
```

### API Performance

**Test:**
```bash
# Test transaction endpoint latency
time curl http://localhost:8080/v1/txs/0x{hash}

# Test block transactions endpoint
time curl http://localhost:8080/v1/blocks/1234/transactions?limit=100
```

**Expected:**
- ✅ Transaction detail: <100ms (p95)
- ✅ Block transactions: <150ms (p95)
- ✅ Address history: <200ms (p95 with index)

---

## Data Validation

### Database Integrity Checks

```sql
-- Check foreign key integrity
SELECT COUNT(*) FROM transactions t
LEFT JOIN blocks b ON t.block_height = b.height
WHERE b.height IS NULL;
-- Expected: 0 (all transactions have valid block references)

-- Check transaction counts match
SELECT b.height, b.tx_count, COUNT(t.hash) as actual_txs
FROM blocks b
LEFT JOIN transactions t ON t.block_height = b.height
WHERE b.orphaned = FALSE
GROUP BY b.height, b.tx_count
HAVING b.tx_count != COUNT(t.hash);
-- Expected: 0 rows (tx_count matches actual transactions)

-- Check no duplicate transactions
SELECT hash, COUNT(*) FROM transactions
GROUP BY hash HAVING COUNT(*) > 1;
-- Expected: 0 rows (all hashes unique)

-- Sample transaction data
SELECT
    encode(hash, 'hex') as hash,
    block_height,
    encode(from_addr, 'hex') as from_addr,
    encode(to_addr, 'hex') as to_addr,
    value_wei,
    success
FROM transactions
LIMIT 5;
-- Expected: Real hex hashes and addresses
```

---

## Acceptance Criteria

### Backend Validation
- [ ] Worker logs show "X transactions inserted" messages
- [ ] Database transactions table populated
- [ ] Foreign keys valid (no orphaned transactions)
- [ ] Transaction counts match (`blocks.tx_count` = actual count)
- [ ] No duplicate transaction hashes

### API Validation
- [ ] `/v1/txs/{hash}` returns real transactions
- [ ] `/v1/blocks/{height}/transactions` works
- [ ] `/v1/address/{addr}/txs` returns history
- [ ] Response times within targets (<200ms)
- [ ] Pagination works correctly

### Frontend Validation
- [ ] Transaction table shows real data (not mock)
- [ ] Transaction hashes are real (verify on Sepolia explorer)
- [ ] Addresses are real Ethereum addresses
- [ ] Values match blockchain (can verify on Etherscan)
- [ ] Real-time updates work (new blocks → new transactions)
- [ ] Navigation works (click tx → detail page)
- [ ] No console errors

### Performance Validation
- [ ] Backfill completes in reasonable time (<10 min)
- [ ] No memory leaks in worker
- [ ] API responses fast (<200ms p95)
- [ ] Frontend responsive (no UI lag)

---

## Success Metrics

After all tests pass:

```
✅ Blocks indexed: 5000+
✅ Transactions extracted: 100,000+ (depends on block activity)
✅ Logs extracted: 0 (deferred to future)
✅ API endpoints functional: 8/8
✅ Frontend features working: All transaction features
✅ Performance targets: Met (with minor adjustment)
✅ Data accuracy: 100% (verified against Sepolia)
```

---

## Common Issues and Solutions

### Issue 1: Transactions Table Empty
**Cause:** ParseRPCBlock() not extracting transactions
**Solution:** Check Task 2 implementation, verify loop executes
**Debug:** Add log in parseTransaction() to verify it's called

### Issue 2: Foreign Key Violations
**Cause:** Block not inserted before transactions
**Solution:** Verify InsertBlock() inserts block first, then transactions
**Debug:** Check database transaction isolation level

### Issue 3: Frontend Shows Empty Table
**Cause:** API not returning transactions or frontend not fetching
**Solution:** Test API with curl first, then check browser network tab
**Debug:** Browser DevTools → Network tab → Check API responses

### Issue 4: Slow Backfill
**Cause:** Transaction insertion adds overhead
**Solution:** Expected behavior, adjust target or optimize batch inserts
**Mitigation:** Accept 8-10 minute backfill time

---

## Completion Criteria

**Definition of Done:**
- All 6 test cases pass
- All acceptance criteria met
- No blocking issues found
- Performance acceptable (backfill <10 min)
- Data accuracy verified
- Frontend displays real data
- No mock data remaining

---

## Next Steps After Testing

1. **Update sprint status:** Mark transaction extraction as complete
2. **Document findings:** Note any performance impacts or limitations
3. **Create summary:** Document what works, what's deferred (logs, receipts)
4. **Plan enhancements:** Create backlog items for receipt fetching, log extraction

---

_This task validates the entire transaction extraction feature and confirms the blockchain explorer is fully functional._
