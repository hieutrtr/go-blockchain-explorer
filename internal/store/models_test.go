package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBlockModel(t *testing.T) {
	now := time.Now()
	block := Block{
		Height:      12345,
		Hash:        "0xabc123",
		ParentHash:  "0xdef456",
		Miner:       "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
		GasUsed:     "1000000",
		GasLimit:    "2000000",
		Timestamp:   now.Unix(),
		TxCount:     10,
		Orphaned:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, int64(12345), block.Height)
	assert.Equal(t, "0xabc123", block.Hash)
	assert.False(t, block.Orphaned)
	assert.Equal(t, 10, block.TxCount)
}

func TestTransactionModel(t *testing.T) {
	toAddr := "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
	blockTimestamp := int64(1234567890)
	tx := Transaction{
		Hash:           "0xabc123",
		BlockHeight:    12345,
		BlockTimestamp: &blockTimestamp,
		TxIndex:        0,
		FromAddr:       "0x1234567890123456789012345678901234567890",
		ToAddr:         &toAddr,
		ValueWei:       "1000000000000000000",
		FeeWei:         "21000000000000",
		GasUsed:        "21000",
		GasPrice:       "1000000000",
		Nonce:          5,
		Success:        true,
	}

	assert.Equal(t, "0xabc123", tx.Hash)
	assert.NotNil(t, tx.ToAddr)
	assert.Equal(t, toAddr, *tx.ToAddr)
	assert.NotNil(t, tx.BlockTimestamp)
	assert.Equal(t, blockTimestamp, *tx.BlockTimestamp)
	assert.True(t, tx.Success)
}

func TestTransactionModelContractCreation(t *testing.T) {
	tx := Transaction{
		Hash:        "0xabc123",
		BlockHeight: 12345,
		TxIndex:     0,
		FromAddr:    "0x1234567890123456789012345678901234567890",
		ToAddr:      nil, // Contract creation
		ValueWei:    "0",
		FeeWei:      "21000000000000",
		GasUsed:     "21000",
		GasPrice:    "1000000000",
		Nonce:       5,
		Success:     true,
	}

	assert.Nil(t, tx.ToAddr, "contract creation should have nil to address")
}

func TestLogModel(t *testing.T) {
	topic0 := "0xtopic0..."
	topic1 := "0xtopic1..."

	log := Log{
		ID:       1,
		TxHash:   "0xabc123",
		LogIndex: 0,
		Address:  "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
		Topic0:   &topic0,
		Topic1:   &topic1,
		Topic2:   nil,
		Topic3:   nil,
		Data:     "0xdata...",
	}

	assert.Equal(t, int64(1), log.ID)
	assert.NotNil(t, log.Topic0)
	assert.NotNil(t, log.Topic1)
	assert.Nil(t, log.Topic2)
	assert.Nil(t, log.Topic3)
}

func TestChainStatsModel(t *testing.T) {
	now := time.Now()
	stats := ChainStats{
		LatestBlock:        12345,
		TotalBlocks:        12346,
		TotalTransactions:  50000,
		IndexerLagBlocks:   0,
		IndexerLagSeconds:  5,
		LastUpdated:        now,
	}

	assert.Equal(t, int64(12345), stats.LatestBlock)
	assert.Equal(t, int64(12346), stats.TotalBlocks)
	assert.Equal(t, int64(50000), stats.TotalTransactions)
}

func TestHealthStatusModel(t *testing.T) {
	now := time.Now()

	t.Run("healthy status", func(t *testing.T) {
		health := HealthStatus{
			Status:             "healthy",
			Database:           "connected",
			IndexerLastBlock:   12345,
			IndexerLastUpdated: now,
			IndexerLagSeconds:  5,
			Version:            "1.0.0",
			Errors:             []string{},
		}

		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, "connected", health.Database)
		assert.Empty(t, health.Errors)
	})

	t.Run("unhealthy status with errors", func(t *testing.T) {
		health := HealthStatus{
			Status:   "unhealthy",
			Database: "disconnected",
			Version:  "1.0.0",
			Errors:   []string{"database connection failed", "timeout exceeded"},
		}

		assert.Equal(t, "unhealthy", health.Status)
		assert.Equal(t, "disconnected", health.Database)
		assert.Len(t, health.Errors, 2)
	})
}
