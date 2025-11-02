# Transaction Extraction - Task Dependency Graph

**Date:** 2025-11-01
**Plan:** Transaction and Log Extraction Implementation
**Total Tasks:** 6
**Critical Path:** Tasks 1 → 2 → 3 → 5 → 6

---

## Visual Dependency Graph

```
┌─────────────────────────────────────────────────────────────┐
│                     TASK EXECUTION FLOW                      │
└─────────────────────────────────────────────────────────────┘

START
  │
  ▼
┌──────────────────────────────────────┐
│ Task 1: Update Domain Model          │
│ • Add Transactions field to Block    │
│ • Define Transaction struct          │
│ • Define Log struct (future)         │
│ Effort: 10 min                       │
│ File: internal/index/livetail.go     │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│ Task 2: Implement Transaction Parsing│
│ • Extract txs from RPC blocks        │
│ • Signature recovery for from_addr   │
│ • Handle contract creation (nil to)  │
│ Effort: 30 min                       │
│ File: internal/store/adapter.go      │
└────────────┬─────────────────────────┘
             │
             ▼
┌──────────────────────────────────────┐
│ Task 3: Update Database Insertion    │
│ • Insert transactions with blocks    │
│ • Calculate fee_wei                  │
│ • Handle foreign keys                │
│ Effort: 30 min                       │
│ File: internal/store/adapter.go      │
└────────────┬─────────────────────────┘
             │
             ├──────────────────────────────┐
             │                              │
             ▼                              ▼
┌──────────────────────────────┐  ┌─────────────────────────────┐
│ Task 4: Add API Endpoint     │  │ Task 5: Fix Frontend Mock   │
│ • GET /v1/blocks/{id}/txs   │  │ • Remove mock tx hashes     │
│ • Store.GetBlockTransactions │  │ • Use real API calls        │
│ Effort: 30 min (OPTIONAL)    │  │ • Update handleNewBlock     │
│ Files: handlers.go, queries  │  │ Effort: 20 min              │
└──────────────┬───────────────┘  │ File: web/app.js            │
               │                  └─────────────┬───────────────┘
               │                                │
               └────────────┬───────────────────┘
                            │
                            ▼
               ┌─────────────────────────────────┐
               │ Task 6: End-to-End Testing      │
               │ • Verify worker indexes txs     │
               │ • Verify API returns txs        │
               │ • Verify frontend displays txs  │
               │ • Performance validation        │
               │ Effort: 20 min                  │
               └─────────────────────────────────┘
                            │
                            ▼
                          DONE
```

---

## Dependency Matrix

| Task | Depends On | Blocks | Can Run In Parallel | Required For Completion |
|------|------------|--------|---------------------|-------------------------|
| Task 1 | None | 2, 3, 4, 5, 6 | No | ✅ CRITICAL PATH |
| Task 2 | Task 1 | 3, 4, 5, 6 | No | ✅ CRITICAL PATH |
| Task 3 | Task 1, 2 | 4, 5, 6 | No | ✅ CRITICAL PATH |
| Task 4 | Task 3 | 5 | ⚠️ Can skip (optional) | ⚠️ OPTIONAL |
| Task 5 | Task 3, (Task 4) | 6 | No | ✅ CRITICAL PATH |
| Task 6 | All above | None | No | ✅ VERIFICATION |

---

## Critical Path

**Minimum Required Tasks (Critical Path):**

```
Task 1 (10 min) → Task 2 (30 min) → Task 3 (30 min) → Task 5 (20 min) → Task 6 (20 min)
```

**Total Time:** 110 minutes (1 hour 50 minutes)

**Task 4 is optional** - Frontend can fetch transactions from multiple blocks instead of using dedicated endpoint.

---

## Execution Strategies

### Strategy A: Sequential Execution (Safest)
```
1. Complete Task 1 fully
2. Test compilation
3. Complete Task 2 fully
4. Test parsing with unit tests
5. Complete Task 3 fully
6. Test database insertion
7. Skip Task 4 or do it after
8. Complete Task 5
9. Test frontend manually
10. Complete Task 6 (full validation)
```

**Time:** 2-3 hours total
**Risk:** Low (verify each step)

### Strategy B: Batch Execution (Faster)
```
Batch 1 (Backend - 70 min):
  - Task 1: Domain model
  - Task 2: Parsing
  - Task 3: Database insertion
  - Verify with database queries

Batch 2 (Frontend - 20 min):
  - Task 5: Fix frontend
  - Quick manual test

Batch 3 (Optional API - 30 min):
  - Task 4: Add endpoint
  - Update frontend to use it

Batch 4 (Validation - 20 min):
  - Task 6: Full E2E testing
```

**Time:** 2-2.5 hours total
**Risk:** Medium (less verification between steps)

### Strategy C: Parallel Execution (Fastest, Riskier)
```
Developer A:
  - Task 1 + 2 + 3 (Backend)

Developer B (or after A):
  - Task 4 + 5 (API + Frontend)

Then together:
  - Task 6 (Testing)
```

**Time:** 1.5-2 hours total
**Risk:** Higher (integration issues possible)

---

## Recommended Execution Order

**For single developer (you):**

### Phase 1: Backend Implementation (70 minutes)
```
1. Task 1: Update domain model (10 min)
   - Quick compile check

2. Task 2: Implement parsing (30 min)
   - Add unit tests
   - Verify transactions extracted

3. Task 3: Database insertion (30 min)
   - Add integration test
   - Verify transactions in database
```

**Checkpoint:** Run worker, check database has transactions

### Phase 2: Frontend Integration (20 minutes)
```
4. Task 5: Fix frontend mock data (20 min)
   - Remove hardcoded hashes
   - Use real API calls
   - Quick browser test
```

**Checkpoint:** Frontend shows real data

### Phase 3: Optional Enhancement (30 minutes)
```
5. Task 4: Add block transactions endpoint (30 min)
   - Only if time permits
   - Improves frontend performance
   - Not critical for MVP
```

### Phase 4: Validation (20 minutes)
```
6. Task 6: End-to-end testing (20 min)
   - Full pipeline validation
   - Performance check
   - Data accuracy verification
```

---

## Task Status Tracking

| Task | Status | Start Time | End Time | Duration | Issues |
|------|--------|------------|----------|----------|--------|
| Task 1 | Not Started | - | - | - | - |
| Task 2 | Not Started | - | - | - | - |
| Task 3 | Not Started | - | - | - | - |
| Task 4 | Not Started | - | - | - | - |
| Task 5 | Not Started | - | - | - | - |
| Task 6 | Not Started | - | - | - | - |

---

## Risk Assessment

### High Risk Points

1. **Signature Recovery Failures (Task 2)**
   - Risk: Invalid transactions cause worker to crash
   - Mitigation: Use zero address fallback, log errors

2. **Database Transaction Deadlocks (Task 3)**
   - Risk: Concurrent inserts cause deadlocks
   - Mitigation: Worker is single-threaded, no concurrency issues

3. **Frontend Breaking Changes (Task 5)**
   - Risk: Removing mock data breaks UI before real data flows
   - Mitigation: Test with real API first, keep mock as fallback

### Medium Risk Points

4. **Performance Degradation (Task 3)**
   - Risk: Transaction insertion slows backfill to >10 minutes
   - Mitigation: Accept slower backfill, document as known limitation

5. **API Endpoint Conflicts (Task 4)**
   - Risk: New route conflicts with existing routes
   - Mitigation: Test route registration, verify no conflicts

---

## Rollback Strategy

**If critical issue found:**

```
1. Revert Task 5 (Frontend):
   git checkout web/app.js

2. Revert Task 3 (Database):
   git checkout internal/store/adapter.go
   TRUNCATE transactions CASCADE;

3. Revert Task 2 (Parsing):
   git checkout internal/store/adapter.go

4. Revert Task 1 (Domain):
   git checkout internal/index/livetail.go

5. Rebuild and restart:
   go build ./cmd/worker
   go build ./cmd/api
```

**Rollback Time:** <5 minutes

---

## Success Criteria

**Plan is complete when:**
- ✅ All 6 tasks documented
- ✅ Dependencies clearly defined
- ✅ Acceptance criteria established
- ✅ Testing strategy documented
- ✅ Rollback plan exists
- ✅ Time estimates provided

**Implementation is complete when:**
- ✅ All critical path tasks (1,2,3,5,6) done
- ✅ Frontend shows real transaction data
- ✅ API returns real transactions
- ✅ Database populated correctly
- ✅ No blocking issues
- ✅ Performance acceptable

---

_Execute tasks in order: 1 → 2 → 3 → 5 → 6 (skip 4 if time constrained)_
