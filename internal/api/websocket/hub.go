package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// Hub manages all WebSocket client connections and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast channel for messages
	broadcast chan BroadcastMessage

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Configuration
	config *Config

	// Metrics
	stats *HubStats
}

// BroadcastMessage represents a message to broadcast to clients
type BroadcastMessage struct {
	Channel string
	Data    interface{}
}

// HubStats tracks hub metrics
type HubStats struct {
	TotalConnections    uint64
	ActiveConnections   int
	MessagesSent        uint64
	MessagesDropped     uint64
	BroadcastLatencyMs  int64
}

// NewHub creates a new WebSocket hub
func NewHub(config *Config) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan BroadcastMessage, 256),
		config:     config,
		stats:      &HubStats{},
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	util.Info("WebSocket hub starting")

	for {
		select {
		case <-ctx.Done():
			util.Info("WebSocket hub shutting down")
			h.closeAllClients()
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	h.stats.TotalConnections++
	h.stats.ActiveConnections = len(h.clients)
	h.mu.Unlock()

	remoteAddr := "unknown"
	if client.conn != nil {
		remoteAddr = client.conn.RemoteAddr().String()
	}

	util.Info("WebSocket client registered",
		"client_id", client.id,
		"remote_addr", remoteAddr,
		"active_connections", h.stats.ActiveConnections,
	)

	// Update metrics
	UpdateConnectionMetrics(h.stats.ActiveConnections)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		h.stats.ActiveConnections = len(h.clients)
	}
	h.mu.Unlock()

	util.Info("WebSocket client unregistered",
		"client_id", client.id,
		"active_connections", h.stats.ActiveConnections,
	)

	// Update metrics
	UpdateConnectionMetrics(h.stats.ActiveConnections)
}

// broadcastMessage sends a message to all subscribed clients
func (h *Hub) broadcastMessage(message BroadcastMessage) {
	start := time.Now()

	h.mu.RLock()
	clientCount := len(h.clients)
	h.mu.RUnlock()

	if clientCount == 0 {
		return
	}

	sent := 0
	dropped := 0

	h.mu.RLock()
	for client := range h.clients {
		// Check if client is subscribed to this channel
		if !client.isSubscribed(message.Channel) {
			continue
		}

		// Non-blocking send with select/default
		select {
		case client.send <- message:
			sent++
		default:
			// Channel full - drop message or disconnect client
			dropped++
			util.Warn("WebSocket client send buffer full, dropping message",
				"client_id", client.id,
				"channel", message.Channel,
			)
			IncrementErrorMetrics("buffer_full")
		}
	}
	h.mu.RUnlock()

	duration := time.Since(start).Milliseconds()
	h.stats.MessagesSent += uint64(sent)
	h.stats.MessagesDropped += uint64(dropped)
	h.stats.BroadcastLatencyMs = duration

	if duration > 200 {
		util.Warn("Slow broadcast detected",
			"duration_ms", duration,
			"clients", clientCount,
			"sent", sent,
			"dropped", dropped,
		)
	}

	util.Debug("Broadcast complete",
		"channel", message.Channel,
		"clients", clientCount,
		"sent", sent,
		"dropped", dropped,
		"duration_ms", duration,
	)

	// Update metrics
	IncrementMessageMetrics(message.Channel, sent)
}

// BroadcastBlock broadcasts a new block to all newBlocks subscribers
func (h *Hub) BroadcastBlock(block BlockData) {
	message := BroadcastMessage{
		Channel: "newBlocks",
		Data: map[string]interface{}{
			"type": "newBlock",
			"data": block,
		},
	}

	select {
	case h.broadcast <- message:
	default:
		util.Warn("Broadcast channel full, dropping block message", "height", block.Height)
		IncrementErrorMetrics("broadcast_buffer_full")
	}
}

// BroadcastTransaction broadcasts a new transaction to all newTxs subscribers
func (h *Hub) BroadcastTransaction(tx TransactionData) {
	message := BroadcastMessage{
		Channel: "newTxs",
		Data: map[string]interface{}{
			"type": "newTx",
			"data": tx,
		},
	}

	select {
	case h.broadcast <- message:
	default:
		util.Warn("Broadcast channel full, dropping transaction message", "hash", tx.Hash)
		IncrementErrorMetrics("broadcast_buffer_full")
	}
}

// closeAllClients closes all active client connections
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		if client.conn != nil {
			client.conn.Close()
		}
	}
	h.clients = make(map[*Client]bool)
	h.stats.ActiveConnections = 0

	util.Info("All WebSocket clients closed")
}

// Stats returns current hub statistics
func (h *Hub) Stats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return HubStats{
		TotalConnections:    h.stats.TotalConnections,
		ActiveConnections:   h.stats.ActiveConnections,
		MessagesSent:        h.stats.MessagesSent,
		MessagesDropped:     h.stats.MessagesDropped,
		BroadcastLatencyMs:  h.stats.BroadcastLatencyMs,
	}
}

// BlockData represents block data for broadcasting
type BlockData struct {
	Height    uint64 `json:"height"`
	Hash      string `json:"hash"`
	TxCount   int    `json:"tx_count"`
	Timestamp int64  `json:"timestamp"`
	Miner     string `json:"miner"`
	GasUsed   uint64 `json:"gas_used"`
}

// TransactionData represents transaction data for broadcasting
type TransactionData struct {
	Hash        string `json:"hash"`
	FromAddr    string `json:"from_addr"`
	ToAddr      string `json:"to_addr"`
	ValueWei    string `json:"value_wei"`
	BlockHeight uint64 `json:"block_height"`
}
