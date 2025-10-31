package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	config := &Config{
		MaxConnections: 100,
		PingInterval:   30 * time.Second,
	}

	hub := NewHub(config)

	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.Equal(t, config, hub.config)
}

func TestHub_RegisterUnregister(t *testing.T) {
	config := &Config{MaxConnections: 100}
	hub := NewHub(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start hub
	go hub.Run(ctx)

	// Create mock client
	mockClient := &Client{
		id:            "test-client-1",
		send:          make(chan BroadcastMessage, 256),
		subscriptions: make(map[string]bool),
	}

	// Register client
	hub.register <- mockClient

	// Give hub time to process
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	stats := hub.Stats()
	assert.Equal(t, 1, stats.ActiveConnections)
	assert.Equal(t, uint64(1), stats.TotalConnections)

	// Unregister client
	hub.unregister <- mockClient

	// Give hub time to process
	time.Sleep(10 * time.Millisecond)

	// Verify client is unregistered
	stats = hub.Stats()
	assert.Equal(t, 0, stats.ActiveConnections)
}

func TestHub_BroadcastToSubscribedClients(t *testing.T) {
	config := &Config{MaxConnections: 100}
	hub := NewHub(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start hub
	go hub.Run(ctx)

	// Create two mock clients
	client1 := &Client{
		id:            "client-1",
		send:          make(chan BroadcastMessage, 256),
		subscriptions: map[string]bool{"newBlocks": true},
	}

	client2 := &Client{
		id:            "client-2",
		send:          make(chan BroadcastMessage, 256),
		subscriptions: map[string]bool{"newTxs": true},
	}

	// Register clients
	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	// Broadcast a block
	hub.BroadcastBlock(BlockData{
		Height:    100,
		Hash:      "0x1234",
		TxCount:   5,
		Timestamp: time.Now().Unix(),
	})

	// Give hub time to broadcast
	time.Sleep(20 * time.Millisecond)

	// Client 1 (subscribed to newBlocks) should receive message
	select {
	case msg := <-client1.send:
		assert.Equal(t, "newBlocks", msg.Channel)
		data := msg.Data.(map[string]interface{})
		assert.Equal(t, "newBlock", data["type"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 1 did not receive block message")
	}

	// Client 2 (subscribed to newTxs) should NOT receive message
	select {
	case <-client2.send:
		t.Fatal("Client 2 should not receive block message (not subscribed)")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message received
	}
}

func TestHub_NonBlockingBroadcast(t *testing.T) {
	config := &Config{MaxConnections: 100}
	hub := NewHub(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start hub
	go hub.Run(ctx)

	// Create slow client with small buffer
	slowClient := &Client{
		id:            "slow-client",
		send:          make(chan BroadcastMessage, 2), // Small buffer
		subscriptions: map[string]bool{"newBlocks": true},
	}

	// Create fast client
	fastClient := &Client{
		id:            "fast-client",
		send:          make(chan BroadcastMessage, 256),
		subscriptions: map[string]bool{"newBlocks": true},
	}

	// Register clients
	hub.register <- slowClient
	hub.register <- fastClient
	time.Sleep(10 * time.Millisecond)

	// Fill slow client's buffer by not reading from it
	// Send multiple blocks rapidly
	for i := 0; i < 10; i++ {
		hub.BroadcastBlock(BlockData{
			Height:    uint64(100 + i),
			Hash:      "0x1234",
			TxCount:   5,
			Timestamp: time.Now().Unix(),
		})
		time.Sleep(2 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	// Fast client should still receive messages (non-blocking)
	messagesReceived := 0
	timeout := time.After(100 * time.Millisecond)

readLoop:
	for {
		select {
		case <-fastClient.send:
			messagesReceived++
			if messagesReceived >= 5 {
				break readLoop
			}
		case <-timeout:
			break readLoop
		}
	}

	// Fast client should have received at least some messages
	assert.GreaterOrEqual(t, messagesReceived, 1, "Fast client should receive messages even when slow client blocks")
}

func TestClient_IsSubscribed(t *testing.T) {
	client := &Client{
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	// Initially not subscribed
	assert.False(t, client.isSubscribed("newBlocks"))

	// Subscribe
	client.subscribe([]string{"newBlocks", "newTxs"})

	// Now subscribed
	assert.True(t, client.isSubscribed("newBlocks"))
	assert.True(t, client.isSubscribed("newTxs"))

	// Unsubscribe from one channel
	client.unsubscribe([]string{"newBlocks"})

	// Verify state
	assert.False(t, client.isSubscribed("newBlocks"))
	assert.True(t, client.isSubscribed("newTxs"))
}

func TestIsValidChannel(t *testing.T) {
	tests := []struct {
		channel string
		valid   bool
	}{
		{"newBlocks", true},
		{"newTxs", true},
		{"invalidChannel", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			result := isValidChannel(tt.channel)
			assert.Equal(t, tt.valid, result)
		})
	}
}
