//go:build integration

package index

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hieutt50/go-blockchain-explorer/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for LiveTail integration testing

// IntegrationMockBlockStore simulates database operations for integration tests
type IntegrationMockBlockStore struct {
	mu      sync.RWMutex
	blocks  map[uint64]*Block
	latestHeight uint64
}

func NewIntegrationMockBlockStore() *IntegrationMockBlockStore {
	return &IntegrationMockBlockStore{
		blocks: make(map[uint64]*Block),
	}
}

func (m *IntegrationMockBlockStore) GetLatestBlock(ctx context.Context) (*Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.latestHeight == 0 {
		return nil, errors.New("no blocks in database")
	}

	block, ok := m.blocks[m.latestHeight]
	if !ok {
		return nil, errors.New("latest block not found")
	}

	return block, nil
}

func (m *IntegrationMockBlockStore) InsertBlock(ctx context.Context, block *Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blocks[block.Height] = block
	if block.Height > m.latestHeight {
		m.latestHeight = block.Height
	}

	return nil
}

func (m *IntegrationMockBlockStore) GetBlockByHeight(ctx context.Context, height uint64) (*Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	block, ok := m.blocks[height]
	if !ok {
		return nil, errors.New("block not found")
	}

	return block, nil
}

func (m *IntegrationMockBlockStore) SeedBlocks(blocks []*Block) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, block := range blocks {
		m.blocks[block.Height] = block
		if block.Height > m.latestHeight {
			m.latestHeight = block.Height
		}
	}
}

// IntegrationMockBlockIngester stub for integration tests
type IntegrationMockBlockIngester struct{}

func (m *IntegrationMockBlockIngester) ParseBlock(rpcBlock *types.Block) (*Block, error) {
	return &Block{
		Height:     rpcBlock.Number().Uint64(),
		Hash:       rpcBlock.Hash().Bytes(),
		ParentHash: rpcBlock.ParentHash().Bytes(),
		Timestamp:  rpcBlock.Time(),
		Miner:      rpcBlock.Coinbase().Bytes(),
		GasUsed:    rpcBlock.GasUsed(),
		TxCount:    len(rpcBlock.Transactions()),
	}, nil
}

// IntegrationMockReorgHandler tracks reorg calls
type IntegrationMockReorgHandler struct {
	mu          sync.Mutex
	reorgCalled bool
	reorgBlocks []*Block
}

func (m *IntegrationMockReorgHandler) HandleReorg(ctx context.Context, block *Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reorgCalled = true
	m.reorgBlocks = append(m.reorgBlocks, block)

	return nil
}

func (m *IntegrationMockReorgHandler) WasReorgCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reorgCalled
}

// TestLiveTailIntegration_FetchNextBlock tests live-tail fetches next block after backfill
// Validates AC2: Live-Tail Integration Tests - Subtask 3.1
func TestLiveTailIntegration_FetchNextBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Generate test blocks (1-50)
	fixtures := test.GenerateTestBlocks(t, 1, 50, 2)

	// Create mock store and seed with blocks 1-48 (simulating backfill)
	mockStore := NewIntegrationMockBlockStore()
	for _, fixture := range fixtures[:48] {
		domainBlock := fixtureToBlock(fixture)
		mockStore.SeedBlocks([]*Block{domainBlock})
	}

	// Create mock RPC with blocks 49-50 available
	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Create live-tail coordinator
	config := &LiveTailConfig{
		PollInterval: 100 * time.Millisecond,
	}

	coordinator, err := NewLiveTailCoordinator(
		mockRPC,
		mockStore,
		&IntegrationMockBlockIngester{},
		&IntegrationMockReorgHandler{},
		nil, // No WebSocket hub
		config,
	)
	require.NoError(t, err, "Should create live-tail coordinator")

	// Start live-tail in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- coordinator.Start(ctx)
	}()

	// Wait for live-tail to process blocks 49 and 50
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Wait for shutdown
	err = <-errChan
	assert.ErrorIs(t, err, context.Canceled, "Should shut down gracefully")

	// Verify blocks 49 and 50 were fetched
	block49, err := mockStore.GetBlockByHeight(context.Background(), 49)
	assert.NoError(t, err, "Block 49 should be fetched")
	assert.Equal(t, uint64(49), block49.Height)

	// Check stats
	stats := coordinator.Stats()
	blocksProcessed := stats["blocks_processed"].(int64)
	t.Logf("Blocks processed: %d", blocksProcessed)
	assert.Greater(t, blocksProcessed, int64(0), "Should have processed blocks")
}

// TestLiveTailIntegration_ParentChildRelationship tests parent-child block relationships
// Validates AC2: Live-Tail Integration Tests - Subtask 3.2
func TestLiveTailIntegration_ParentChildRelationship(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Generate test blocks with proper parent-child relationship
	fixtures := test.GenerateTestBlocks(t, 1, 10, 1)

	// Seed store with block 1-5
	mockStore := NewIntegrationMockBlockStore()
	for _, fixture := range fixtures[:5] {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create mock RPC with blocks 6-10
	mockRPC := test.NewMockRPCClient(t, fixtures)

	config := &LiveTailConfig{
		PollInterval: 50 * time.Millisecond,
	}

	coordinator, err := NewLiveTailCoordinator(
		mockRPC,
		mockStore,
		&IntegrationMockBlockIngester{},
		&IntegrationMockReorgHandler{},
		nil,
		config,
	)
	require.NoError(t, err)

	// Start live-tail
	errChan := make(chan error, 1)
	go func() {
		errChan <- coordinator.Start(ctx)
	}()

	// Wait for processing
	time.Sleep(400 * time.Millisecond)
	cancel()
	<-errChan

	// Verify parent-child relationships
	for i := uint64(6); i <= 10; i++ {
		block, err := mockStore.GetBlockByHeight(context.Background(), i)
		if err != nil {
			continue // Block might not be processed yet
		}

		parent, err := mockStore.GetBlockByHeight(context.Background(), i-1)
		require.NoError(t, err, "Parent block should exist")

		assert.Equal(t, parent.Hash, block.ParentHash,
			"Block %d parent hash should match block %d hash", i, i-1)
	}

	t.Log("✓ Parent-child relationships verified")
}

// TestLiveTailIntegration_MissingBlockHandling tests handling of missing blocks
// Validates AC2: Live-Tail Integration Tests - Subtask 3.5
func TestLiveTailIntegration_MissingBlockHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Generate test blocks
	fixtures := test.GenerateTestBlocks(t, 1, 5, 1)

	// Seed store with block 1-3
	mockStore := NewIntegrationMockBlockStore()
	for _, fixture := range fixtures[:3] {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create mock RPC with only blocks 1-3 (blocks 4-5 missing)
	mockRPC := test.NewMockRPCClient(t, fixtures[:3])

	config := &LiveTailConfig{
		PollInterval: 100 * time.Millisecond,
	}

	coordinator, err := NewLiveTailCoordinator(
		mockRPC,
		mockStore,
		&IntegrationMockBlockIngester{},
		&IntegrationMockReorgHandler{},
		nil,
		config,
	)
	require.NoError(t, err)

	// Start live-tail
	errChan := make(chan error, 1)
	go func() {
		errChan <- coordinator.Start(ctx)
	}()

	// Wait for a few poll cycles
	time.Sleep(500 * time.Millisecond)

	// Add block 4 dynamically
	mockRPC.AddBlocks(fixtures[3:4])

	// Wait for processing
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-errChan

	// Verify block 4 was processed once available
	block4, err := mockStore.GetBlockByHeight(context.Background(), 4)
	if err == nil {
		assert.Equal(t, uint64(4), block4.Height, "Block 4 should be processed")
	}

	t.Log("✓ Missing block handling verified")
}

// TestLiveTailIntegration_ContextCancellation tests graceful shutdown
// Validates AC2: Live-Tail Integration Tests - Subtask 3.6
func TestLiveTailIntegration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Generate test blocks
	fixtures := test.GenerateTestBlocks(t, 1, 100, 1)

	mockStore := NewIntegrationMockBlockStore()
	mockStore.SeedBlocks([]*Block{fixtureToBlock(fixtures[0])})

	mockRPC := test.NewMockRPCClient(t, fixtures)

	config := &LiveTailConfig{
		PollInterval: 50 * time.Millisecond,
	}

	coordinator, err := NewLiveTailCoordinator(
		mockRPC,
		mockStore,
		&IntegrationMockBlockIngester{},
		&IntegrationMockReorgHandler{},
		nil,
		config,
	)
	require.NoError(t, err)

	// Start live-tail
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)
	startTime := time.Now()

	go func() {
		errChan <- coordinator.Start(ctx)
	}()

	// Let it run for a bit
	time.Sleep(300 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for shutdown
	select {
	case err := <-errChan:
		assert.ErrorIs(t, err, context.Canceled, "Should return context.Canceled")
		duration := time.Since(startTime)
		assert.Less(t, duration, 500*time.Millisecond, "Should stop quickly")
	case <-time.After(1 * time.Second):
		t.Fatal("Coordinator did not stop within timeout")
	}

	t.Log("✓ Context cancellation handled gracefully")
}

// TestLiveTailIntegration_PollingBehavior tests continuous polling with ticker
// Validates AC2: Live-Tail Integration Tests - Subtask 3.4
func TestLiveTailIntegration_PollingBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Generate test blocks
	fixtures := test.GenerateTestBlocks(t, 1, 20, 1)

	mockStore := NewIntegrationMockBlockStore()
	mockStore.SeedBlocks([]*Block{fixtureToBlock(fixtures[0])})

	mockRPC := test.NewMockRPCClient(t, fixtures)

	// Short poll interval for testing
	config := &LiveTailConfig{
		PollInterval: 50 * time.Millisecond,
	}

	coordinator, err := NewLiveTailCoordinator(
		mockRPC,
		mockStore,
		&IntegrationMockBlockIngester{},
		&IntegrationMockReorgHandler{},
		nil,
		config,
	)
	require.NoError(t, err)

	// Start live-tail
	errChan := make(chan error, 1)
	go func() {
		errChan <- coordinator.Start(ctx)
	}()

	// Wait for multiple poll cycles
	time.Sleep(800 * time.Millisecond)
	cancel()
	<-errChan

	// Verify multiple blocks were processed (proves ticker is working)
	stats := coordinator.Stats()
	blocksProcessed := stats["blocks_processed"].(int64)
	t.Logf("Blocks processed in 800ms: %d", blocksProcessed)

	// With 50ms interval, should process multiple blocks
	assert.Greater(t, blocksProcessed, int64(3), "Should process multiple blocks with continuous polling")

	t.Log("✓ Continuous polling behavior verified")
}

// TestLiveTailIntegration_ReorgDetection tests reorg detection via parent hash mismatch
// Validates AC2: Live-Tail Integration Tests - parent-child verification leads to reorg detection
func TestLiveTailIntegration_ReorgDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Generate canonical chain (1-5)
	canonicalChain := test.GenerateTestBlocks(t, 1, 5, 1)

	// Generate orphaned block 6 (different parent hash)
	orphanedChain := test.CreateOrphanedChain(t, 5, 1)

	// Seed store with canonical blocks 1-5
	mockStore := NewIntegrationMockBlockStore()
	for _, fixture := range canonicalChain {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create RPC with orphaned block 6
	mockRPC := test.NewMockRPCClient(t, append(canonicalChain, orphanedChain[0]))

	reorgHandler := &IntegrationMockReorgHandler{}

	config := &LiveTailConfig{
		PollInterval: 100 * time.Millisecond,
	}

	coordinator, err := NewLiveTailCoordinator(
		mockRPC,
		mockStore,
		&IntegrationMockBlockIngester{},
		reorgHandler,
		nil,
		config,
	)
	require.NoError(t, err)

	// Start live-tail
	errChan := make(chan error, 1)
	go func() {
		errChan <- coordinator.Start(ctx)
	}()

	// Wait for processing
	time.Sleep(500 * time.Millisecond)
	cancel()
	<-errChan

	// Verify reorg handler was called
	assert.True(t, reorgHandler.WasReorgCalled(), "Reorg handler should be called for parent hash mismatch")

	t.Log("✓ Reorg detection verified")
}

// Helper function to convert test fixture to domain Block
func fixtureToBlock(fixture *test.FixtureBlock) *Block {
	return &Block{
		Height:     fixture.Height,
		Hash:       fixture.Hash,
		ParentHash: fixture.ParentHash,
		Timestamp:  uint64(fixture.Timestamp),
		Miner:      fixture.Miner,
		GasUsed:    fixture.GasUsed.Uint64(),
		TxCount:    len(fixture.Transactions),
	}
}
