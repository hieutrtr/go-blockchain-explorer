// Configuration constants
const RECONNECT_BASE_DELAY_MS = 1000;
const RECONNECT_MAX_DELAY_MS = 8000;
const TIMESTAMP_UPDATE_INTERVAL_MS = 10000;
const API_RETRY_DELAY_MS = 2000;
const DEBOUNCE_DELAY_MS = 300;
const CACHE_MAX_SIZE = 20;

// Global state
let ws;
let reconnectTimer;
let reconnectAttempts = 0;
let blocks = [];
let transactions = [];
let timestampInterval;

// Pagination state
let blocksPage = 1;
let blocksLimit = 10;
let blocksTotal = 0;

let transactionsPage = 1;
let transactionsLimit = 25; // Real transaction limit from API
let transactionsTotal = 0; // Will be updated from API

// Routing state
let currentRoute = null;
let currentView = 'home';

// LRU Cache for API responses
const cache = {
    data: new Map(),
    keys: [],

    get(key) {
        if (!this.data.has(key)) return null;
        // Move to end (most recently used)
        this.keys = this.keys.filter(k => k !== key);
        this.keys.push(key);
        return this.data.get(key);
    },

    set(key, value) {
        if (this.data.has(key)) {
            // Update existing
            this.keys = this.keys.filter(k => k !== key);
        } else if (this.keys.length >= CACHE_MAX_SIZE) {
            // Evict oldest
            const oldest = this.keys.shift();
            this.data.delete(oldest);
        }
        this.data.set(key, value);
        this.keys.push(key);
    },

    has(key) {
        return this.data.has(key);
    },

    clear() {
        this.data.clear();
        this.keys = [];
    },

    invalidateBlocks() {
        // Invalidate all block-related cache entries
        const blockKeys = this.keys.filter(k => k.startsWith('block/'));
        blockKeys.forEach(k => {
            this.data.delete(k);
            this.keys = this.keys.filter(key => key !== k);
        });
    }
};

// Utility functions
function truncateHash(hash) {
    if (!hash || hash.length < 14) return hash;
    return `${hash.substring(0, 6)}...${hash.substring(hash.length - 4)}`;
}

function formatTimestamp(timestamp) {
    const now = Math.floor(Date.now() / 1000);
    const diff = now - timestamp;

    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
}

function formatEth(valueWei, fullPrecision = false) {
    if (!valueWei || valueWei === '0') return fullPrecision ? '0' : '0.0000';
    const eth = parseInt(valueWei) / 1e18;
    return fullPrecision ? eth.toString() : eth.toFixed(4);
}

// Router system
const router = {
    parseRoute() {
        const hash = window.location.hash.slice(1) || '/';
        const parts = hash.split('/').filter(p => p);

        if (parts.length === 0 || hash === '/') {
            return { view: 'home', params: {} };
        }

        if (parts[0] === 'block' && parts.length === 2) {
            return { view: 'blockDetail', params: { id: parts[1] } };
        }

        if (parts[0] === 'tx' && parts.length === 2) {
            return { view: 'txDetail', params: { hash: parts[1] } };
        }

        if (parts[0] === 'address' && parts.length === 2) {
            return { view: 'addressHistory', params: { address: parts[1] } };
        }

        return { view: '404', params: {} };
    },

    navigate(path) {
        window.location.hash = path;
    },

    updateTitle(title) {
        document.title = title ? `${title} - Blockchain Explorer` : 'Blockchain Explorer - Ethereum Sepolia Testnet';
    },

    renderView(route) {
        currentRoute = route;
        currentView = route.view;

        // Hide all containers
        const homeView = document.getElementById('home-view');
        const detailView = document.getElementById('detail-view');

        if (homeView) homeView.style.display = 'none';
        if (detailView) detailView.style.display = 'none';

        switch (route.view) {
            case 'home':
                router.renderHome();
                break;
            case 'blockDetail':
                router.renderBlockDetail(route.params.id);
                break;
            case 'txDetail':
                router.renderTxDetail(route.params.hash);
                break;
            case 'addressHistory':
                router.renderAddressHistory(route.params.address);
                break;
            case '404':
                router.render404();
                break;
        }
    },

    renderHome() {
        const homeView = document.getElementById('home-view');
        if (homeView) {
            homeView.style.display = 'block';
        }
        router.updateTitle('');
    },

    async renderBlockDetail(id) {
        const detailView = document.getElementById('detail-view');
        if (!detailView) return;

        detailView.style.display = 'block';
        detailView.innerHTML = '<div class="loading">Loading block details...</div>';

        const cacheKey = `block/${id}`;
        let block = cache.get(cacheKey);

        if (!block) {
            try {
                const response = await fetch(`/v1/blocks/${id}`);
                if (!response.ok) {
                    if (response.status === 404) {
                        detailView.innerHTML = '<div class="error-message">Block not found</div>';
                        router.updateTitle('Block Not Found');
                        return;
                    }
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                block = await response.json();
                cache.set(cacheKey, block);
            } catch (error) {
                console.error('Error fetching block:', error);
                detailView.innerHTML = '<div class="error-message">Error loading block details. <button onclick="location.reload()">Retry</button></div>';
                return;
            }
        }

        router.updateTitle(`Block #${block.height}`);

        detailView.innerHTML = `
            <div class="detail-header">
                <button class="back-button" onclick="router.navigate('/')">← Back</button>
                <h2>Block #${block.height}</h2>
            </div>
            <div class="detail-content">
                <dl class="detail-list">
                    <dt>Hash</dt>
                    <dd><span class="hash-full">${block.hash}</span> <button class="copy-btn" data-copy="${block.hash}">Copy</button></dd>

                    <dt>Parent Hash</dt>
                    <dd><span class="hash-full">${block.parent_hash}</span> <button class="copy-btn" data-copy="${block.parent_hash}">Copy</button></dd>

                    <dt>Timestamp</dt>
                    <dd>${new Date(block.timestamp * 1000).toLocaleString()} (${formatTimestamp(block.timestamp)})</dd>

                    <dt>Miner</dt>
                    <dd><a href="#/address/${block.miner}" class="address-link">${block.miner}</a> <button class="copy-btn" data-copy="${block.miner}">Copy</button></dd>

                    <dt>Gas Used / Limit</dt>
                    <dd>${block.gas_used.toLocaleString()} / ${block.gas_limit.toLocaleString()}</dd>

                    <dt>Transaction Count</dt>
                    <dd>${block.tx_count}</dd>
                </dl>

                ${block.transactions && block.transactions.length > 0 ? `
                    <h3>Transactions</h3>
                    <div class="transactions-list">
                        ${block.transactions.map(txHash => `
                            <div class="transaction-item">
                                <a href="#/tx/${txHash}" class="tx-link">${truncateHash(txHash)}</a>
                            </div>
                        `).join('')}
                    </div>
                ` : '<p class="no-data">No transactions in this block</p>'}
            </div>
        `;

        // Attach copy button listeners
        attachCopyListeners();
    },

    async renderTxDetail(hash) {
        const detailView = document.getElementById('detail-view');
        if (!detailView) return;

        detailView.style.display = 'block';
        detailView.innerHTML = '<div class="loading">Loading transaction details...</div>';

        const cacheKey = `tx/${hash}`;
        let tx = cache.get(cacheKey);

        if (!tx) {
            try {
                const response = await fetch(`/v1/txs/${hash}`);
                if (!response.ok) {
                    if (response.status === 404) {
                        detailView.innerHTML = '<div class="error-message">Transaction not found</div>';
                        router.updateTitle('Transaction Not Found');
                        return;
                    }
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                tx = await response.json();
                cache.set(cacheKey, tx);
            } catch (error) {
                console.error('Error fetching transaction:', error);
                detailView.innerHTML = '<div class="error-message">Error loading transaction details. <button onclick="location.reload()">Retry</button></div>';
                return;
            }
        }

        router.updateTitle(`Transaction ${truncateHash(hash)}`);

        detailView.innerHTML = `
            <div class="detail-header">
                <button class="back-button" onclick="router.navigate('/')">← Back</button>
                <h2>Transaction Details</h2>
            </div>
            <div class="detail-content">
                <dl class="detail-list">
                    <dt>Transaction Hash</dt>
                    <dd><span class="hash-full">${tx.hash}</span> <button class="copy-btn" data-copy="${tx.hash}">Copy</button></dd>

                    <dt>Status</dt>
                    <dd><span class="status-badge ${tx.success ? 'success' : 'failed'}">${tx.success ? 'Success' : 'Failed'}</span></dd>

                    <dt>Block</dt>
                    <dd><a href="#/block/${tx.block_height}" class="block-link">#${tx.block_height}</a></dd>

                    <dt>From</dt>
                    <dd><a href="#/address/${tx.from_addr}" class="address-link">${tx.from_addr}</a> <button class="copy-btn" data-copy="${tx.from_addr}">Copy</button></dd>

                    <dt>To</dt>
                    <dd>${tx.to_addr ? `<a href="#/address/${tx.to_addr}" class="address-link">${tx.to_addr}</a> <button class="copy-btn" data-copy="${tx.to_addr}">Copy</button>` : '<span class="contract-creation">Contract Creation</span>'}</dd>

                    <dt>Value</dt>
                    <dd>${formatEth(tx.value_wei, true)} ETH</dd>

                    <dt>Transaction Fee</dt>
                    <dd>${formatEth(tx.fee_wei, true)} ETH</dd>

                    <dt>Gas Used</dt>
                    <dd>${tx.gas_used.toLocaleString()}</dd>

                    <dt>Gas Price</dt>
                    <dd>${(tx.gas_price / 1e9).toFixed(2)} Gwei</dd>

                    <dt>Nonce</dt>
                    <dd>${tx.nonce}</dd>

                    <dt>Transaction Index</dt>
                    <dd>${tx.tx_index}</dd>
                </dl>
            </div>
        `;

        attachCopyListeners();
    },

    async renderAddressHistory(address, page = 1) {
        const detailView = document.getElementById('detail-view');
        if (!detailView) return;

        detailView.style.display = 'block';
        if (page === 1) {
            detailView.innerHTML = '<div class="loading">Loading address history...</div>';
        }

        const limit = 50;
        const offset = (page - 1) * limit;
        const cacheKey = `address/${address}/${page}`;
        let data = cache.get(cacheKey);

        if (!data) {
            try {
                const response = await fetch(`/v1/address/${address}/txs?limit=${limit}&offset=${offset}`);
                if (!response.ok) {
                    if (response.status === 404) {
                        detailView.innerHTML = '<div class="error-message">Address not found or no transactions</div>';
                        router.updateTitle('Address Not Found');
                        return;
                    }
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                data = await response.json();
                cache.set(cacheKey, data);
            } catch (error) {
                console.error('Error fetching address history:', error);
                detailView.innerHTML = '<div class="error-message">Error loading address history. <button onclick="location.reload()">Retry</button></div>';
                return;
            }
        }

        router.updateTitle(`Address ${truncateHash(address)}`);

        const totalPages = Math.ceil(data.total / limit);

        detailView.innerHTML = `
            <div class="detail-header">
                <button class="back-button" onclick="router.navigate('/')">← Back</button>
                <h2>Address Transaction History</h2>
            </div>
            <div class="detail-content">
                <div class="address-info">
                    <span class="hash-full">${address}</span> <button class="copy-btn" data-copy="${address}">Copy</button>
                    <p class="tx-count-info">Total Transactions: ${data.total}</p>
                </div>

                ${data.transactions && data.transactions.length > 0 ? `
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>Transaction Hash</th>
                                <th>Block</th>
                                <th>From</th>
                                <th>To</th>
                                <th>Value (ETH)</th>
                                <th>Age</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${data.transactions.map(tx => `
                                <tr>
                                    <td><a href="#/tx/${tx.hash}" class="hash-truncated">${truncateHash(tx.hash)}</a></td>
                                    <td><a href="#/block/${tx.block_height}" class="block-number">#${tx.block_height}</a></td>
                                    <td><a href="#/address/${tx.from_addr}" class="hash-truncated ${tx.from_addr.toLowerCase() === address.toLowerCase() ? 'highlight' : ''}">${truncateHash(tx.from_addr)}</a></td>
                                    <td>${tx.to_addr ? `<a href="#/address/${tx.to_addr}" class="hash-truncated ${tx.to_addr.toLowerCase() === address.toLowerCase() ? 'highlight' : ''}">${truncateHash(tx.to_addr)}</a>` : 'Contract'}</td>
                                    <td class="tx-value">${formatEth(tx.value_wei)}</td>
                                    <td>${formatTimestamp(tx.timestamp)}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                    <div class="pagination">
                        <button class="pagination-btn" ${page <= 1 ? 'disabled' : ''} onclick="router.renderAddressHistory('${address}', ${page - 1})">← Previous</button>
                        <span class="page-info">Page ${page} of ${totalPages}</span>
                        <button class="pagination-btn" ${page >= totalPages ? 'disabled' : ''} onclick="router.renderAddressHistory('${address}', ${page + 1})">Next →</button>
                    </div>
                ` : '<p class="no-data">No transactions found for this address</p>'}
            </div>
        `;

        attachCopyListeners();
    },

    render404() {
        const detailView = document.getElementById('detail-view');
        if (!detailView) return;

        detailView.style.display = 'block';
        detailView.innerHTML = `
            <div class="detail-header">
                <button class="back-button" onclick="router.navigate('/')">← Back</button>
                <h2>Page Not Found</h2>
            </div>
            <div class="detail-content">
                <p class="error-message">The requested page could not be found.</p>
            </div>
        `;
        router.updateTitle('Page Not Found');
    }
};

// Copy to clipboard functionality
function attachCopyListeners() {
    document.querySelectorAll('.copy-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            const text = e.target.getAttribute('data-copy');
            try {
                await navigator.clipboard.writeText(text);
                showToast('Copied to clipboard!');
            } catch (err) {
                // Fallback for older browsers
                const textarea = document.createElement('textarea');
                textarea.value = text;
                document.body.appendChild(textarea);
                textarea.select();
                document.execCommand('copy');
                document.body.removeChild(textarea);
                showToast('Copied to clipboard!');
            }
        });
    });
}

function showToast(message) {
    let toast = document.getElementById('toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'toast';
        toast.className = 'toast';
        document.body.appendChild(toast);
    }

    toast.textContent = message;
    toast.classList.add('show');

    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

// Search functionality
let searchDebounceTimer;

function debounce(func, delay) {
    return function(...args) {
        clearTimeout(searchDebounceTimer);
        searchDebounceTimer = setTimeout(() => func.apply(this, args), delay);
    };
}

function detectSearchType(input) {
    const cleaned = input.trim().toLowerCase();

    // Transaction hash: 0x followed by 64 hex characters
    if (/^0x[0-9a-f]{64}$/i.test(cleaned)) {
        return { type: 'transaction', value: cleaned };
    }

    // Address: 0x followed by 40 hex characters
    if (/^0x[0-9a-f]{40}$/i.test(cleaned)) {
        return { type: 'address', value: cleaned };
    }

    // Block height: digits only
    if (/^\d+$/.test(cleaned)) {
        return { type: 'block', value: cleaned };
    }

    return { type: 'unknown', value: cleaned };
}

async function performSearch(query) {
    if (!query || query.length < 3) return;

    const searchLoading = document.getElementById('search-loading');
    const searchClear = document.getElementById('search-clear');

    searchLoading.style.display = 'block';
    searchClear.style.display = 'none';

    const detected = detectSearchType(query);

    try {
        switch (detected.type) {
            case 'transaction':
                // Navigate to transaction detail page
                router.navigate(`/tx/${detected.value}`);
                break;

            case 'address':
                // Navigate to address history page
                router.navigate(`/address/${detected.value}`);
                break;

            case 'block':
                // Navigate to block detail page
                router.navigate(`/block/${detected.value}`);
                break;

            default:
                showToast('Invalid search query. Please enter a transaction hash, address, or block number.');
        }
    } catch (error) {
        console.error('Search error:', error);
        showToast('Search error. Please try again.');
    } finally {
        searchLoading.style.display = 'none';
        searchClear.style.display = query ? 'block' : 'none';
    }
}

const debouncedSearch = debounce(performSearch, DEBOUNCE_DELAY_MS);

function initializeSearch() {
    const searchInput = document.getElementById('search-input');
    const searchClear = document.getElementById('search-clear');

    if (!searchInput) return;

    // Input event with debounce
    searchInput.addEventListener('input', (e) => {
        const value = e.target.value;
        searchClear.style.display = value ? 'block' : 'none';

        // Only search if Enter hasn't been pressed (debounce for typing)
        if (!e.inputType || e.inputType !== 'insertLineBreak') {
            debouncedSearch(value);
        }
    });

    // Enter key - immediate search
    searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            clearTimeout(searchDebounceTimer); // Cancel debounce
            performSearch(searchInput.value);
        } else if (e.key === 'Escape') {
            searchInput.value = '';
            searchClear.style.display = 'none';
            searchInput.blur();
        }
    });

    // Clear button
    searchClear.addEventListener('click', () => {
        searchInput.value = '';
        searchClear.style.display = 'none';
        searchInput.focus();
    });

    // Global "/" key to focus search
    document.addEventListener('keydown', (e) => {
        // Only trigger if not already typing in an input
        if (e.key === '/' && !['INPUT', 'TEXTAREA'].includes(document.activeElement.tagName)) {
            e.preventDefault();
            searchInput.focus();
        }
    });
}

// Connection status management
function updateConnectionStatus(connected) {
    const statusDot = document.getElementById('connection-status');
    const statusText = document.getElementById('connection-text');

    if (connected) {
        statusDot.classList.remove('disconnected');
        statusDot.classList.add('connected');
        statusText.textContent = 'Connected';
    } else {
        statusDot.classList.remove('connected');
        statusDot.classList.add('disconnected');
        statusText.textContent = 'Disconnected';
    }
}

// WebSocket connection management
function connectWebSocket() {
    const wsUrl = `ws://${window.location.host}/v1/stream`;
    console.log('Connecting to WebSocket:', wsUrl);

    updateConnectionStatus(false);

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('WebSocket connected');
        updateConnectionStatus(true);
        reconnectAttempts = 0;

        // Subscribe to newBlocks channel (correct protocol format)
        ws.send(JSON.stringify({
            action: 'subscribe',
            channels: ['newBlocks']
        }));
    };

    ws.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            console.log('WebSocket message:', message);

            if (message.type === 'newBlock') {
                handleNewBlock(message.data);
            }
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        updateConnectionStatus(false);
    };

    ws.onclose = () => {
        console.log('WebSocket closed');
        updateConnectionStatus(false);

        // Exponential backoff: 1s, 2s, 4s, 8s, max 8s
        const delay = Math.min(RECONNECT_BASE_DELAY_MS * Math.pow(2, reconnectAttempts), RECONNECT_MAX_DELAY_MS);
        reconnectAttempts++;

        console.log(`Reconnecting in ${delay / 1000}s... (attempt ${reconnectAttempts})`);
        reconnectTimer = setTimeout(connectWebSocket, delay);
    };
}

// API functions
async function fetchBlocks(page = 1) {
    try {
        blocksPage = page;
        const offset = (page - 1) * blocksLimit;
        const response = await fetch(`/v1/blocks?limit=${blocksLimit}&offset=${offset}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        console.log('Fetched blocks:', data);

        if (data.blocks && data.blocks.length > 0) {
            blocks = data.blocks;
            blocksTotal = data.total || 0;
            renderBlocks();
            updateBlocksPagination();

            // Fetch transactions from the latest block (only on first page)
            if (page === 1) {
                await fetchTransactions();
            }
        }
    } catch (error) {
        console.error('Error fetching blocks:', error);
        // Retry once
        setTimeout(() => fetchBlocks(page), API_RETRY_DELAY_MS);
    }
}

async function fetchTransactions() {
    try {
        // Fetch recent blocks and extract their transactions
        const blocksResponse = await fetch(`${API_BASE}/v1/blocks?limit=5&offset=0`);
        if (!blocksResponse.ok) {
            throw new Error(`HTTP ${blocksResponse.status}`);
        }

        const blocksData = await blocksResponse.json();
        const allTransactions = [];

        // Fetch transactions from each recent block
        for (const block of (blocksData.blocks || [])) {
            if (block.tx_count === 0) continue;

            try {
                // Fetch transactions for this block
                const txResponse = await fetch(
                    `${API_BASE}/v1/blocks/${block.height}/transactions?limit=100`
                );

                if (!txResponse.ok) {
                    console.warn(`Failed to fetch transactions for block ${block.height}`);
                    continue;
                }

                const txData = await txResponse.json();
                if (txData.transactions) {
                    allTransactions.push(...txData.transactions);
                }

                // Stop once we have enough transactions
                if (allTransactions.length >= transactionsLimit) break;
            } catch (err) {
                console.warn(`Error fetching transactions for block ${block.height}:`, err);
            }
        }

        transactions = allTransactions.slice(0, transactionsLimit);
        transactionsTotal = transactions.length;
        transactionsPage = 1; // Reset to first page
        renderTransactions();

    } catch (error) {
        console.error('Failed to fetch recent transactions:', error);

        // Retry once after delay
        setTimeout(fetchTransactions, API_RETRY_DELAY_MS);
    }
}

// Render functions
function renderBlocks() {
    const blocksTable = document.getElementById('blocks-table');
    const blocksEmpty = document.getElementById('blocks-empty');
    const blocksPagination = document.getElementById('blocks-pagination');
    const tbody = document.getElementById('blocks-tbody');

    if (blocks.length === 0) {
        blocksTable.style.display = 'none';
        blocksEmpty.style.display = 'block';
        blocksPagination.style.display = 'none';
        return;
    }

    blocksTable.style.display = 'table';
    blocksEmpty.style.display = 'none';
    blocksPagination.style.display = 'flex';

    tbody.innerHTML = blocks.map(block => `
        <tr>
            <td><a href="#/block/${block.height}" class="block-number">#${block.height}</a></td>
            <td><a href="#/block/${block.hash}" class="hash-truncated">${truncateHash(block.hash)}</a></td>
            <td><span class="block-age">${formatTimestamp(block.timestamp)}</span></td>
            <td><span class="tx-count">${block.tx_count || 0}</span></td>
        </tr>
    `).join('');
}

function renderTransactions() {
    const txTable = document.getElementById('transactions-table');
    const txEmpty = document.getElementById('transactions-empty');
    const txPagination = document.getElementById('transactions-pagination');
    const tbody = document.getElementById('transactions-tbody');

    if (transactions.length === 0) {
        txTable.style.display = 'none';
        txEmpty.style.display = 'block';
        txPagination.style.display = 'none';
        return;
    }

    txTable.style.display = 'table';
    txEmpty.style.display = 'none';
    txPagination.style.display = 'flex';

    tbody.innerHTML = transactions.map(tx => `
        <tr>
            <td><a href="#/tx/${tx.hash}" class="hash-truncated">${truncateHash(tx.hash)}</a></td>
            <td><a href="#/address/${tx.from_addr}" class="hash-truncated">${truncateHash(tx.from_addr)}</a></td>
            <td><a href="#/address/${tx.to_addr}" class="hash-truncated">${truncateHash(tx.to_addr)}</a></td>
            <td><span class="tx-value">${formatEth(tx.value_wei)}</span></td>
            <td><a href="#/block/${tx.block_height}" class="block-number">#${tx.block_height}</a></td>
        </tr>
    `).join('');

    updateTransactionsPagination();
}

// Pagination functions
function updateBlocksPagination() {
    const prevBtn = document.getElementById('blocks-prev');
    const nextBtn = document.getElementById('blocks-next');
    const pageInfo = document.getElementById('blocks-page-info');

    const totalPages = Math.ceil(blocksTotal / blocksLimit);

    prevBtn.disabled = blocksPage <= 1;
    nextBtn.disabled = blocksPage >= totalPages;
    pageInfo.textContent = `Page ${blocksPage} of ${totalPages}`;
}

function updateTransactionsPagination() {
    const prevBtn = document.getElementById('transactions-prev');
    const nextBtn = document.getElementById('transactions-next');
    const pageInfo = document.getElementById('transactions-page-info');

    const totalPages = Math.ceil(transactionsTotal / transactionsLimit);

    prevBtn.disabled = transactionsPage <= 1;
    nextBtn.disabled = transactionsPage >= totalPages;
    pageInfo.textContent = `Page ${transactionsPage} of ${totalPages}`;
}

// Handle new block from WebSocket
function handleNewBlock(block) {
    console.log('New block received:', block);

    // Invalidate block cache on new block
    cache.invalidateBlocks();

    // Prepend new block
    blocks.unshift(block);

    // Limit to 10 blocks
    if (blocks.length > 10) {
        blocks = blocks.slice(0, 10);
    }

    renderBlocks();

    // Fetch new transactions from this block
    fetchTransactionsFromBlock(block.height);
}

async function fetchTransactionsFromBlock(blockHeight) {
    try {
        // Fetch transactions from the newly indexed block
        const txResponse = await fetch(
            `${API_BASE}/v1/blocks/${blockHeight}/transactions?limit=100`
        );

        if (!txResponse.ok) {
            console.warn(`Failed to fetch transactions for block ${blockHeight}`);
            return;
        }

        const txData = await txResponse.json();
        if (!txData.transactions || txData.transactions.length === 0) {
            console.log(`Block ${blockHeight} has no transactions`);
            return;
        }

        // Add new transactions to the front of the list
        transactions = [...txData.transactions, ...transactions].slice(0, transactionsLimit);
        transactionsTotal = transactions.length;
        renderTransactions();

        console.log(`Fetched ${txData.transactions.length} transactions from block ${blockHeight}`);
    } catch (error) {
        console.error(`Error fetching transactions for block ${blockHeight}:`, error);
    }
}

// Update timestamps periodically
function startTimestampUpdater() {
    timestampInterval = setInterval(() => {
        // Update block ages
        const ageElements = document.querySelectorAll('.block-age');
        blocks.forEach((block, index) => {
            if (ageElements[index]) {
                ageElements[index].textContent = formatTimestamp(block.timestamp);
            }
        });
    }, TIMESTAMP_UPDATE_INTERVAL_MS); // Update every 10 seconds
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    console.log('Page loaded, initializing...');

    // Initialize routing
    const handleRoute = () => {
        const route = router.parseRoute();
        router.renderView(route);
    };

    // Handle hash change (browser back/forward)
    window.addEventListener('hashchange', handleRoute);

    // Initial route
    handleRoute();

    // Only fetch initial data if on home page
    if (currentView === 'home') {
        fetchBlocks();
    }

    // Initialize search
    initializeSearch();

    // Connect WebSocket
    connectWebSocket();

    // Start timestamp updater
    startTimestampUpdater();

    // Setup pagination event listeners
    document.getElementById('blocks-prev').addEventListener('click', () => {
        if (blocksPage > 1) {
            fetchBlocks(blocksPage - 1);
        }
    });

    document.getElementById('blocks-next').addEventListener('click', () => {
        const totalPages = Math.ceil(blocksTotal / blocksLimit);
        if (blocksPage < totalPages) {
            fetchBlocks(blocksPage + 1);
        }
    });

    document.getElementById('transactions-prev').addEventListener('click', () => {
        if (transactionsPage > 1) {
            transactionsPage--;
            updateTransactionsPagination();
        }
    });

    document.getElementById('transactions-next').addEventListener('click', () => {
        const totalPages = Math.ceil(transactionsTotal / transactionsLimit);
        if (transactionsPage < totalPages) {
            transactionsPage++;
            updateTransactionsPagination();
        }
    });
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    console.log('Page unloading, cleaning up...');

    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
    }

    if (timestampInterval) {
        clearInterval(timestampInterval);
    }

    if (ws) {
        ws.close();
    }
});
