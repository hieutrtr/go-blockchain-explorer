package store

import "time"

// Block represents a blockchain block
type Block struct {
	Height      int64     `json:"height"`
	Hash        string    `json:"hash"`         // 0x-prefixed hex
	ParentHash  string    `json:"parent_hash"`  // 0x-prefixed hex
	Miner       string    `json:"miner"`        // 0x-prefixed hex
	GasUsed     string    `json:"gas_used"`     // String to avoid precision loss
	GasLimit    string    `json:"gas_limit"`    // String to avoid precision loss
	Timestamp   int64     `json:"timestamp"`    // Unix timestamp
	TxCount     int       `json:"tx_count"`
	Orphaned    bool      `json:"orphaned"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash           string    `json:"hash"`             // 0x-prefixed hex
	BlockHeight    int64     `json:"block_height"`
	BlockTimestamp *int64    `json:"block_timestamp,omitempty"` // Unix timestamp, nullable for compatibility
	TxIndex        int       `json:"tx_index"`
	FromAddr       string    `json:"from_addr"`        // 0x-prefixed hex
	ToAddr         *string   `json:"to_addr"`          // 0x-prefixed hex, nullable for contract creation
	ValueWei       string    `json:"value_wei"`        // String to avoid precision loss
	FeeWei         string    `json:"fee_wei"`          // String to avoid precision loss
	GasUsed        string    `json:"gas_used"`         // String to avoid precision loss
	GasPrice       string    `json:"gas_price"`        // String to avoid precision loss
	Nonce          int64     `json:"nonce"`
	Success        bool      `json:"success"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
}

// Log represents an event log
type Log struct {
	ID        int64     `json:"id"`
	TxHash    string    `json:"tx_hash"`     // 0x-prefixed hex
	LogIndex  int       `json:"log_index"`
	Address   string    `json:"address"`     // 0x-prefixed hex
	Topic0    *string   `json:"topic0"`      // 0x-prefixed hex, nullable
	Topic1    *string   `json:"topic1"`      // 0x-prefixed hex, nullable
	Topic2    *string   `json:"topic2"`      // 0x-prefixed hex, nullable
	Topic3    *string   `json:"topic3"`      // 0x-prefixed hex, nullable
	Data      string    `json:"data"`        // 0x-prefixed hex
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// ChainStats represents blockchain statistics
type ChainStats struct {
	LatestBlock        int64     `json:"latest_block"`
	TotalBlocks        int64     `json:"total_blocks"`
	TotalTransactions  int64     `json:"total_transactions"`
	IndexerLagBlocks   int64     `json:"indexer_lag_blocks"`
	IndexerLagSeconds  int64     `json:"indexer_lag_seconds"`
	LastUpdated        time.Time `json:"last_updated"`
}

// HealthStatus represents system health status
type HealthStatus struct {
	Status              string    `json:"status"`               // "healthy" or "unhealthy"
	Database            string    `json:"database"`             // "connected" or "disconnected"
	IndexerLastBlock    int64     `json:"indexer_last_block"`
	IndexerLastUpdated  time.Time `json:"indexer_last_updated"`
	IndexerLagSeconds   int64     `json:"indexer_lag_seconds"`
	Version             string    `json:"version"`
	Errors              []string  `json:"errors,omitempty"`
}
