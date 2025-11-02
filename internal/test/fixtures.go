//go:build integration

package test

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// FixtureBlock generates a deterministic test block for integration tests
type FixtureBlock struct {
	Height       uint64
	Hash         []byte
	ParentHash   []byte
	Miner        []byte
	GasUsed      *big.Int
	GasLimit     *big.Int
	Timestamp    int64
	Transactions []*FixtureTransaction
}

// FixtureTransaction generates a deterministic test transaction
type FixtureTransaction struct {
	Hash       []byte
	FromAddr   []byte
	ToAddr     []byte
	ValueWei   *big.Int
	GasUsed    *big.Int
	GasPrice   *big.Int
	Nonce      uint64
	Success    bool
	Logs       []*FixtureLog
}

// FixtureLog generates a deterministic test log
type FixtureLog struct {
	Address []byte
	Topic0  []byte
	Topic1  []byte
	Topic2  []byte
	Topic3  []byte
	Data    []byte
}

// GenerateTestBlocks generates a chain of N deterministic test blocks
// Each block has a configurable number of transactions and logs
// Returns blocks in ascending order (height 1, 2, 3, ...)
func GenerateTestBlocks(t *testing.T, startHeight uint64, count int, txPerBlock int) []*FixtureBlock {
	t.Helper()

	blocks := make([]*FixtureBlock, count)

	// Genesis parent hash for first block
	var parentHash []byte
	if startHeight == 0 {
		parentHash = make([]byte, 32) // All zeros for genesis
	} else {
		// Use deterministic parent hash for non-genesis
		parentHash = generateDeterministicHash(startHeight - 1)
	}

	for i := 0; i < count; i++ {
		height := startHeight + uint64(i)

		block := &FixtureBlock{
			Height:     height,
			Hash:       generateDeterministicHash(height),
			ParentHash: parentHash,
			Miner:      generateDeterministicAddress(height),
			GasUsed:    big.NewInt(int64(1000000 + height*1000)),
			GasLimit:   big.NewInt(8000000),
			Timestamp:  time.Now().Unix() - int64(count-i)*12, // 12 sec block time
			Transactions: make([]*FixtureTransaction, txPerBlock),
		}

		// Generate transactions for this block
		for j := 0; j < txPerBlock; j++ {
			block.Transactions[j] = generateTestTransaction(t, height, j)
		}

		blocks[i] = block

		// Next block's parent is this block's hash
		parentHash = block.Hash
	}

	return blocks
}

// generateTestTransaction generates a single deterministic test transaction
func generateTestTransaction(t *testing.T, blockHeight uint64, txIndex int) *FixtureTransaction {
	t.Helper()

	// Deterministic transaction hash based on block height and tx index
	txHash := generateDeterministicHash(blockHeight*1000 + uint64(txIndex))

	// Create from/to addresses
	fromAddr := generateDeterministicAddress(blockHeight*1000 + uint64(txIndex))
	toAddr := generateDeterministicAddress(blockHeight*1000 + uint64(txIndex) + 1)

	// Generate 1-3 logs per transaction
	logCount := (txIndex % 3) + 1
	logs := make([]*FixtureLog, logCount)
	for i := 0; i < logCount; i++ {
		logs[i] = generateTestLog(t, blockHeight, txIndex, i)
	}

	return &FixtureTransaction{
		Hash:     txHash,
		FromAddr: fromAddr,
		ToAddr:   toAddr,
		ValueWei: big.NewInt(int64(1000000000000000000 + blockHeight*100000000 + uint64(txIndex)*1000000)), // ~1 ETH + small delta
		GasUsed:  big.NewInt(21000 + int64(txIndex)*1000),
		GasPrice: big.NewInt(20000000000), // 20 gwei
		Nonce:    blockHeight*100 + uint64(txIndex),
		Success:  txIndex%10 != 7, // 90% success rate (fail on index 7, 17, 27, ...)
		Logs:     logs,
	}
}

// generateTestLog generates a single deterministic test log
func generateTestLog(t *testing.T, blockHeight uint64, txIndex int, logIndex int) *FixtureLog {
	t.Helper()

	seed := blockHeight*100000 + uint64(txIndex)*100 + uint64(logIndex)

	return &FixtureLog{
		Address: generateDeterministicAddress(seed),
		Topic0:  generateDeterministicHash(seed + 1),
		Topic1:  generateDeterministicHash(seed + 2),
		Topic2:  generateDeterministicHash(seed + 3),
		Topic3:  nil, // Not all logs have 4 topics
		Data:    generateDeterministicBytes(seed, 128),
	}
}

// generateDeterministicHash generates a 32-byte hash from a seed
func generateDeterministicHash(seed uint64) []byte {
	hash := make([]byte, 32)
	// Use seed to generate deterministic bytes
	for i := 0; i < 32; i++ {
		hash[i] = byte((seed >> (i % 8 * 8)) & 0xFF)
	}
	// Add some variation based on position
	for i := 0; i < 32; i++ {
		hash[i] ^= byte(i * 7)
	}
	return hash
}

// generateDeterministicAddress generates a 20-byte address from a seed
func generateDeterministicAddress(seed uint64) []byte {
	addr := make([]byte, 20)
	for i := 0; i < 20; i++ {
		addr[i] = byte((seed >> (i % 8 * 8)) & 0xFF)
	}
	// Add some variation
	for i := 0; i < 20; i++ {
		addr[i] ^= byte(i * 11)
	}
	return addr
}

// generateDeterministicBytes generates N bytes of deterministic data
func generateDeterministicBytes(seed uint64, length int) []byte {
	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = byte((seed >> (i % 8 * 8)) & 0xFF)
		data[i] ^= byte(i * 13)
	}
	return data
}

// GenerateEthereumBlock converts a FixtureBlock to an ethereum types.Block
// Useful for testing RPC client and ingestion layer
func GenerateEthereumBlock(t *testing.T, fixture *FixtureBlock) *types.Block {
	t.Helper()

	// Convert transactions
	ethTxs := make([]*types.Transaction, len(fixture.Transactions))
	for i, tx := range fixture.Transactions {
		// Create a legacy transaction
		ethTx := types.NewTransaction(
			tx.Nonce,
			common.BytesToAddress(tx.ToAddr),
			tx.ValueWei,
			tx.GasUsed.Uint64(),
			tx.GasPrice,
			nil, // No data for simple transfers
		)
		ethTxs[i] = ethTx
	}

	// Create block header
	// For test fixtures, use a deterministic transaction hash based on block height
	txHash := common.BytesToHash(generateDeterministicHash(fixture.Height * 10000))

	header := &types.Header{
		ParentHash: common.BytesToHash(fixture.ParentHash),
		Coinbase:   common.BytesToAddress(fixture.Miner),
		Root:       common.Hash{}, // State root (not critical for tests)
		TxHash:     txHash,
		Number:     big.NewInt(int64(fixture.Height)),
		GasLimit:   fixture.GasLimit.Uint64(),
		GasUsed:    fixture.GasUsed.Uint64(),
		Time:       uint64(fixture.Timestamp),
	}

	// Create block
	block := types.NewBlockWithHeader(header).WithBody(types.Body{
		Transactions: ethTxs,
	})

	return block
}

// CreateOrphanedChain creates a fork/orphaned chain for reorg testing
// Returns blocks that diverge from mainHeight with different parent hashes
func CreateOrphanedChain(t *testing.T, forkPoint uint64, depth int) []*FixtureBlock {
	t.Helper()

	orphanedBlocks := make([]*FixtureBlock, depth)

	// Start with parent hash from fork point
	parentHash := generateDeterministicHash(forkPoint)

	// Add marker to create different hashes (so they're orphaned)
	marker := []byte("ORPHAN")

	for i := 0; i < depth; i++ {
		height := forkPoint + uint64(i) + 1

		// Create hash with orphan marker to make it different
		hash := append(generateDeterministicHash(height), marker...)
		hash = hash[:32] // Truncate to 32 bytes

		block := &FixtureBlock{
			Height:       height,
			Hash:         hash,
			ParentHash:   parentHash,
			Miner:        generateDeterministicAddress(height + 999999), // Different miner
			GasUsed:      big.NewInt(int64(2000000 + height*1000)),      // Different gas
			GasLimit:     big.NewInt(8000000),
			Timestamp:    time.Now().Unix() - int64(depth-i)*12,
			Transactions: []*FixtureTransaction{}, // Empty for simplicity
		}

		orphanedBlocks[i] = block
		parentHash = block.Hash
	}

	return orphanedBlocks
}

// RandomBytes generates random bytes (for scenarios where determinism isn't required)
func RandomBytes(t *testing.T, length int) []byte {
	t.Helper()

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	require.NoError(t, err, "Failed to generate random bytes")

	return bytes
}

// AssertBlockEquals asserts two blocks are equal (for testing purposes)
func AssertBlockEquals(t *testing.T, expected, actual *FixtureBlock) {
	t.Helper()

	require.Equal(t, expected.Height, actual.Height, "Block height mismatch")
	require.Equal(t, expected.Hash, actual.Hash, "Block hash mismatch")
	require.Equal(t, expected.ParentHash, actual.ParentHash, "Parent hash mismatch")
	require.Equal(t, len(expected.Transactions), len(actual.Transactions), "Transaction count mismatch")
}

// CreateTestChain creates a simple test chain with configurable parameters
// Returns a slice of FixtureBlocks starting from height 1
func CreateTestChain(t *testing.T, blockCount, txPerBlock, logsPerTx int) []*FixtureBlock {
	t.Helper()

	blocks := GenerateTestBlocks(t, 1, blockCount, txPerBlock)

	t.Logf("Created test chain: %d blocks, %d tx/block, %d logs/tx",
		blockCount, txPerBlock, logsPerTx)

	return blocks
}

// BlockHashesMatch checks if a slice of block hashes matches expected values
func BlockHashesMatch(t *testing.T, blocks []*FixtureBlock, expectedHashes [][]byte) bool {
	t.Helper()

	if len(blocks) != len(expectedHashes) {
		return false
	}

	for i, block := range blocks {
		if string(block.Hash) != string(expectedHashes[i]) {
			return false
		}
	}

	return true
}
