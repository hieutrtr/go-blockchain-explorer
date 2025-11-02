# Task 5: Fix Frontend Mock Data

**Status:** Ready for Implementation
**Priority:** MEDIUM
**Estimated Time:** 20 minutes
**Dependencies:** Task 4 (API endpoint available)
**Blocks:** Task 6

---

## Objective

Remove mock/hardcoded transaction data from the frontend and replace with real API calls to display actual blockchain transactions indexed from Ethereum Sepolia.

---

## Current State (Mock Data)

**File:** `web/app.js`

**Mock Data Locations:**
```javascript
// Line ~136: Mock transaction hashes
let transactionsLimit = 18; // Show all available mock transactions

// Line ~145-155: Mock workaround
// Fetch known mock transactions directly (workaround until backend provides tx listing)
const mockTxHashes = [
    '0x123...', // Hardcoded
    '0x456...',
    // ... more mock hashes
];

for (const hash of mockTxHashes) {
    const tx = await fetchTransaction(hash);
    if (tx) {
        newTransactions.push(tx);
    }
}

// Line ~220-230: Mock comment
// For mock data testing, new transactions would come from real blockchain indexing
// For mock data, transactions don't change - just update UI
```

**Problem:** Frontend displays fake data, not real blockchain transactions.

---

## Target State (Real Data)

**File:** `web/app.js`

Replace mock data fetching with real API calls:

```javascript
// Fetch recent transactions from indexed blocks
async function fetchRecentTransactions() {
    try {
        // Strategy: Fetch latest blocks and get their transactions
        const blocksResponse = await fetch(`${API_BASE}/v1/blocks?limit=5&offset=0`);
        if (!blocksResponse.ok) {
            throw new Error(`HTTP ${blocksResponse.status}`);
        }

        const blocksData = await blocksResponse.json();
        const allTransactions = [];

        // Fetch transactions from each recent block
        for (const block of blocksData.blocks) {
            if (block.tx_count === 0) continue;

            // Fetch transactions for this block
            const txResponse = await fetch(
                `${API_BASE}/v1/blocks/${block.height}/transactions?limit=25`
            );

            if (!txResponse.ok) {
                console.warn(`Failed to fetch transactions for block ${block.height}`);
                continue;
            }

            const txData = await txResponse.json();
            allTransactions.push(...txData.transactions);

            // Stop once we have enough transactions
            if (allTransactions.length >= 25) break;
        }

        return allTransactions.slice(0, 25);

    } catch (error) {
        console.error('Failed to fetch recent transactions:', error);

        // Retry once after delay
        await new Promise(resolve => setTimeout(resolve, API_RETRY_DELAY_MS));

        try {
            // Simplified retry: just fetch from latest block
            const txResponse = await fetch(`${API_BASE}/v1/blocks/latest/transactions?limit=25`);
            const txData = await txResponse.json();
            return txData.transactions || [];
        } catch (retryError) {
            console.error('Retry failed:', retryError);
            return [];
        }
    }
}
```

---

## Implementation Steps

### Step 1: Remove Mock Transaction Hashes

**File:** `web/app.js` (around lines 140-160)

**Remove:**
```javascript
let transactionsLimit = 18; // Show all available mock transactions
const mockTxHashes = [
    '0x7a1234abcd...',
    '0x8b5678efgh...',
    // ... all hardcoded hashes
];
```

### Step 2: Update fetchRecentTransactions Function

**File:** `web/app.js` (replace existing implementation)

**Find and replace:**
```javascript
// OLD: Mock workaround
async function fetchRecentTransactions() {
    // Fetch known mock transactions directly (workaround...)
    const newTransactions = [];
    for (const hash of mockTxHashes) {
        // ...
    }
}

// NEW: Real API calls
async function fetchRecentTransactions() {
    try {
        const blocksResponse = await fetch(`${API_BASE}/v1/blocks?limit=5&offset=0`);
        if (!blocksResponse.ok) throw new Error(`HTTP ${blocksResponse.status}`);

        const blocksData = await blocksResponse.json();
        const allTransactions = [];

        for (const block of blocksData.blocks) {
            if (block.tx_count === 0) continue;

            const txResponse = await fetch(
                `${API_BASE}/v1/blocks/${block.height}/transactions?limit=25`
            );
            if (!txResponse.ok) continue;

            const txData = await txResponse.json();
            allTransactions.push(...txData.transactions);

            if (allTransactions.length >= 25) break;
        }

        return allTransactions.slice(0, 25);
    } catch (error) {
        console.error('Failed to fetch recent transactions:', error);
        return [];
    }
}
```

### Step 3: Update handleNewBlock Function

**File:** `web/app.js` (around lines 220-230)

**Remove mock data comments:**
```javascript
function handleNewBlock(blockData) {
    // ... existing code ...

    // REMOVE: For mock data, transactions don't change - just update UI
    // REMOVE: For mock data testing, new transactions would come from real blockchain indexing

    // ADD: Refresh transactions when new block arrives
    if (blockData.tx_count > 0) {
        fetchRecentTransactions().then(txs => {
            transactions = txs;
            renderTransactions();
        });
    }
}
```

### Step 4: Clean Up Comments

**File:** `web/app.js`

Search for and remove/update:
- "mock data"
- "workaround"
- "TODO" related to transactions
- Hardcoded transaction limits

---

## Alternative: Use Recent Transactions Endpoint

**If we implement `/v1/transactions/recent`:**

**Simpler Implementation:**
```javascript
async function fetchRecentTransactions() {
    try {
        const response = await fetch(`${API_BASE}/v1/transactions/recent?limit=25`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const data = await response.json();
        return data.transactions || [];

    } catch (error) {
        console.error('Failed to fetch recent transactions:', error);

        // Retry once
        await new Promise(resolve => setTimeout(resolve, API_RETRY_DELAY_MS));
        try {
            const response = await fetch(`${API_BASE}/v1/transactions/recent?limit=25`);
            const data = await response.json();
            return data.transactions || [];
        } catch (retryError) {
            return [];
        }
    }
}
```

**Advantages:**
- Much simpler frontend code
- Single API call instead of multiple
- Better performance

**Trade-off:**
- Requires implementing additional backend endpoint

---

## Data Flow After Changes

```
1. Worker indexes block
   └─> ParseRPCBlock() extracts transactions
       └─> InsertBlock() stores transactions in database

2. Frontend loads page
   └─> fetchRecentTransactions() calls API
       └─> API queries transactions table
           └─> Returns real blockchain data

3. WebSocket sends new block
   └─> Frontend refreshes transactions
       └─> Shows newly indexed transactions
```

---

## Testing

### Browser Console Testing

```javascript
// Open browser console on frontend

// Test transaction fetching
fetchRecentTransactions().then(txs => {
    console.log('Fetched transactions:', txs.length);
    console.log('First transaction:', txs[0]);
});

// Expected output:
// Fetched transactions: 25
// First transaction: {hash: "0x...", from_addr: "0x...", value_wei: "..."}
```

### Visual Verification

1. Start worker: `go run cmd/worker/main.go`
2. Wait for blocks to be indexed
3. Open frontend: `http://localhost:8080`
4. **Verify:** Transaction table shows real data (not mock hashes)
5. **Verify:** Clicking transaction hash navigates to detail page
6. **Verify:** New blocks update transaction list

### API Testing

```bash
# Fetch transactions for block 1234
curl http://localhost:8080/v1/blocks/1234/transactions

# Expected:
{
  "transactions": [
    {
      "hash": "0xabc...",
      "from_addr": "0xdef...",
      "to_addr": "0x123...",
      "value_wei": "1000000000000000000",
      // ... real blockchain data
    }
  ],
  "total": 47
}
```

---

## Edge Cases

### 1. Block with No Transactions
**Response:**
```json
{
  "transactions": [],
  "total": 0,
  "limit": 25,
  "offset": 0
}
```

**Frontend Handling:**
```javascript
if (txs.length === 0) {
    document.getElementById('transactions-list').innerHTML =
        '<tr><td colspan="5">No transactions yet</td></tr>';
}
```

### 2. API Endpoint Not Available (Server Down)
**Frontend:**
```javascript
catch (error) {
    console.error('API error:', error);
    // Keep displaying last known transactions
    // Show error indicator in UI
}
```

### 3. Block Still Being Indexed
**Scenario:** Frontend asks for block N, but worker hasn't indexed it yet

**Response:** 404 Not Found or empty array

**Frontend:** Retry after delay or show "Block not indexed yet"

---

## Acceptance Criteria

- [ ] Mock transaction hash arrays removed
- [ ] `fetchRecentTransactions()` uses real API
- [ ] No hardcoded transaction data remaining
- [ ] Frontend displays real blockchain transactions
- [ ] Transaction table updates when new blocks arrive
- [ ] Empty state handled gracefully
- [ ] API errors handled with retry logic
- [ ] Browser console shows no errors
- [ ] Visual verification: Real transaction hashes displayed

---

## Verification Checklist

```bash
# 1. Check no mock data remains
grep -n "mock.*transaction\|TODO.*transaction" web/app.js
# Expected: No results

# 2. Test frontend loads
open http://localhost:8080
# Verify: Transaction table populated with real data

# 3. Check browser console
# Expected: No errors, transactions fetched successfully

# 4. Verify transaction details
# Click transaction hash, should navigate to detail page with real data
```

---

## Rollback Plan

If issues arise:

1. **Keep mock data as fallback:**
   ```javascript
   async function fetchRecentTransactions() {
       try {
           // Try real API first
           return await fetchFromAPI();
       } catch (error) {
           // Fallback to mock data
           return getMockTransactions();
       }
   }
   ```

2. **Revert to previous version:**
   ```bash
   git checkout web/app.js
   ```

---

## Next Task

**Task 6:** End-to-End Testing
- Verify complete transaction flow
- Test backfill → database → API → frontend
- Validate data correctness
