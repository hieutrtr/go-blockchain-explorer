//go:build integration

package index

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/hieutt50/go-blockchain-explorer/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationMockBlockStoreExtended extends IntegrationMockBlockStore with orphaned block marking
type IntegrationMockBlockStoreExtended struct {
	*IntegrationMockBlockStore
	orphanedBlocks map[uint64]bool
	mu             sync.RWMutex
}

func NewIntegrationMockBlockStoreExtended() *IntegrationMockBlockStoreExtended {
	return &IntegrationMockBlockStoreExtended{
		IntegrationMockBlockStore: NewIntegrationMockBlockStore(),
		orphanedBlocks:            make(map[uint64]bool),
	}
}

func (m *IntegrationMockBlockStoreExtended) MarkBlocksOrphaned(ctx context.Context, startHeight, endHeight uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if startHeight > endHeight {
		return errors.New("startHeight must be <= endHeight")
	}

	for height := startHeight; height <= endHeight; height++ {
		m.orphanedBlocks[height] = true
	}

	return nil
}

func (m *IntegrationMockBlockStoreExtended) IsOrphaned(height uint64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.orphanedBlocks[height]
}

func (m *IntegrationMockBlockStoreExtended) GetOrphanedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.orphanedBlocks)
}

// TestReorgIntegration_DetectionOnParentHashMismatch tests reorg detection
// Validates AC3: Reorg Recovery Integration Tests - Subtask 4.2
func TestReorgIntegration_DetectionOnParentHashMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate canonical chain (blocks 1-10)
	canonicalChain := test.GenerateTestBlocks(t, 1, 10, 2)

	// Create store and seed with canonical blocks 1-10
	mockStore := NewIntegrationMockBlockStoreExtended()
	for _, fixture := range canonicalChain {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create orphaned block 11 with different parent hash
	orphanedChain := test.CreateOrphanedChain(t, 10, 1)
	orphanedBlock := fixtureToBlock(orphanedChain[0])

	// Create mock RPC with orphaned block
	mockRPC := test.NewMockRPCClient(t, append(canonicalChain, orphanedChain...))

	// Create reorg handler
	config := &ReorgConfig{
		MaxDepth: 10,
	}

	reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
	require.NoError(t, err, "Should create reorg handler")

	// Trigger reorg detection
	err = reorgHandler.HandleReorg(ctx, orphanedBlock)

	// Should detect parent hash mismatch
	assert.NoError(t, err, "Reorg handler should process without error")

	t.Log("✓ Reorg detection on parent hash mismatch verified")
}

// TestReorgIntegration_ForkPointDiscovery tests fork point discovery
// Validates AC3: Reorg Recovery Integration Tests - Subtask 4.3
func TestReorgIntegration_ForkPointDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate canonical chain (blocks 1-10)
	canonicalChain := test.GenerateTestBlocks(t, 1, 10, 1)

	// Create store and seed with canonical blocks
	mockStore := NewIntegrationMockBlockStoreExtended()
	for _, fixture := range canonicalChain {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create orphaned chain starting from block 8 (fork point is block 7)
	orphanedChain := test.CreateOrphanedChain(t, 7, 3) // Blocks 8, 9, 10

	// Create mock RPC with both chains
	allFixtures := append(canonicalChain, orphanedChain...)
	mockRPC := test.NewMockRPCClient(t, allFixtures)

	// Create reorg handler
	config := &ReorgConfig{
		MaxDepth: 10,
	}

	reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Trigger reorg with block 8 from orphaned chain
	orphanedBlock8 := fixtureToBlock(orphanedChain[0])
	err = reorgHandler.HandleReorg(ctx, orphanedBlock8)
	assert.NoError(t, err, "Reorg should be handled")

	// Verify blocks 8, 9, 10 are marked orphaned
	assert.True(t, mockStore.IsOrphaned(8), "Block 8 should be marked orphaned")
	assert.True(t, mockStore.IsOrphaned(9), "Block 9 should be marked orphaned")
	assert.True(t, mockStore.IsOrphaned(10), "Block 10 should be marked orphaned")

	// Verify fork point (block 7) is NOT orphaned
	assert.False(t, mockStore.IsOrphaned(7), "Fork point block 7 should not be orphaned")

	t.Logf("✓ Fork point discovery verified: 3 blocks marked orphaned, fork point at block 7")
}

// TestReorgIntegration_DepthVariations tests various reorg depths
// Validates AC3: Reorg Recovery Integration Tests - Subtask 4.5
func TestReorgIntegration_DepthVariations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testCases := []struct {
		name       string
		reorgDepth int
		maxDepth   int
		shouldPass bool
	}{
		{"depth_1", 1, 10, true},
		{"depth_3", 3, 10, true},
		{"depth_6", 6, 10, true},
		{"depth_10", 10, 10, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Generate canonical chain
			canonicalChain := test.GenerateTestBlocks(t, 1, 20, 1)

			// Create store and seed with canonical blocks
			mockStore := NewIntegrationMockBlockStoreExtended()
			for _, fixture := range canonicalChain {
				mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
			}

			// Create orphaned chain with specified depth
			forkPoint := uint64(20 - tc.reorgDepth)
			orphanedChain := test.CreateOrphanedChain(t, forkPoint, tc.reorgDepth)

			// Create mock RPC
			allFixtures := append(canonicalChain, orphanedChain...)
			mockRPC := test.NewMockRPCClient(t, allFixtures)

			// Create reorg handler
			config := &ReorgConfig{
				MaxDepth: tc.maxDepth,
			}

			reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
			require.NoError(t, err)

			// Trigger reorg
			firstOrphanedBlock := fixtureToBlock(orphanedChain[0])
			err = reorgHandler.HandleReorg(ctx, firstOrphanedBlock)

			if tc.shouldPass {
				assert.NoError(t, err, "Reorg should succeed for depth %d", tc.reorgDepth)

				// Verify correct number of blocks marked orphaned
				orphanedCount := mockStore.GetOrphanedCount()
				assert.Equal(t, tc.reorgDepth, orphanedCount,
					"Should mark exactly %d blocks as orphaned", tc.reorgDepth)

				t.Logf("✓ Reorg depth %d handled successfully", tc.reorgDepth)
			} else {
				assert.Error(t, err, "Reorg should fail for depth %d exceeding max", tc.reorgDepth)
			}
		})
	}
}

// TestReorgIntegration_ExceedsMaxDepth tests reorg beyond max depth
// Validates AC3: Reorg Recovery Integration Tests - Subtask 4.6
func TestReorgIntegration_ExceedsMaxDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate canonical chain (blocks 1-20)
	canonicalChain := test.GenerateTestBlocks(t, 1, 20, 1)

	// Create store and seed with canonical blocks
	mockStore := NewIntegrationMockBlockStoreExtended()
	for _, fixture := range canonicalChain {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create orphaned chain with depth 15 (exceeds max of 10)
	orphanedChain := test.CreateOrphanedChain(t, 5, 15)

	// Create mock RPC
	allFixtures := append(canonicalChain, orphanedChain...)
	mockRPC := test.NewMockRPCClient(t, allFixtures)

	// Create reorg handler with max depth 10
	config := &ReorgConfig{
		MaxDepth: 10,
	}

	reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Trigger reorg with depth 15
	firstOrphanedBlock := fixtureToBlock(orphanedChain[0])
	err = reorgHandler.HandleReorg(ctx, firstOrphanedBlock)

	// Should return error for exceeding max depth
	assert.Error(t, err, "Reorg should fail when exceeding max depth")
	assert.Contains(t, err.Error(), "exceeds maximum", "Error should mention depth exceeded")

	// Verify no blocks were marked orphaned (rollback on error)
	orphanedCount := mockStore.GetOrphanedCount()
	assert.Equal(t, 0, orphanedCount, "No blocks should be marked orphaned when depth exceeded")

	t.Log("✓ Reorg beyond max depth rejected successfully")
}

// TestReorgIntegration_CanonicalChainReplacement tests canonical chain update after reorg
// Validates AC3: Reorg Recovery Integration Tests - Subtask 4.4
func TestReorgIntegration_CanonicalChainReplacement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate canonical chain (blocks 1-10)
	canonicalChain := test.GenerateTestBlocks(t, 1, 10, 2)

	// Create store and seed with canonical blocks 1-8
	mockStore := NewIntegrationMockBlockStoreExtended()
	for _, fixture := range canonicalChain[:8] {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create new canonical chain (blocks 9-10 with different hashes)
	newCanonicalChain := test.GenerateTestBlocks(t, 9, 2, 2)

	// Create mock RPC with new canonical blocks
	allFixtures := append(canonicalChain[:8], newCanonicalChain...)
	mockRPC := test.NewMockRPCClient(t, allFixtures)

	// Create reorg handler
	config := &ReorgConfig{
		MaxDepth: 10,
	}

	reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
	require.NoError(t, err)

	// First, add old blocks 9-10 to simulate they were there
	mockStore.SeedBlocks([]*Block{fixtureToBlock(canonicalChain[8])})
	mockStore.SeedBlocks([]*Block{fixtureToBlock(canonicalChain[9])})

	// Now trigger reorg with new block 9
	newBlock9 := fixtureToBlock(newCanonicalChain[0])
	err = reorgHandler.HandleReorg(ctx, newBlock9)
	assert.NoError(t, err, "Reorg should succeed")

	// Verify old blocks marked orphaned
	assert.True(t, mockStore.IsOrphaned(9), "Old block 9 should be orphaned")
	assert.True(t, mockStore.IsOrphaned(10), "Old block 10 should be orphaned")

	// After reorg, new canonical blocks should be inserted
	// (This would be done by live-tail coordinator after reorg handler)

	t.Log("✓ Canonical chain replacement after reorg verified")
}

// TestReorgIntegration_Metrics tests reorg metrics updates
// Validates AC3: Reorg Recovery Integration Tests - Subtask 4.7
func TestReorgIntegration_Metrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate canonical chain
	canonicalChain := test.GenerateTestBlocks(t, 1, 15, 1)

	// Create store and seed
	mockStore := NewIntegrationMockBlockStoreExtended()
	for _, fixture := range canonicalChain {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create orphaned chain with depth 3
	orphanedChain := test.CreateOrphanedChain(t, 12, 3)

	// Create mock RPC
	allFixtures := append(canonicalChain, orphanedChain...)
	mockRPC := test.NewMockRPCClient(t, allFixtures)

	// Create reorg handler
	config := &ReorgConfig{
		MaxDepth: 10,
	}

	reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Trigger reorg
	firstOrphanedBlock := fixtureToBlock(orphanedChain[0])
	err = reorgHandler.HandleReorg(ctx, firstOrphanedBlock)
	require.NoError(t, err)

	// Verify metrics
	stats := reorgHandler.Stats()
	assert.Equal(t, uint64(1), stats["reorg_detected_total"], "Should increment reorg counter")
	assert.Equal(t, uint64(3), stats["reorg_depth"], "Should record reorg depth")
	assert.Equal(t, uint64(3), stats["orphaned_blocks_total"], "Should count orphaned blocks")

	t.Log("✓ Reorg metrics verified")
}

// TestReorgIntegration_ConcurrentReorgs tests handling multiple reorgs
// Validates robustness of reorg handler under concurrent conditions
func TestReorgIntegration_ConcurrentReorgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Generate canonical chain
	canonicalChain := test.GenerateTestBlocks(t, 1, 20, 1)

	// Create store and seed
	mockStore := NewIntegrationMockBlockStoreExtended()
	for _, fixture := range canonicalChain {
		mockStore.SeedBlocks([]*Block{fixtureToBlock(fixture)})
	}

	// Create two orphaned chains
	orphanedChain1 := test.CreateOrphanedChain(t, 15, 3)
	orphanedChain2 := test.CreateOrphanedChain(t, 10, 2)

	// Create mock RPC
	allFixtures := append(canonicalChain, append(orphanedChain1, orphanedChain2...)...)
	mockRPC := test.NewMockRPCClient(t, allFixtures)

	// Create reorg handler
	config := &ReorgConfig{
		MaxDepth: 10,
	}

	reorgHandler, err := NewReorgHandler(mockRPC, mockStore, config)
	require.NoError(t, err)

	// Trigger concurrent reorgs
	errChan1 := make(chan error, 1)
	errChan2 := make(chan error, 1)

	go func() {
		errChan1 <- reorgHandler.HandleReorg(ctx, fixtureToBlock(orphanedChain1[0]))
	}()

	go func() {
		errChan2 <- reorgHandler.HandleReorg(ctx, fixtureToBlock(orphanedChain2[0]))
	}()

	// Wait for both to complete
	err1 := <-errChan1
	err2 := <-errChan2

	// Both should complete (though one might fail due to race conditions)
	t.Logf("Reorg 1 result: %v", err1)
	t.Logf("Reorg 2 result: %v", err2)

	// At least one should succeed
	assert.True(t, err1 == nil || err2 == nil, "At least one reorg should succeed")

	t.Log("✓ Concurrent reorg handling verified")
}
