package index

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBlockStore implements BlockStore interface for testing
type MockBlockStore struct {
	latestBlock     *Block
	insertedBlocks  []*Block
	insertError     error
	getLatestError  error
	getByHeightFunc func(uint64) *Block
}

func (m *MockBlockStore) GetLatestBlock(ctx context.Context) (*Block, error) {
	if m.getLatestError != nil {
		return nil, m.getLatestError
	}
	return m.latestBlock, nil
}

func (m *MockBlockStore) InsertBlock(ctx context.Context, block *Block) error {
	if m.insertError != nil {
		return m.insertError
	}
	m.insertedBlocks = append(m.insertedBlocks, block)
	m.latestBlock = block
	return nil
}

func (m *MockBlockStore) GetBlockByHeight(ctx context.Context, height uint64) (*Block, error) {
	if m.getByHeightFunc != nil {
		return m.getByHeightFunc(height), nil
	}
	return nil, errors.New("not found")
}

// MockReorgHandler implements ReorgHandler interface for testing
type MockReorgHandler struct {
	handleReorgCalled bool
	lastBlock        *Block
	handleError      error
}

func (m *MockReorgHandler) HandleReorg(ctx context.Context, block *Block) error {
	m.handleReorgCalled = true
	m.lastBlock = block
	return m.handleError
}

// MockRPCBlockFetcher implements RPCBlockFetcher for testing
type MockRPCBlockFetcher struct {
	blockCache map[uint64]*types.Block
	fetchError func() error
}

func (m *MockRPCBlockFetcher) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
	if m.fetchError != nil {
		return nil, m.fetchError()
	}
	if m.blockCache != nil {
		if block, ok := m.blockCache[height]; ok {
			return block, nil
		}
	}
	return nil, errors.New("not found")
}

// Test AC1: Sequential Block Processing
func TestLiveTailCoordinator_NewCoordinator(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	mockStore := &MockBlockStore{latestBlock: &Block{Height: 100}}
	config := DefaultConfig()

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)
	assert.NotNil(t, coordinator)
	assert.Equal(t, config.PollInterval, coordinator.config.PollInterval)
}

func TestLiveTailCoordinator_NewCoordinator_NilRPC(t *testing.T) {
	mockStore := &MockBlockStore{}
	config := DefaultConfig()

	_, err := NewLiveTailCoordinator(nil, mockStore, nil, nil, nil, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rpcClient cannot be nil")
}

func TestLiveTailCoordinator_NewCoordinator_NilStore(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{}
	config := DefaultConfig()

	_, err := NewLiveTailCoordinator(mockRPC, nil, nil, nil, nil, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store cannot be nil")
}

// Test AC1: Sequential processing - fetch next height correctly
func TestLiveTailCoordinator_SequentialProcessing(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}

	// Set up initial DB head with known hash
	initialHash := []byte("hash_100________________________")
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: initialHash, ParentHash: make([]byte, 32)},
	}
	config := &LiveTailConfig{PollInterval: 1 * time.Millisecond}

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)

	// Override parseRPCBlock to return consistent block chain
	nextHash := []byte("hash_101________________________")
	mockRPC.blockCache[101] = generateTestRPCBlock(101)
	coordinator.parseRPCBlock = func(rpcBlock *types.Block, height uint64) *Block {
		if height == 101 {
			return &Block{Height: 101, Hash: nextHash, ParentHash: initialHash}
		}
		hash102 := []byte("hash_102________________________")
		return &Block{Height: 102, Hash: hash102, ParentHash: nextHash}
	}
	mockRPC.blockCache[102] = generateTestRPCBlock(102)

	// Process one block manually
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Simulate one iteration
	err = coordinator.processNextBlock(ctx)
	require.NoError(t, err)

	// Verify block was inserted
	assert.Len(t, mockStore.insertedBlocks, 1)
	assert.Equal(t, uint64(101), mockStore.insertedBlocks[0].Height)

	// Process next block
	err = coordinator.processNextBlock(ctx)
	require.NoError(t, err)

	assert.Len(t, mockStore.insertedBlocks, 2)
	assert.Equal(t, uint64(102), mockStore.insertedBlocks[1].Height)
}

// Test AC2: Polling and Timing - ticker cadence
func TestLiveTailCoordinator_PollingCadence(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	initialHash := make([]byte, 32)
	initialHash[0] = 100
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: initialHash, ParentHash: make([]byte, 32)},
	}

	// Fast ticker for testing
	config := &LiveTailConfig{PollInterval: 10 * time.Millisecond}

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)

	// Setup block chain with proper hashes
	setupBlockChainForTest(mockRPC, mockStore, coordinator, 101, 110)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	go coordinator.Start(ctx)

	// Wait for context to timeout
	<-ctx.Done()
	duration := time.Since(start)

	// Should have processed multiple blocks in ~100ms with 10ms ticker
	stats := coordinator.Stats()
	blocksProcessed := stats["blocks_processed"].(int64)

	// Expect roughly 100ms / 10ms = 10 iterations, but with overhead should be less
	assert.Greater(t, blocksProcessed, int64(3), "should process at least 3 blocks in 100ms with 10ms ticker")
	assert.Less(t, duration, 200*time.Millisecond, "should complete within reasonable time")
}

// Test AC2: "Block not found" handling
func TestLiveTailCoordinator_BlockNotFound(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: make([]byte, 32), ParentHash: make([]byte, 32)},
	}
	config := DefaultConfig()

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)

	// Mock RPC to return "not found" for next block (no block in cache)
	ctx := context.Background()
	err = coordinator.processNextBlock(ctx)

	// Should return nil (not an error)
	assert.NoError(t, err)

	// Should not have inserted any blocks
	assert.Len(t, mockStore.insertedBlocks, 0)
}

// Test AC3: Error Handling - continue despite transient error
func TestLiveTailCoordinator_ErrorResilience(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	initialHash := make([]byte, 32)
	initialHash[0] = 100
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: initialHash, ParentHash: make([]byte, 32)},
	}
	config := DefaultConfig()

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)

	// Setup block chain
	setupBlockChainForTest(mockRPC, mockStore, coordinator, 101, 101)

	ctx := context.Background()

	// First call: simulate RPC error
	mockRPC.fetchError = func() error {
		return errors.New("temporary network error")
	}

	err = coordinator.processNextBlock(ctx)
	assert.Error(t, err)
	assert.Len(t, mockStore.insertedBlocks, 0, "should not insert block on error")

	// Second call: RPC succeeds
	mockRPC.fetchError = nil
	err = coordinator.processNextBlock(ctx)
	assert.NoError(t, err, "should succeed after error cleared")
	assert.Len(t, mockStore.insertedBlocks, 1, "should insert block on retry")
}

// Test AC3: Context cancellation
func TestLiveTailCoordinator_ContextCancellation(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: make([]byte, 32), ParentHash: make([]byte, 32)},
	}

	// Slow ticker
	config := &LiveTailConfig{PollInterval: 1 * time.Second}

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Start coordinator in goroutine
	done := make(chan error, 1)
	go func() {
		done <- coordinator.Start(ctx)
	}()

	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Should receive context cancelled error
	err = <-done
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// Test AC4: Reorg detection - parent hash mismatch
func TestLiveTailCoordinator_ReorgDetection(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}

	// Latest block with hash A
	latestBlock := &Block{
		Height:     100,
		Hash:       []byte("hash_A_________________________"),
		ParentHash: []byte("parent_hash_A___________________"),
	}
	mockStore := &MockBlockStore{latestBlock: latestBlock}
	mockReorgHandler := &MockReorgHandler{}

	config := DefaultConfig()

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, mockReorgHandler, nil, config)
	require.NoError(t, err)

	// Create block with different parent hash (hash B instead of A)
	differentParentHash := []byte("different_parent_hash___________")
	mockRPC.blockCache[101] = generateTestRPCBlock(101)

	// Override parseRPCBlock to return mismatched parent
	ctx := context.Background()

	// Manually set up the reorg scenario
	coordinator.parseRPCBlock = func(rpcBlock *types.Block, height uint64) *Block {
		return &Block{
			Height:     height,
			Hash:       []byte("hash_B__________________________"),
			ParentHash: differentParentHash, // Mismatch!
		}
	}

	err = coordinator.processNextBlock(ctx)
	require.NoError(t, err)

	// Verify reorg handler was called
	assert.True(t, mockReorgHandler.handleReorgCalled)
	assert.NotNil(t, mockReorgHandler.lastBlock)

	// Block should not have been inserted (skipped due to reorg)
	assert.Len(t, mockStore.insertedBlocks, 0)
}

// Test AC5: Configuration
func TestLiveTailConfig_NewConfig_FromEnv(t *testing.T) {
	t.Setenv("LIVETAIL_POLL_INTERVAL", "5s")

	config, err := NewLiveTailConfig()
	require.NoError(t, err)

	assert.Equal(t, 5*time.Second, config.PollInterval)
}

func TestLiveTailConfig_NewConfig_Defaults(t *testing.T) {
	t.Setenv("LIVETAIL_POLL_INTERVAL", "")

	config, err := NewLiveTailConfig()
	require.NoError(t, err)

	assert.Equal(t, 2*time.Second, config.PollInterval)
}

func TestLiveTailConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *LiveTailConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  &LiveTailConfig{PollInterval: 2 * time.Second},
			wantErr: false,
		},
		{
			name:    "invalid - zero interval",
			config:  &LiveTailConfig{PollInterval: 0},
			wantErr: true,
		},
		{
			name:    "invalid - negative interval",
			config:  &LiveTailConfig{PollInterval: -1 * time.Second},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test AC5: Metrics collection
func TestLiveTailCoordinator_Stats(t *testing.T) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	initialHash := make([]byte, 32)
	initialHash[0] = 100
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: initialHash, ParentHash: make([]byte, 32)},
	}
	config := DefaultConfig()

	coordinator, err := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)
	require.NoError(t, err)

	// Setup block chain with proper hashes
	setupBlockChainForTest(mockRPC, mockStore, coordinator, 101, 103)

	// Add a few blocks
	for h := uint64(101); h <= uint64(103); h++ {
		coordinator.processNextBlock(context.Background())
	}

	stats := coordinator.Stats()

	assert.Equal(t, int64(3), stats["blocks_processed"])
	assert.Equal(t, int64(103), stats["current_height"])
	assert.Equal(t, coordinator.config.PollInterval, stats["poll_interval"])
}

// Helper: generate test block
func generateTestRPCBlock(height uint64) *types.Block {
	header := &types.Header{
		Number: big.NewInt(int64(height)),
		Time:   uint64(time.Now().Unix()),
	}
	// Use types.NewBlock to properly construct a block with the header
	return types.NewBlock(header, nil, nil, nil)
}

// Helper: setup block chain with proper parent hashes
func setupBlockChainForTest(mockRPC *MockRPCBlockFetcher, mockStore *MockBlockStore, coordinator *LiveTailCoordinator, startHeight, endHeight uint64) {
	// Get the current head from store
	currentHash := mockStore.latestBlock.Hash

	// Create a parse override that returns consistent blocks
	coordinator.parseRPCBlock = func(rpcBlock *types.Block, height uint64) *Block {
		hash := make([]byte, 32)
		// Simple hash generation based on height
		hash[0] = byte((height >> 24) & 0xff)
		hash[1] = byte((height >> 16) & 0xff)
		hash[2] = byte((height >> 8) & 0xff)
		hash[3] = byte(height & 0xff)

		parentHash := make([]byte, 32)
		if height == startHeight {
			// First block: parent is the current DB head
			copy(parentHash, currentHash)
		} else {
			// Subsequent blocks: parent is the previous block's hash
			prevHeight := height - 1
			parentHash[0] = byte((prevHeight >> 24) & 0xff)
			parentHash[1] = byte((prevHeight >> 16) & 0xff)
			parentHash[2] = byte((prevHeight >> 8) & 0xff)
			parentHash[3] = byte(prevHeight & 0xff)
		}

		return &Block{Height: height, Hash: hash, ParentHash: parentHash}
	}

	// Add blocks to RPC cache
	for h := startHeight; h <= endHeight; h++ {
		mockRPC.blockCache[h] = generateTestRPCBlock(h)
	}
}

// Benchmark
func BenchmarkLiveTail_ProcessBlock(b *testing.B) {
	mockRPC := &MockRPCBlockFetcher{blockCache: make(map[uint64]*types.Block)}
	mockStore := &MockBlockStore{
		latestBlock: &Block{Height: 100, Hash: make([]byte, 32), ParentHash: make([]byte, 32)},
	}
	config := DefaultConfig()

	coordinator, _ := NewLiveTailCoordinator(mockRPC, mockStore, nil, nil, nil, config)

	// Pre-populate blocks
	for h := uint64(101); h <= uint64(10000); h++ {
		mockRPC.blockCache[h] = generateTestRPCBlock(h)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		coordinator.processNextBlock(context.Background())
	}
}
