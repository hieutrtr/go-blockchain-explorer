# UX Specification - Blockchain Explorer Frontend

**Project:** Blockchain Explorer
**Author:** Hieu
**Date:** 2025-10-29
**Project Level:** Level 2 (Small complete system with UI)
**UI Scope:** Minimal Single-Page Application for portfolio demonstration

---

## 1. Overview

### 1.1 Purpose

This UX specification documents the user interface design for the Blockchain Explorer's minimal frontend. The interface serves to:

1. **Demonstrate Technical Capabilities** - Showcase backend functionality (indexing, real-time updates, API design) through a functional interface
2. **Enable Quick Evaluation** - Allow technical evaluators to interact with the system within 5-10 minutes
3. **Prove Real-Time Competency** - Display live blockchain data updates via WebSocket
4. **Provide Search Functionality** - Enable lookup of blocks, transactions, and addresses

### 1.2 Design Philosophy

**"Functional Minimalism"** - The frontend is intentionally minimal to:

- Keep implementation within 7-day timeline (backend focus)
- Avoid distraction from backend capabilities being demonstrated
- Demonstrate production-ready patterns without over-engineering
- Enable evaluators to understand functionality immediately

**Key Principles:**
- **Clarity over aesthetics** - Information is presented clearly but without elaborate styling
- **Immediate feedback** - Live updates show system activity in real-time
- **Zero friction** - No authentication, no configuration, works immediately
- **Technical transparency** - Interface exposes system behavior for technical audience

### 1.3 Target Users

**Primary Persona:** Technical Evaluator (Senior Engineer, Hiring Manager)
- **Goal:** Assess candidate's technical competency in 15 minutes
- **Needs:** See system working, understand architecture, evaluate code quality
- **Behavior:** Will clone repo, run Docker Compose, observe system, review code

**Secondary Persona:** Developer Exploring Project
- **Goal:** Understand implementation patterns for learning or reference
- **Needs:** Working demo, clear documentation, accessible code
- **Behavior:** Will read README, run locally, explore codebase

---

## 2. User Personas

### Persona 1: Sarah - Senior Backend Engineer (Hiring Manager)

**Demographics:**
- 35 years old
- 10+ years experience in backend systems
- Currently hiring for senior backend role

**Context:**
- Reviewing portfolio projects from 5 candidates
- Has 30 minutes per candidate
- Values production-ready patterns over feature completeness

**Goals:**
- Quickly assess technical depth (concurrency, databases, APIs)
- Verify candidate can build real systems, not just tutorials
- Find evidence of production thinking (metrics, logging, error handling)

**Pain Points:**
- Too many portfolio projects are "toy apps"
- Hard to assess real competency from GitHub repos alone
- Limited time for deep code review

**How This UI Helps:**
- Live system proves it actually works (not vaporware)
- Real-time updates demonstrate WebSocket competency
- Metrics endpoint shows production observability mindset
- Clean interface suggests attention to user experience despite backend focus

---

### Persona 2: Alex - Mid-Level Developer (Learning)

**Demographics:**
- 27 years old
- 3 years experience, looking to level up
- Exploring blockchain indexing patterns

**Context:**
- Searching GitHub for reference implementations
- Learning Go and PostgreSQL
- Building similar system for learning

**Goals:**
- Understand architecture patterns for data-intensive apps
- See real-world Go code organization
- Learn indexing and real-time update patterns

**Pain Points:**
- Many examples are incomplete or outdated
- Hard to find working demos to interact with
- Unclear how pieces fit together

**How This UI Helps:**
- Working demo shows complete system in action
- Simple frontend makes system behavior observable
- Can experiment with searches to understand API design
- Documentation links code to running system

---

## 3. User Flows

### Flow 1: Technical Evaluator First Experience (Primary Flow)

**Scenario:** Sarah clones the repo and wants to assess the project in 15 minutes

**Entry Point:** README.md with setup instructions

**Steps:**

1. **Setup** (2 minutes)
   - Clone repository
   - Run `docker compose up`
   - Wait for services to start

2. **Observe Indexing** (3 minutes)
   - Watch terminal logs showing parallel block processing
   - See blocks being indexed (500/5000, 1000/5000, etc.)
   - Observe performance metrics in logs

3. **View Frontend** (5 minutes)
   - Open browser to `http://localhost:8080`
   - **See:** Live blocks ticker showing recently indexed blocks appearing
   - **See:** Recent transactions table populating
   - **Notice:** Real-time updates (blocks appearing every 12 seconds)
   - **Try:** Search for a transaction hash from the table
   - **See:** Transaction details displayed

4. **Explore APIs** (3 minutes)
   - Open `/metrics` endpoint in browser
   - See Prometheus metrics (blocks indexed, lag, API latency)
   - Try API endpoints using curl or browser (optional)

5. **Review Code** (2 minutes)
   - Quick scan of directory structure
   - Review component organization
   - Check test coverage

**Success Criteria:**
- System runs successfully within 2 minutes of `docker compose up`
- Live updates are immediately visible
- Search functionality works correctly
- Evaluator forms positive impression of technical competency

**Exit Points:**
- **Positive:** "This developer knows how to build real systems" → shortlist for interview
- **Neutral:** "Decent implementation" → continue reviewing other projects
- **Negative:** System fails to start or UI is broken → move to next candidate

---

### Flow 2: Search for Transaction (Core Feature Flow)

**Scenario:** User wants to look up a specific transaction to understand transaction details view

**Entry Point:** Main page with search bar

**Steps:**

1. **Locate Search Bar**
   - Search bar prominently displayed in header
   - Placeholder text: "Search by block height, hash, transaction hash, or address"

2. **Enter Transaction Hash**
   - User copies transaction hash from recent transactions table: `0x1234...abcd`
   - Pastes into search bar
   - Presses Enter or clicks search icon

3. **Validation**
   - Frontend validates input format (hex, correct length)
   - Shows loading indicator (brief spinner or "Searching...")

4. **API Request**
   - Frontend calls `GET /v1/txs/{hash}`
   - Backend queries database
   - Returns transaction details

5. **Display Results**
   - UI transitions to transaction detail view
   - Shows:
     - Transaction hash (full)
     - Block height (clickable link to block)
     - From address (clickable link to address history)
     - To address (clickable link to address history)
     - Value in ETH (converted from wei)
     - Fee in ETH
     - Gas used / Gas price
     - Status (✓ Success or ✗ Failed)
     - Timestamp

6. **Navigate Further** (Optional)
   - User clicks block height → navigates to block detail
   - User clicks address → navigates to address transaction history
   - User clicks back or uses search again

**Success Criteria:**
- Transaction found and displayed within 150ms (p95)
- All transaction details accurately displayed
- Links to related entities work correctly
- Error handling graceful (transaction not found)

**Alternative Flows:**
- **Transaction not found:** Display "Transaction not found. It may not be indexed yet."
- **Invalid format:** Display "Invalid transaction hash format. Expected 0x followed by 64 hex characters."

---

### Flow 3: View Address Transaction History (Data Exploration Flow)

**Scenario:** User wants to explore all transactions for a given address (sent + received)

**Entry Point:** Transaction detail view (clicked on from address) or direct search

**Steps:**

1. **Enter Address**
   - User enters Ethereum address: `0xabcd...1234`
   - Or clicks address link from transaction detail

2. **API Request**
   - Frontend calls `GET /v1/address/{addr}/txs?limit=25&offset=0`
   - Backend queries composite indexes for address
   - Returns transactions where address is sender OR receiver

3. **Display Transaction List**
   - Header: "Transactions for 0xabcd...1234"
   - Table with columns:
     - Transaction Hash (truncated, hover for full)
     - Direction (→ Sent or ← Received)
     - Counterparty Address (truncated)
     - Value (in ETH)
     - Block Height (clickable)
     - Timestamp
   - Pagination controls at bottom (← Previous | Next →)

4. **Pagination**
   - User clicks "Next" to see transactions 26-50
   - Frontend calls `GET /v1/address/{addr}/txs?limit=25&offset=25`
   - Table updates with new results

5. **Drill Down** (Optional)
   - User clicks transaction hash → transaction detail view
   - User clicks counterparty address → that address's transaction history

**Success Criteria:**
- Address history loads within 150ms for typical addresses
- Pagination works smoothly
- Direction (sent/received) clearly indicated
- All values accurately displayed

**Alternative Flows:**
- **No transactions found:** Display "No transactions found for this address."
- **Invalid address:** Display "Invalid Ethereum address format. Expected 0x followed by 40 hex characters."

---

## 4. Wireframes

### 4.1 Main Page (Default View)

```
┌─────────────────────────────────────────────────────────────────────┐
│ ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ │
│ ┃  🔗 Blockchain Explorer             [Search____________] 🔍  ┃ │
│ ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────┐   │
│ │ 📊 LIVE BLOCKS                                   ⟳ Live     │   │
│ ├─────────────────────────────────────────────────────────────┤   │
│ │ #5000  0x1a2b3c...  12:34:56  42 txs            [Details]  │   │
│ │ #4999  0x9f8e7d...  12:34:44  38 txs            [Details]  │   │
│ │ #4998  0x5c4b3a...  12:34:32  51 txs            [Details]  │   │
│ │ #4997  0x2d1e0f...  12:34:20  29 txs            [Details]  │   │
│ │ #4996  0x8a7b6c...  12:34:08  45 txs            [Details]  │   │
│ │ #4995  0x3f2e1d...  12:33:56  33 txs            [Details]  │   │
│ │ #4994  0x9c8b7a...  12:33:44  47 txs            [Details]  │   │
│ │ #4993  0x6e5d4c...  12:33:32  39 txs            [Details]  │   │
│ │ #4992  0x1a0b9f...  12:33:20  55 txs            [Details]  │   │
│ │ #4991  0x7c6b5a...  12:33:08  41 txs            [Details]  │   │
│ └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────┐   │
│ │ 📝 RECENT TRANSACTIONS                                       │   │
│ ├──────────┬──────────┬──────────┬─────────┬──────────────────┤   │
│ │ Tx Hash  │ From     │ To       │ Value   │ Block            │   │
│ ├──────────┼──────────┼──────────┼─────────┼──────────────────┤   │
│ │ 0x1a2b.. │ 0xabcd.. │ 0x1234.. │ 0.5 ETH │ #5000            │   │
│ │ 0x3c4d.. │ 0xef12.. │ 0x5678.. │ 1.2 ETH │ #5000            │   │
│ │ 0x5e6f.. │ 0x9abc.. │ 0xdef0.. │ 0.1 ETH │ #4999            │   │
│ │ 0x7g8h.. │ 0x3456.. │ 0x789a.. │ 2.5 ETH │ #4999            │   │
│ │ 0x9i0j.. │ 0xbcde.. │ 0xf012.. │ 0.3 ETH │ #4998            │   │
│ │ ... (20 more rows) ...                                      │   │
│ ├─────────────────────────────────────────────────────────────┤   │
│ │                    [← Previous] [Next →]                    │   │
│ └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────┐   │
│ │ 📈 STATS                                                     │   │
│ │ Latest Block: #5000  |  Indexed: 5,000  |  Lag: 1 block    │   │
│ └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│                   GitHub | Metrics | Health Check                  │
└─────────────────────────────────────────────────────────────────────┘
```

**Layout Zones:**
- **Header (60px):** Logo/title on left, search bar on right
- **Live Blocks Ticker (300px):** Auto-scrolling list of recent blocks
- **Recent Transactions Table (400px):** Paginated table
- **Stats Footer (60px):** Key metrics summary
- **Footer (40px):** Links

---

### 4.2 Transaction Detail View

```
┌─────────────────────────────────────────────────────────────────────┐
│ ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ │
│ ┃  🔗 Blockchain Explorer             [Search____________] 🔍  ┃ │
│ ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ │
│                                                                     │
│  ← Back to main                                                    │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────┐   │
│ │ 💳 TRANSACTION DETAILS                                       │   │
│ ├─────────────────────────────────────────────────────────────┤   │
│ │                                                              │   │
│ │  Status:           ✓ Success                                │   │
│ │                                                              │   │
│ │  Transaction Hash: 0x1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8... │   │
│ │                                                              │   │
│ │  Block:            #5000 (View Block)                       │   │
│ │  Timestamp:        2025-10-29 12:34:56 UTC                  │   │
│ │                                                              │   │
│ │  From:             0xabcdef1234567890abcdef1234567890abcd   │   │
│ │                    (View Address History)                   │   │
│ │                                                              │   │
│ │  To:               0x1234567890abcdef1234567890abcdef1234   │   │
│ │                    (View Address History)                   │   │
│ │                                                              │   │
│ │  Value:            0.5 ETH (500,000,000,000,000,000 wei)    │   │
│ │  Transaction Fee:  0.000021 ETH                             │   │
│ │                                                              │   │
│ │  Gas Used:         21,000                                   │   │
│ │  Gas Price:        1 gwei                                   │   │
│ │  Nonce:            42                                       │   │
│ │                                                              │   │
│ └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Search Another]                                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

### 4.3 Address Transaction History View

```
┌─────────────────────────────────────────────────────────────────────┐
│ ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ │
│ ┃  🔗 Blockchain Explorer             [Search____________] 🔍  ┃ │
│ ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ │
│                                                                     │
│  ← Back to main                                                    │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────┐   │
│ │ 👤 ADDRESS TRANSACTION HISTORY                               │   │
│ ├─────────────────────────────────────────────────────────────┤   │
│ │                                                              │   │
│ │  Address: 0xabcdef1234567890abcdef1234567890abcdef1234      │   │
│ │  Total Transactions: 127                                    │   │
│ │  Showing: 1-25 of 127                                       │   │
│ │                                                              │   │
│ ├────┬───────────┬──────────┬──────────┬─────────┬───────────┤   │
│ │ Dir│ Tx Hash   │ Counter  │ Value    │ Block   │ Time      │   │
│ ├────┼───────────┼──────────┼─────────┼───────────┼──────────┤   │
│ │ →  │ 0x1a2b..  │ 0x1234.. │ 0.5 ETH  │ #5000    │ 12:34:56 │   │
│ │ ←  │ 0x3c4d..  │ 0x5678.. │ 1.2 ETH  │ #4998    │ 12:33:44 │   │
│ │ →  │ 0x5e6f..  │ 0x9abc.. │ 0.1 ETH  │ #4995    │ 12:32:20 │   │
│ │ ←  │ 0x7g8h..  │ 0xdef0.. │ 2.5 ETH  │ #4990    │ 12:31:08 │   │
│ │ →  │ 0x9i0j..  │ 0x3456.. │ 0.3 ETH  │ #4987    │ 12:29:44 │   │
│ │ ... (20 more rows) ...                                      │   │
│ ├─────────────────────────────────────────────────────────────┤   │
│ │                  [← Previous] Page 1 of 6 [Next →]          │   │
│ └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Legend:**
- `→` = Sent (address is sender)
- `←` = Received (address is receiver)

---

## 5. UI Component Specifications

### 5.1 Header Component

**Purpose:** Navigation and search

**Elements:**
- Logo/Title: "🔗 Blockchain Explorer" (left, 24px font)
- Search Bar: Text input (center-right, 400px width)
- Search Button: Magnifying glass icon (right of search bar)

**States:**
- **Default:** Search bar empty with placeholder text
- **Focused:** Search bar highlighted with border
- **Searching:** Disabled with loading spinner
- **Error:** Red border with error message below

**Styling:**
- Background: Dark blue (#1a1f2e)
- Text: White (#ffffff)
- Search bar: Light background (#2a2f3e) with rounded corners
- Height: 60px
- Fixed position at top

---

### 5.2 Live Blocks Ticker Component

**Purpose:** Display most recent 10 blocks with real-time updates

**Elements:**
- Section header: "📊 LIVE BLOCKS" with green "⟳ Live" indicator
- Block rows (10 rows):
  - Block height (e.g., #5000)
  - Block hash (truncated: 0x1a2b3c...)
  - Timestamp (HH:MM:SS format)
  - Transaction count (e.g., "42 txs")
  - "Details" button

**Behavior:**
- WebSocket connection receives new block → prepend to list
- Oldest block (11th) removed automatically
- New block fades in with green highlight (1 second)
- "Live" indicator pulses green when updates received
- If WebSocket disconnects, show orange "⚠ Reconnecting..."

**Update Animation:**
- New block fades in from top (300ms ease-in)
- Existing blocks slide down
- Bottom block fades out

**States:**
- **Loading:** Skeleton rows with shimmer effect
- **Connected:** Green "Live" indicator
- **Disconnected:** Orange "Reconnecting..." with retry countdown
- **Error:** Red "Connection failed" with retry button

**Styling:**
- Background: Light grey (#f5f5f5)
- Borders: 1px solid #ddd
- New block highlight: Light green (#e8f5e9) fading to white
- Monospace font for hashes and heights

---

### 5.3 Recent Transactions Table Component

**Purpose:** Display paginated list of recent transactions

**Elements:**
- Section header: "📝 RECENT TRANSACTIONS"
- Table columns:
  - Tx Hash (truncated, clickable)
  - From Address (truncated, clickable)
  - To Address (truncated, clickable)
  - Value (in ETH)
  - Block (clickable)
- Pagination controls:
  - "← Previous" button
  - Page indicator (e.g., "Page 1 of 20")
  - "Next →" button

**Behavior:**
- Fetches from `GET /v1/blocks?limit=25&offset=0` on load
- Clicking "Next" increments offset by 25
- Clicking transaction hash → navigate to transaction detail
- Clicking address → navigate to address history
- Clicking block → navigate to block detail

**States:**
- **Loading:** Skeleton rows with shimmer
- **Loaded:** Display data
- **Error:** "Failed to load transactions. Retry?"
- **Empty:** "No transactions found."

**Pagination States:**
- First page: "Previous" button disabled
- Last page: "Next" button disabled
- Middle pages: Both buttons enabled

**Styling:**
- Table: Full width with alternating row colors
- Headers: Bold, grey background
- Clickable items: Blue underline on hover
- Truncated values: Ellipsis with full value on hover (tooltip)

---

### 5.4 Search Bar Component

**Purpose:** Accept user input for searching blocks, transactions, addresses

**Input Validation:**
- **Block Height:** Numeric (1-999999999)
- **Block Hash:** 0x + 64 hex characters
- **Transaction Hash:** 0x + 64 hex characters
- **Address:** 0x + 40 hex characters

**Behavior:**
1. User types input
2. On Enter or button click:
   - Validate format
   - Detect input type
   - Call appropriate API endpoint
   - Navigate to result view

**Error Handling:**
- Invalid format → Show error message below input
- Not found → Show "Not found" message with retry option

**Placeholder Text:**
"Search by block height, hash, transaction hash, or address"

**Autocomplete:** Off (no suggestions needed for demo)

---

### 5.5 Transaction Detail Panel Component

**Purpose:** Display comprehensive transaction information

**Sections:**
1. **Status Badge:**
   - ✓ Success (green) or ✗ Failed (red)

2. **Transaction Info:**
   - Transaction Hash (full, monospace)
   - Block (clickable link)
   - Timestamp (UTC)

3. **Addresses:**
   - From (clickable, truncated with copy button)
   - To (clickable, truncated with copy button)

4. **Value & Fees:**
   - Value in ETH (primary) + wei (secondary, grey)
   - Transaction fee in ETH

5. **Gas Info:**
   - Gas used
   - Gas price (in gwei)
   - Nonce

**Actions:**
- "← Back to main" link at top
- "View Block" link (goes to block detail)
- "View Address History" links for from/to addresses
- Copy buttons for hashes and addresses

**Styling:**
- Card layout with white background, shadow
- Monospace font for hashes and addresses
- Large, clear labels
- Generous spacing (16px between fields)

---

### 5.6 Address Transaction History Table Component

**Purpose:** Display paginated transaction history for an address

**Elements:**
- Address header (full address, monospace)
- Transaction count summary
- Table columns:
  - Direction (→ sent, ← received)
  - Transaction hash (truncated, clickable)
  - Counterparty address (truncated, clickable)
  - Value (in ETH)
  - Block (clickable)
  - Timestamp
- Pagination controls

**Direction Indicators:**
- `→` (green): Sent from this address
- `←` (blue): Received by this address

**Behavior:**
- Fetches from `GET /v1/address/{addr}/txs?limit=25&offset=0`
- Pagination works same as recent transactions table
- Clicking transaction → transaction detail
- Clicking counterparty address → that address's history

**States:**
- **Loading:** Skeleton rows
- **Loaded:** Display data
- **Error:** "Failed to load address history. Retry?"
- **Empty:** "No transactions found for this address."

---

### 5.7 Stats Footer Component

**Purpose:** Display key system metrics

**Elements:**
- Latest Block: e.g., "#5000"
- Total Indexed: e.g., "5,000 blocks"
- Lag: e.g., "1 block (0.8s)"

**Behavior:**
- Updates every 30 seconds via polling `GET /v1/stats/chain`
- Or updates via WebSocket if available

**Styling:**
- Small text, grey background
- Fixed at bottom of live blocks ticker
- Horizontal layout with dividers (|)

---

## 6. Interaction Patterns

### 6.1 Real-Time Updates (WebSocket)

**Connection Lifecycle:**

1. **Page Load:**
   - JavaScript establishes WebSocket connection to `/v1/stream`
   - Sends subscription message: `{"action": "subscribe", "channel": "newBlocks"}`

2. **Connected:**
   - Green "Live" indicator shown
   - Ready to receive updates

3. **Update Received:**
   - WebSocket message: `{"channel": "newBlocks", "data": {...}}`
   - Frontend prepends new block to ticker
   - Animation: fade in + slide down

4. **Disconnected:**
   - Orange "Reconnecting..." indicator
   - Automatic reconnection attempts (exponential backoff: 1s, 2s, 4s, 8s, 16s)
   - If 5 attempts fail, show "Connection failed. Refresh page."

5. **Reconnected:**
   - Green "Live" indicator restored
   - Re-subscribe to channel
   - Fetch missed updates via REST API (if needed)

**Error Handling:**
- Network errors → Automatic retry
- Invalid messages → Log to console, ignore
- Server closes connection → Reconnect attempt

---

### 6.2 Navigation Patterns

**Single-Page Application (SPA) Routing:**

- **Main Page:** `/` or `/#/`
- **Transaction Detail:** `/#/tx/0x1a2b3c...`
- **Block Detail:** `/#/block/5000`
- **Address History:** `/#/address/0xabcd...`

**Navigation Methods:**

1. **Direct URL:** User can bookmark or share URLs
2. **Click Links:** Clicking hashes/addresses navigates
3. **Search:** Search results navigate to detail view
4. **Back Button:** Browser back button works (history API)

**Breadcrumbs:**
- Transaction Detail: "← Back to main"
- Address History: "← Back to main"
- Block Detail: "← Back to main"

---

### 6.3 Loading States

**Progressive Loading:**

1. **Initial Page Load:**
   - Show skeleton UI (grey boxes where content will appear)
   - Connect WebSocket
   - Fetch initial data (recent blocks, recent txs)
   - Hydrate UI with data

2. **Search:**
   - Disable search input
   - Show spinner in search button
   - Fetch data
   - Navigate to result

3. **Pagination:**
   - Disable pagination buttons
   - Show loading spinner in table
   - Fetch next page
   - Replace table content

**Skeleton UI:**
- Grey boxes with shimmer animation (light gradient moving left to right)
- Same dimensions as actual content
- Indicates loading without blocking UI

---

### 6.4 Error Handling

**Error Display Patterns:**

1. **Inline Errors (Form Validation):**
   - Red border on input
   - Error message below input (small, red text)
   - Icon: ⚠

2. **Toast Notifications (Transient Errors):**
   - Small popup at top-right
   - Auto-dismiss after 5 seconds
   - Types: Error (red), Warning (orange), Success (green)

3. **Empty States:**
   - Center message with icon
   - Example: "No transactions found for this address."

4. **API Errors:**
   - Connection error → "Failed to connect. Check your network."
   - 404 Not Found → "Transaction not found. It may not be indexed yet."
   - 500 Server Error → "Server error. Please try again."

---

## 7. Visual Design System

### 7.1 Color Palette

**Primary Colors:**
- Dark Blue: `#1a1f2e` (header background)
- Medium Blue: `#2a3f5e` (accents)
- Light Blue: `#4a8fe7` (links, clickable items)

**Status Colors:**
- Success Green: `#4caf50`
- Error Red: `#f44336`
- Warning Orange: `#ff9800`
- Info Blue: `#2196f3`

**Neutral Colors:**
- Background: `#ffffff` (main content)
- Light Grey: `#f5f5f5` (table alternating rows)
- Medium Grey: `#e0e0e0` (borders)
- Dark Grey: `#616161` (secondary text)
- Black: `#212121` (primary text)

**Direction Indicators:**
- Sent (→): `#4caf50` (green)
- Received (←): `#2196f3` (blue)

---

### 7.2 Typography

**Font Stack:**
```css
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto,
               'Helvetica Neue', Arial, sans-serif;
}

.monospace {
  font-family: 'Courier New', Courier, monospace;
}
```

**Font Sizes:**
- Header Title: 24px (bold)
- Section Headers: 18px (bold)
- Body Text: 14px (regular)
- Small Text: 12px (secondary info, footer)
- Table Text: 14px (regular)

**Font Weights:**
- Regular: 400
- Bold: 600

**Line Height:**
- Body: 1.5
- Headers: 1.2
- Tables: 1.4

---

### 7.3 Spacing System

**Base Unit:** 8px

**Spacing Scale:**
- xs: 4px (tight spacing)
- sm: 8px (compact spacing)
- md: 16px (standard spacing)
- lg: 24px (generous spacing)
- xl: 32px (section separation)

**Application:**
- Padding inside cards: 16px
- Margin between sections: 24px
- Table cell padding: 8px 16px
- Input padding: 8px 12px

---

### 7.4 Layout

**Container:**
- Max width: 1200px
- Centered with auto margins
- Padding: 16px on mobile, 24px on desktop

**Grid System:**
- Main content: Single column (mobile), optionally two columns (desktop)
- Live blocks ticker: Full width
- Transactions table: Full width

**Component Sizing:**
- Header: Fixed 60px height
- Live blocks ticker: ~300px height (fixed)
- Transactions table: Flexible height, minimum 400px
- Footer: Fixed 40px height

---

### 7.5 Borders & Shadows

**Borders:**
- Standard: 1px solid #e0e0e0
- Input focus: 2px solid #4a8fe7
- Error: 2px solid #f44336

**Border Radius:**
- Buttons: 4px
- Cards: 8px
- Inputs: 4px

**Box Shadows:**
- Cards: `0 2px 4px rgba(0,0,0,0.1)`
- Hover: `0 4px 8px rgba(0,0,0,0.15)`
- Modal: `0 8px 16px rgba(0,0,0,0.2)`

---

## 8. Responsive Design

### 8.1 Breakpoints

**Mobile:** 0-767px
**Tablet:** 768-1023px
**Desktop:** 1024px+

### 8.2 Mobile Adaptations (< 768px)

**Header:**
- Logo text shortened to "Explorer"
- Search bar full width below logo (stacked layout)

**Live Blocks Ticker:**
- Reduce to 5 blocks instead of 10
- Truncate hashes more aggressively (0x1a2b...)
- Stack info vertically per block

**Transactions Table:**
- Convert to card layout (vertical stack)
- Each transaction becomes a card:
  ```
  ┌─────────────────────────┐
  │ 0x1a2b3c...            │
  │ From: 0xabcd...        │
  │ To: 0x1234...          │
  │ Value: 0.5 ETH         │
  │ Block: #5000           │
  └─────────────────────────┘
  ```
- Pagination buttons stacked vertically

**Transaction Detail:**
- Full width cards
- Labels above values (vertical layout)
- Full hashes (no truncation, allow horizontal scroll)

### 8.3 Tablet Adaptations (768-1023px)

**Mostly desktop layout with adjustments:**
- Table columns narrower
- Search bar slightly smaller (300px instead of 400px)
- Live blocks ticker shows 8 blocks

### 8.4 Desktop Optimizations (1024px+)

**Full layout as shown in wireframes:**
- Max width container (1200px)
- Tables use full column set
- Generous spacing

---

## 9. Accessibility

### 9.1 Semantic HTML

**Structure:**
- `<header>` for page header
- `<main>` for primary content
- `<section>` for live blocks and transactions
- `<table>` for tabular data
- `<footer>` for footer links

**ARIA Labels:**
- Search input: `aria-label="Search blockchain data"`
- Live indicator: `aria-live="polite"` (announces updates)
- Pagination: `aria-label="Pagination navigation"`

### 9.2 Keyboard Navigation

**Tab Order:**
1. Search input
2. Search button
3. Block detail buttons (in ticker)
4. Transaction links (in table)
5. Pagination controls
6. Footer links

**Keyboard Shortcuts:**
- `/` - Focus search bar
- `Esc` - Clear search, return to main
- `Enter` - Submit search
- `Tab` - Navigate between interactive elements
- `Space` or `Enter` - Activate buttons/links

### 9.3 Color Contrast

**WCAG AA Compliance:**
- Text on white: #212121 (contrast ratio 16:1 ✓)
- Links: #4a8fe7 (contrast ratio 7:1 ✓)
- Secondary text: #616161 (contrast ratio 7:1 ✓)

### 9.4 Screen Reader Support

**Alt Text:**
- Icons have aria-labels or text alternatives
- Images (if any) have descriptive alt text

**Live Regions:**
- Live blocks ticker: `aria-live="polite"` announces new blocks
- Error messages: `role="alert"` for immediate announcement

### 9.5 Focus Indicators

**Visible Focus:**
- Blue outline (2px solid #4a8fe7) around focused elements
- Never remove focus styles (`:focus-visible` used where appropriate)

---

## 10. Technical Implementation Notes

### 10.1 Technology Stack

**Frontend Only:**
- HTML5 (semantic markup)
- CSS3 (no preprocessor, vanilla CSS)
- JavaScript ES6+ (no framework, vanilla JS)

**No Build Tools:**
- No webpack, Babel, npm scripts
- Files served directly from `web/` directory
- Browser-native modules if needed (ESM)

### 10.2 File Structure

```
web/
├── index.html         # Single page, all UI markup
├── style.css          # All styles in one file
└── app.js             # All JavaScript logic
```

**Rationale:** Simplicity over modularity for demo scope.

### 10.3 API Integration

**REST API Calls:**
```javascript
async function fetchBlocks(limit = 10, offset = 0) {
  const response = await fetch(`/v1/blocks?limit=${limit}&offset=${offset}`);
  if (!response.ok) throw new Error('Failed to fetch blocks');
  return await response.json();
}
```

**WebSocket Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/v1/stream');

ws.onopen = () => {
  ws.send(JSON.stringify({action: 'subscribe', channel: 'newBlocks'}));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.channel === 'newBlocks') {
    updateLiveBlocksTicker(message.data);
  }
};
```

### 10.4 State Management

**Simple JavaScript Objects:**
```javascript
const state = {
  blocks: [],
  transactions: [],
  currentView: 'main',
  searchQuery: '',
  isLoading: false,
  wsConnected: false
};
```

**No Complex State Library:** Use vanilla JS with simple state object.

### 10.5 Routing

**Hash-Based Routing:**
```javascript
window.addEventListener('hashchange', () => {
  const hash = window.location.hash;
  if (hash.startsWith('#/tx/')) {
    showTransactionDetail(hash.replace('#/tx/', ''));
  } else if (hash.startsWith('#/address/')) {
    showAddressHistory(hash.replace('#/address/', ''));
  } else {
    showMainView();
  }
});
```

### 10.6 Performance Considerations

**Optimizations:**
- Debounce search input (300ms delay)
- Limit live blocks ticker to 10 blocks (memory efficiency)
- Reuse table rows for pagination (DOM reuse)
- Lazy load transaction details (fetch only when clicked)

**Bundle Size:**
- HTML: ~10KB
- CSS: ~15KB
- JavaScript: ~20KB
- **Total:** ~45KB (uncompressed, no minification needed for demo)

---

## 11. Future Enhancements (Out of Scope for MVP)

**If Time Permits (Day 7):**
- Dark mode toggle
- Better animations (e.g., number counters, chart sparklines)
- Copy-to-clipboard buttons with toast notifications
- Export transaction history as CSV

**Post-MVP (Not for 7-Day Sprint):**
- Mobile app (React Native)
- Advanced charts (gas price trends, transaction volume)
- ERC-20 token transfers display
- Smart contract verification integration

---

## 12. Success Metrics

**UX Success Criteria:**

1. **Usability:**
   - Evaluator can use all features within 5 minutes without documentation
   - Search yields correct results 100% of the time
   - No broken links or 404 errors

2. **Performance:**
   - Initial page load: <2 seconds
   - API responses: <150ms (p95)
   - WebSocket latency: <100ms

3. **Reliability:**
   - No JavaScript errors in console
   - WebSocket reconnects automatically on disconnect
   - Graceful error handling for all failure scenarios

4. **Impression:**
   - Evaluator perceives system as "production-ready"
   - UI demonstrates competency despite minimal styling
   - Real-time updates create "wow" moment

---

## Document Status

✅ **Complete and Ready for Implementation**

**Artifacts Defined:**
- [x] User personas (2)
- [x] User flows (3 primary flows)
- [x] Wireframes (3 views)
- [x] Component specifications (7 components)
- [x] Interaction patterns
- [x] Visual design system (colors, typography, spacing)
- [x] Responsive design breakpoints
- [x] Accessibility guidelines
- [x] Technical implementation notes

**Next Steps:**
1. Begin frontend implementation (Day 5 of sprint)
2. Use this spec as reference for HTML/CSS/JS structure
3. Test against user flows to ensure completeness
4. Validate accessibility with screen reader and keyboard navigation

---

**Document Generated:** 2025-10-29
**Author:** Winston (Architect Agent)
**Reviewed Against:** PRD.md, solution-architecture.md, epic-stories.md
**Validation Status:** ✅ Meets Level 2 UI project requirements
