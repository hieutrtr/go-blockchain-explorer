package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = 30 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 1048576 // 1MB
)

// Client represents a WebSocket client connection
type Client struct {
	id string

	// WebSocket connection
	conn *websocket.Conn

	// Hub that owns this client
	hub *Hub

	// Buffered channel of outbound messages
	send chan BroadcastMessage

	// Subscribed channels
	subscriptions map[string]bool
	subMu         sync.RWMutex
}

// ControlMessage represents a control message from client
type ControlMessage struct {
	Action   string   `json:"action"`   // "subscribe" or "unsubscribe"
	Channels []string `json:"channels"` // ["newBlocks", "newTxs"]
}

// NewClient creates a new WebSocket client
func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		id:            id,
		conn:          conn,
		hub:           hub,
		send:          make(chan BroadcastMessage, 256),
		subscriptions: make(map[string]bool),
	}
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				util.Error("WebSocket read error", "client_id", c.id, "error", err)
				IncrementErrorMetrics("read_error")
			}
			break
		}

		// Parse control message
		var controlMsg ControlMessage
		if err := json.Unmarshal(message, &controlMsg); err != nil {
			util.Warn("Invalid JSON in control message",
				"client_id", c.id,
				"error", err,
			)
			IncrementErrorMetrics("invalid_json")
			continue
		}

		// Handle control message
		c.handleControlMessage(controlMsg)
	}
}

// writePump writes messages from the send channel to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write JSON message
			if err := c.conn.WriteJSON(message.Data); err != nil {
				util.Error("WebSocket write error", "client_id", c.id, "error", err)
				IncrementErrorMetrics("write_error")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleControlMessage processes subscribe/unsubscribe messages
func (c *Client) handleControlMessage(msg ControlMessage) {
	switch msg.Action {
	case "subscribe":
		c.subscribe(msg.Channels)
	case "unsubscribe":
		c.unsubscribe(msg.Channels)
	default:
		util.Warn("Unknown control action",
			"client_id", c.id,
			"action", msg.Action,
		)
		IncrementErrorMetrics("unknown_action")
	}
}

// subscribe adds channels to client subscriptions
func (c *Client) subscribe(channels []string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, channel := range channels {
		if isValidChannel(channel) {
			c.subscriptions[channel] = true
			util.Debug("Client subscribed to channel",
				"client_id", c.id,
				"channel", channel,
			)
		} else {
			util.Warn("Invalid channel name",
				"client_id", c.id,
				"channel", channel,
			)
			IncrementErrorMetrics("invalid_channel")
		}
	}
}

// unsubscribe removes channels from client subscriptions
func (c *Client) unsubscribe(channels []string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, channel := range channels {
		delete(c.subscriptions, channel)
		util.Debug("Client unsubscribed from channel",
			"client_id", c.id,
			"channel", channel,
		)
	}
}

// isSubscribed checks if client is subscribed to a channel
func (c *Client) isSubscribed(channel string) bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	return c.subscriptions[channel]
}

// isValidChannel validates channel name
func isValidChannel(channel string) bool {
	validChannels := map[string]bool{
		"newBlocks": true,
		"newTxs":    true,
	}
	return validChannels[channel]
}
