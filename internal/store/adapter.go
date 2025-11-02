package store

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hieutt50/go-blockchain-explorer/internal/db"
	"github.com/hieutt50/go-blockchain-explorer/internal/index"
)

// IndexerAdapter adapts the Store to implement BlockStore and BlockStoreExtended interfaces
// This allows the indexer coordinators to work with the database
type IndexerAdapter struct {
	pool *db.Pool
}

// NewIndexerAdapter creates a new adapter for indexer operations
func NewIndexerAdapter(pool *db.Pool) *IndexerAdapter {
	return &IndexerAdapter{pool: pool}
}

// GetLatestBlock returns the latest non-orphaned block from the database
func (a *IndexerAdapter) GetLatestBlock(ctx context.Context) (*index.Block, error) {
	var b index.Block
	var hashBytes, parentHashBytes, minerBytes []byte

	err := a.pool.Pool.QueryRow(ctx, `
		SELECT height, hash, parent_hash, miner, gas_used, timestamp, tx_count
		FROM blocks
		WHERE orphaned = FALSE
		ORDER BY height DESC
		LIMIT 1
	`).Scan(&b.Height, &hashBytes, &parentHashBytes, &minerBytes,
		&b.GasUsed, &b.Timestamp, &b.TxCount)

	if err != nil {
		// Return nil block if no blocks exist yet (initial state)
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	b.Hash = hashBytes
	b.ParentHash = parentHashBytes
	b.Miner = minerBytes

	return &b, nil
}

// GetBlockByHeight returns a block by height
func (a *IndexerAdapter) GetBlockByHeight(ctx context.Context, height uint64) (*index.Block, error) {
	var b index.Block
	var hashBytes, parentHashBytes, minerBytes []byte

	err := a.pool.Pool.QueryRow(ctx, `
		SELECT height, hash, parent_hash, miner, gas_used, timestamp, tx_count, orphaned
		FROM blocks
		WHERE height = $1
	`, height).Scan(&b.Height, &hashBytes, &parentHashBytes, &minerBytes,
		&b.GasUsed, &b.Timestamp, &b.TxCount, new(bool)) // orphaned placeholder

	if err != nil {
		return nil, fmt.Errorf("failed to get block by height %d: %w", height, err)
	}

	b.Hash = hashBytes
	b.ParentHash = parentHashBytes
	b.Miner = minerBytes

	return &b, nil
}

// InsertBlock inserts a single block with its transactions into the database
// Transactions are inserted in the same database transaction for consistency
func (a *IndexerAdapter) InsertBlock(ctx context.Context, block *index.Block) error {
	tx, err := a.pool.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert block
	_, err = tx.Exec(ctx, `
		INSERT INTO blocks (height, hash, parent_hash, miner, gas_used, gas_limit, timestamp, tx_count, orphaned)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (height) DO UPDATE SET
			hash = EXCLUDED.hash,
			parent_hash = EXCLUDED.parent_hash,
			miner = EXCLUDED.miner,
			gas_used = EXCLUDED.gas_used,
			gas_limit = EXCLUDED.gas_limit,
			timestamp = EXCLUDED.timestamp,
			tx_count = EXCLUDED.tx_count,
			orphaned = EXCLUDED.orphaned,
			updated_at = NOW()
	`, block.Height, block.Hash, block.ParentHash, block.Miner,
		block.GasUsed, 0, block.Timestamp, block.TxCount, false) // gas_limit not in domain model

	if err != nil {
		return fmt.Errorf("failed to insert block %d: %w", block.Height, err)
	}

	// Insert transactions extracted from block
	for _, txn := range block.Transactions {
		// Calculate fee_wei: gas_used * gas_price
		feeWei := new(big.Int).Mul(
			new(big.Int).SetUint64(txn.GasUsed),
			new(big.Int).SetUint64(txn.GasPrice),
		).String()

		_, err = tx.Exec(ctx, `
			INSERT INTO transactions (hash, block_height, tx_index, from_addr, to_addr, value_wei, fee_wei, gas_used, gas_price, nonce, success, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (hash) DO NOTHING
		`, txn.Hash, block.Height, txn.TxIndex, txn.FromAddr, txn.ToAddr,
			txn.ValueWei, feeWei, txn.GasUsed, txn.GasPrice, txn.Nonce, txn.Success, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert transaction %x for block %d: %w", txn.Hash, block.Height, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit block %d with %d transactions: %w", block.Height, len(block.Transactions), err)
	}

	if len(block.Transactions) > 0 {
		slog.Info("inserted block with transactions",
			slog.Uint64("height", block.Height),
			slog.Int("tx_count", len(block.Transactions)))
	}

	return nil
}

// MarkBlocksOrphaned marks blocks as orphaned (soft delete for reorg handling)
func (a *IndexerAdapter) MarkBlocksOrphaned(ctx context.Context, startHeight, endHeight uint64) error {
	tx, err := a.pool.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE blocks
		SET orphaned = true, updated_at = NOW()
		WHERE height >= $1 AND height <= $2
	`, startHeight, endHeight)

	if err != nil {
		return fmt.Errorf("failed to mark blocks as orphaned: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit orphaned blocks update: %w", err)
	}

	return nil
}

// ParseRPCBlock converts an ethereum block to the index.Block domain model
// Includes full transaction extraction with signature recovery
func ParseRPCBlock(rpcBlock *types.Block) *index.Block {
	if rpcBlock == nil {
		return nil
	}

	block := &index.Block{
		Height:       rpcBlock.NumberU64(),
		Hash:         rpcBlock.Hash().Bytes(),
		ParentHash:   rpcBlock.ParentHash().Bytes(),
		Timestamp:    rpcBlock.Time(),
		Miner:        rpcBlock.Coinbase().Bytes(),
		GasUsed:      rpcBlock.GasUsed(),
		TxCount:      len(rpcBlock.Transactions()),
		Transactions: make([]index.Transaction, 0, len(rpcBlock.Transactions())),
	}

	// Extract transactions from block
	for txIndex, tx := range rpcBlock.Transactions() {
		indexerTx := parseTransaction(tx, txIndex)
		block.Transactions = append(block.Transactions, indexerTx)
	}

	return block
}

// parseTransaction converts a go-ethereum transaction to index.Transaction domain model
// Uses basic mode: no receipt fetching, uses gas limit as gas_used estimate, assumes success=true
func parseTransaction(tx *types.Transaction, txIndex int) index.Transaction {
	// Recover sender address from transaction signature
	// This is required because Ethereum transactions don't include from_addr directly
	from, err := types.LatestSignerForChainID(tx.ChainId()).Sender(tx)
	var fromAddr []byte
	if err != nil {
		// Signature recovery failed - use zero address as fallback
		// This can happen for invalid transactions
		fromAddr = common.Address{}.Bytes()
	} else {
		fromAddr = from.Bytes()
	}

	// Get recipient address (nil for contract creation)
	var toAddr *[]byte
	if tx.To() != nil {
		toAddrBytes := tx.To().Bytes()
		toAddr = &toAddrBytes
	}
	// If nil, toAddr remains nil (contract creation transaction)

	// Get gas price in wei
	gasPrice := uint64(0)
	if tx.GasPrice() != nil {
		gasPrice = tx.GasPrice().Uint64()
	}

	return index.Transaction{
		Hash:     tx.Hash().Bytes(),
		TxIndex:  txIndex,
		FromAddr: fromAddr,
		ToAddr:   toAddr,
		ValueWei: tx.Value().String(), // Convert big.Int to string for precision
		GasUsed:  tx.Gas(),             // Using gas limit as estimate (no receipt)
		GasPrice: gasPrice,
		Nonce:    tx.Nonce(),
		Success:  true,            // Assume success (no receipt data)
		Logs:     []index.Log{},   // Empty for basic mode (no receipt)
	}
}

// FormatBlockForDisplay converts index.Block to display-friendly hex strings
func FormatBlockForDisplay(block *index.Block) map[string]interface{} {
	return map[string]interface{}{
		"height":      block.Height,
		"hash":        "0x" + hex.EncodeToString(block.Hash),
		"parent_hash": "0x" + hex.EncodeToString(block.ParentHash),
		"timestamp":   block.Timestamp,
		"miner":       "0x" + hex.EncodeToString(block.Miner),
		"gas_used":    block.GasUsed,
		"tx_count":    block.TxCount,
	}
}
