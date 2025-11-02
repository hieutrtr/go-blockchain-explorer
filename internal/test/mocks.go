//go:build integration

package test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// MockRPCClient is a mock Ethereum RPC client for testing
// Provides deterministic responses and failure injection
type MockRPCClient struct {
	t              *testing.T
	mu             sync.RWMutex
	blocks         map[uint64]*types.Block
	failures       map[uint64]int  // Height -> number of times to fail
	permanentError *uint64         // Height that should always fail
	delay          time.Duration   // Simulated network delay
	callCount      int             // Track number of calls
	failuresLeft   int             // Global failure counter
}

// NewMockRPCClient creates a new mock RPC client with preloaded blocks
func NewMockRPCClient(t *testing.T, fixtures []*FixtureBlock) *MockRPCClient {
	t.Helper()

	client := &MockRPCClient{
		t:         t,
		blocks:    make(map[uint64]*types.Block),
		failures:  make(map[uint64]int),
	}

	// Convert fixtures to Ethereum blocks
	for _, fixture := range fixtures {
		ethBlock := GenerateEthereumBlock(t, fixture)
		client.blocks[fixture.Height] = ethBlock
	}

	t.Logf("MockRPCClient initialized with %d blocks", len(client.blocks))

	return client
}

// GetBlockByNumber mocks ethclient.BlockByNumber
// Supports failure injection and delay simulation
func (m *MockRPCClient) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Simulate network delay if configured
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Check for global failures (used for retry testing)
	if m.failuresLeft > 0 {
		m.failuresLeft--
		m.t.Logf("MockRPCClient: Injecting transient failure (remaining: %d)", m.failuresLeft)
		return nil, errors.New("network timeout")
	}

	// Check for permanent error
	if m.permanentError != nil && *m.permanentError == height {
		m.t.Logf("MockRPCClient: Permanent error for height %d", height)
		return nil, errors.New("invalid block height")
	}

	// Check for height-specific failures
	if failCount, ok := m.failures[height]; ok && failCount > 0 {
		m.failures[height]--
		m.t.Logf("MockRPCClient: Injecting failure for height %d (remaining: %d)", height, m.failures[height])
		return nil, errors.New("temporary failure")
	}

	// Return block if exists
	block, ok := m.blocks[height]
	if !ok {
		return nil, fmt.Errorf("block not found: %d", height)
	}

	return block, nil
}

// GetTransactionReceipt mocks ethclient.TransactionReceipt
func (m *MockRPCClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple mock - return success receipt
	receipt := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 21000,
		Logs:              []*types.Log{},
		TxHash:            txHash,
		GasUsed:           21000,
		BlockNumber:       big.NewInt(1),
	}

	return receipt, nil
}

// SetFailures configures the client to fail N times for a specific height
// Useful for testing retry logic
func (m *MockRPCClient) SetFailures(height uint64, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failures[height] = count
	m.t.Logf("MockRPCClient: Set %d failures for height %d", count, height)
}

// SetGlobalFailures configures the client to fail the next N calls globally
// Useful for testing retry logic without specific heights
func (m *MockRPCClient) SetGlobalFailures(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failuresLeft = count
	m.t.Logf("MockRPCClient: Set %d global failures", count)
}

// SetPermanentError configures a height that always fails
// Useful for testing permanent error handling
func (m *MockRPCClient) SetPermanentError(height uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.permanentError = &height
	m.t.Logf("MockRPCClient: Set permanent error for height %d", height)
}

// SetDelay configures simulated network delay
func (m *MockRPCClient) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.delay = delay
	m.t.Logf("MockRPCClient: Set delay to %v", delay)
}

// AddBlock adds a new block to the mock client
func (m *MockRPCClient) AddBlock(fixture *FixtureBlock) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ethBlock := GenerateEthereumBlock(m.t, fixture)
	m.blocks[fixture.Height] = ethBlock
	m.t.Logf("MockRPCClient: Added block %d", fixture.Height)
}

// AddBlocks adds multiple blocks to the mock client
func (m *MockRPCClient) AddBlocks(fixtures []*FixtureBlock) {
	for _, fixture := range fixtures {
		m.AddBlock(fixture)
	}
}

// GetCallCount returns the number of times GetBlockByNumber was called
func (m *MockRPCClient) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.callCount
}

// ResetCallCount resets the call counter
func (m *MockRPCClient) ResetCallCount() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount = 0
}

// HasBlock checks if a block exists in the mock client
func (m *MockRPCClient) HasBlock(height uint64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.blocks[height]
	return ok
}

// GetBlockCount returns the number of blocks in the mock client
func (m *MockRPCClient) GetBlockCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.blocks)
}

// Clear removes all blocks from the mock client
func (m *MockRPCClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blocks = make(map[uint64]*types.Block)
	m.failures = make(map[uint64]int)
	m.permanentError = nil
	m.delay = 0
	m.callCount = 0
	m.failuresLeft = 0

	m.t.Logf("MockRPCClient: Cleared all data")
}

// MockFailingRPCClient always fails (for testing error paths)
type MockFailingRPCClient struct {
	t       *testing.T
	errType string
}

// NewMockFailingRPCClient creates a client that always fails
func NewMockFailingRPCClient(t *testing.T, errType string) *MockFailingRPCClient {
	return &MockFailingRPCClient{
		t:       t,
		errType: errType,
	}
}

// GetBlockByNumber always returns an error
func (m *MockFailingRPCClient) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
	switch m.errType {
	case "network":
		return nil, errors.New("network timeout")
	case "invalid":
		return nil, errors.New("invalid block height")
	case "context":
		return nil, context.Canceled
	default:
		return nil, errors.New("unknown error")
	}
}

// GetTransactionReceipt always returns an error
func (m *MockFailingRPCClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return nil, errors.New("receipt not found")
}

// MockSlowRPCClient simulates slow network responses
type MockSlowRPCClient struct {
	t         *testing.T
	client    *MockRPCClient
	minDelay  time.Duration
	maxDelay  time.Duration
}

// NewMockSlowRPCClient creates a client with random delays
func NewMockSlowRPCClient(t *testing.T, fixtures []*FixtureBlock, minDelay, maxDelay time.Duration) *MockSlowRPCClient {
	return &MockSlowRPCClient{
		t:        t,
		client:   NewMockRPCClient(t, fixtures),
		minDelay: minDelay,
		maxDelay: maxDelay,
	}
}

// GetBlockByNumber returns blocks with random delay
func (m *MockSlowRPCClient) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
	// Random delay between min and max
	delay := m.minDelay + time.Duration(height%10)*((m.maxDelay-m.minDelay)/10)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(delay):
		return m.client.GetBlockByNumber(ctx, height)
	}
}

// GetTransactionReceipt returns receipts with delay
func (m *MockSlowRPCClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	time.Sleep(m.minDelay)
	return m.client.GetTransactionReceipt(ctx, txHash)
}
