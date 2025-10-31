// WebSocket connection management
let ws;
let reconnectTimer;

function connectWebSocket() {
    const wsUrl = `ws://${window.location.host}/v1/stream`;
    console.log('Connecting to WebSocket:', wsUrl);

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('WebSocket connected');

        // Subscribe to channels
        ws.send(JSON.stringify({
            action: 'subscribe',
            channels: ['newBlocks', 'newTxs']
        }));
    };

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log('WebSocket message:', message);

        if (message.type === 'newBlock') {
            handleNewBlock(message.data);
        } else if (message.type === 'newTx') {
            handleNewTransaction(message.data);
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        console.log('WebSocket closed, reconnecting in 3 seconds...');
        reconnectTimer = setTimeout(connectWebSocket, 3000);
    };
}

function handleNewBlock(block) {
    console.log('New block:', block);
    // TODO: Update UI with new block
    displayBlock(block);
}

function handleNewTransaction(tx) {
    console.log('New transaction:', tx);
    // TODO: Update UI with new transaction
}

function displayBlock(block) {
    const blockList = document.getElementById('blockList');
    if (!blockList) return;

    const blockElement = document.createElement('div');
    blockElement.className = 'block-item';
    blockElement.innerHTML = `
        <div class="block-height">Block #${block.height}</div>
        <div class="block-hash">${block.hash.substring(0, 10)}...${block.hash.substring(block.hash.length - 8)}</div>
        <div class="block-time">${new Date(block.timestamp * 1000).toLocaleTimeString()}</div>
    `;

    blockList.insertBefore(blockElement, blockList.firstChild);

    // Keep only last 10 blocks
    while (blockList.children.length > 10) {
        blockList.removeChild(blockList.lastChild);
    }
}

// Connect when page loads
document.addEventListener('DOMContentLoaded', () => {
    connectWebSocket();
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (reconnectTimer) clearTimeout(reconnectTimer);
    if (ws) ws.close();
});
