package store

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")
)

// Store provides database query methods for the API
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store instance
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ListBlocks returns a paginated list of non-orphaned blocks
func (s *Store) ListBlocks(ctx context.Context, limit, offset int) ([]Block, int64, error) {
	// Get total count of non-orphaned blocks
	var total int64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM blocks
		WHERE orphaned = FALSE
	`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count blocks: %w", err)
	}

	// Get paginated blocks
	rows, err := s.pool.Query(ctx, `
		SELECT height, hash, parent_hash, miner, gas_used, gas_limit, timestamp, tx_count, orphaned
		FROM blocks
		WHERE orphaned = FALSE
		ORDER BY height DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer rows.Close()

	blocks := make([]Block, 0, limit)
	for rows.Next() {
		var b Block
		var hashBytes, parentHashBytes, minerBytes []byte

		err := rows.Scan(&b.Height, &hashBytes, &parentHashBytes, &minerBytes,
			&b.GasUsed, &b.GasLimit, &b.Timestamp, &b.TxCount, &b.Orphaned)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan block: %w", err)
		}

		// Convert bytes to 0x-prefixed hex strings
		b.Hash = "0x" + hex.EncodeToString(hashBytes)
		b.ParentHash = "0x" + hex.EncodeToString(parentHashBytes)
		b.Miner = "0x" + hex.EncodeToString(minerBytes)

		blocks = append(blocks, b)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating blocks: %w", err)
	}

	return blocks, total, nil
}

// GetBlockByHeight returns a single block by height
func (s *Store) GetBlockByHeight(ctx context.Context, height int64) (*Block, error) {
	var b Block
	var hashBytes, parentHashBytes, minerBytes []byte

	err := s.pool.QueryRow(ctx, `
		SELECT height, hash, parent_hash, miner, gas_used, gas_limit, timestamp, tx_count, orphaned
		FROM blocks
		WHERE height = $1 AND orphaned = FALSE
	`, height).Scan(&b.Height, &hashBytes, &parentHashBytes, &minerBytes,
		&b.GasUsed, &b.GasLimit, &b.Timestamp, &b.TxCount, &b.Orphaned)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get block by height: %w", err)
	}

	b.Hash = "0x" + hex.EncodeToString(hashBytes)
	b.ParentHash = "0x" + hex.EncodeToString(parentHashBytes)
	b.Miner = "0x" + hex.EncodeToString(minerBytes)

	return &b, nil
}

// GetBlockByHash returns a single block by hash
func (s *Store) GetBlockByHash(ctx context.Context, blockHash string) (*Block, error) {
	// Remove 0x prefix if present
	hashStr := blockHash
	if len(hashStr) > 2 && hashStr[:2] == "0x" {
		hashStr = hashStr[2:]
	}

	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash: %w", err)
	}

	var b Block
	var hashBytesResult, parentHashBytes, minerBytes []byte

	err = s.pool.QueryRow(ctx, `
		SELECT height, hash, parent_hash, miner, gas_used, gas_limit, timestamp, tx_count, orphaned
		FROM blocks
		WHERE hash = $1 AND orphaned = FALSE
	`, hashBytes).Scan(&b.Height, &hashBytesResult, &parentHashBytes, &minerBytes,
		&b.GasUsed, &b.GasLimit, &b.Timestamp, &b.TxCount, &b.Orphaned)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get block by hash: %w", err)
	}

	b.Hash = "0x" + hex.EncodeToString(hashBytesResult)
	b.ParentHash = "0x" + hex.EncodeToString(parentHashBytes)
	b.Miner = "0x" + hex.EncodeToString(minerBytes)

	return &b, nil
}

// GetTransaction returns a single transaction by hash
func (s *Store) GetTransaction(ctx context.Context, txHash string) (*Transaction, error) {
	// Remove 0x prefix if present
	hashStr := txHash
	if len(hashStr) > 2 && hashStr[:2] == "0x" {
		hashStr = hashStr[2:]
	}

	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction hash: %w", err)
	}

	var tx Transaction
	var hashBytesResult, fromBytes []byte
	var toAddr *[]byte

	err = s.pool.QueryRow(ctx, `
		SELECT hash, block_height, tx_index, from_addr, to_addr, value_wei, fee_wei,
		       gas_used, gas_price, nonce, success
		FROM transactions
		WHERE hash = $1
	`, hashBytes).Scan(&hashBytesResult, &tx.BlockHeight, &tx.TxIndex, &fromBytes, &toAddr,
		&tx.ValueWei, &tx.FeeWei, &tx.GasUsed, &tx.GasPrice, &tx.Nonce, &tx.Success)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	tx.Hash = "0x" + hex.EncodeToString(hashBytesResult)
	tx.FromAddr = "0x" + hex.EncodeToString(fromBytes)

	if toAddr != nil {
		toAddrStr := "0x" + hex.EncodeToString(*toAddr)
		tx.ToAddr = &toAddrStr
	}

	return &tx, nil
}

// GetAddressTransactions returns paginated transactions for an address
func (s *Store) GetAddressTransactions(ctx context.Context, address string, limit, offset int) ([]Transaction, int64, error) {
	// Remove 0x prefix if present
	addrStr := address
	if len(addrStr) > 2 && addrStr[:2] == "0x" {
		addrStr = addrStr[2:]
	}

	addrBytes, err := hex.DecodeString(addrStr)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid address: %w", err)
	}

	// Get total count
	var total int64
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM transactions
		WHERE from_addr = $1 OR to_addr = $1
	`, addrBytes).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Get paginated transactions with block timestamp
	rows, err := s.pool.Query(ctx, `
		SELECT t.hash, t.block_height, b.timestamp, t.tx_index, t.from_addr, t.to_addr,
		       t.value_wei, t.fee_wei, t.gas_used, t.gas_price, t.nonce, t.success
		FROM transactions t
		LEFT JOIN blocks b ON t.block_height = b.height AND b.orphaned = FALSE
		WHERE t.from_addr = $1 OR t.to_addr = $1
		ORDER BY t.block_height DESC, t.tx_index DESC
		LIMIT $2 OFFSET $3
	`, addrBytes, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	txs := make([]Transaction, 0, limit)
	for rows.Next() {
		var tx Transaction
		var hashBytes, fromBytes []byte
		var toAddr *[]byte

		err := rows.Scan(&hashBytes, &tx.BlockHeight, &tx.BlockTimestamp, &tx.TxIndex, &fromBytes, &toAddr,
			&tx.ValueWei, &tx.FeeWei, &tx.GasUsed, &tx.GasPrice, &tx.Nonce, &tx.Success)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction: %w", err)
		}

		tx.Hash = "0x" + hex.EncodeToString(hashBytes)
		tx.FromAddr = "0x" + hex.EncodeToString(fromBytes)

		if toAddr != nil {
			toAddrStr := "0x" + hex.EncodeToString(*toAddr)
			tx.ToAddr = &toAddrStr
		}

		txs = append(txs, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating transactions: %w", err)
	}

	return txs, total, nil
}

// QueryLogs returns paginated event logs with optional filters
func (s *Store) QueryLogs(ctx context.Context, address, topic0 *string, limit, offset int) ([]Log, int64, error) {
	// Build dynamic query based on filters
	query := `SELECT id, tx_hash, log_index, address, topic0, topic1, topic2, topic3, data FROM logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM logs WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	var addressBytes []byte
	var topic0Bytes []byte

	if address != nil && *address != "" {
		addrStr := *address
		if len(addrStr) > 2 && addrStr[:2] == "0x" {
			addrStr = addrStr[2:]
		}
		var err error
		addressBytes, err = hex.DecodeString(addrStr)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid address: %w", err)
		}
		argCount++
		query += fmt.Sprintf(" AND address = $%d", argCount)
		countQuery += fmt.Sprintf(" AND address = $%d", argCount)
		args = append(args, addressBytes)
	}

	if topic0 != nil && *topic0 != "" {
		topicStr := *topic0
		if len(topicStr) > 2 && topicStr[:2] == "0x" {
			topicStr = topicStr[2:]
		}
		var err error
		topic0Bytes, err = hex.DecodeString(topicStr)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid topic0: %w", err)
		}
		argCount++
		query += fmt.Sprintf(" AND topic0 = $%d", argCount)
		countQuery += fmt.Sprintf(" AND topic0 = $%d", argCount)
		args = append(args, topic0Bytes)
	}

	// Get total count
	var total int64
	err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count logs: %w", err)
	}

	// Add pagination
	argCount++
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	// Get paginated logs
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	logs := make([]Log, 0, limit)
	for rows.Next() {
		var log Log
		var txHashBytes, addressBytesResult, dataBytes []byte
		var topic0Ptr, topic1Ptr, topic2Ptr, topic3Ptr *[]byte

		err := rows.Scan(&log.ID, &txHashBytes, &log.LogIndex, &addressBytesResult,
			&topic0Ptr, &topic1Ptr, &topic2Ptr, &topic3Ptr, &dataBytes)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan log: %w", err)
		}

		log.TxHash = "0x" + hex.EncodeToString(txHashBytes)
		log.Address = "0x" + hex.EncodeToString(addressBytesResult)
		log.Data = "0x" + hex.EncodeToString(dataBytes)

		if topic0Ptr != nil {
			topic0Str := "0x" + hex.EncodeToString(*topic0Ptr)
			log.Topic0 = &topic0Str
		}
		if topic1Ptr != nil {
			topic1Str := "0x" + hex.EncodeToString(*topic1Ptr)
			log.Topic1 = &topic1Str
		}
		if topic2Ptr != nil {
			topic2Str := "0x" + hex.EncodeToString(*topic2Ptr)
			log.Topic2 = &topic2Str
		}
		if topic3Ptr != nil {
			topic3Str := "0x" + hex.EncodeToString(*topic3Ptr)
			log.Topic3 = &topic3Str
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating logs: %w", err)
	}

	return logs, total, nil
}

// GetChainStats returns blockchain statistics
func (s *Store) GetChainStats(ctx context.Context) (*ChainStats, error) {
	var stats ChainStats

	// Get latest block height and timestamp
	var latestTimestamp int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(height), 0), COALESCE(MAX(timestamp), 0)
		FROM blocks
		WHERE orphaned = FALSE
	`).Scan(&stats.LatestBlock, &latestTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	// Get total blocks
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM blocks
		WHERE orphaned = FALSE
	`).Scan(&stats.TotalBlocks)
	if err != nil {
		return nil, fmt.Errorf("failed to count blocks: %w", err)
	}

	// Get total transactions
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM transactions
	`).Scan(&stats.TotalTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Calculate indexer lag
	currentTime := time.Now().Unix()
	stats.IndexerLagSeconds = currentTime - latestTimestamp
	stats.IndexerLagBlocks = 0 // Would need external chain head info to calculate

	stats.LastUpdated = time.Now()

	return &stats, nil
}

// CheckHealth performs a database health check
func (s *Store) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Status:   "healthy",
		Database: "connected",
		Version:  "1.0.0",
		Errors:   []string{},
	}

	// Test database connectivity
	var result int
	err := s.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		status.Status = "unhealthy"
		status.Database = "disconnected"
		status.Errors = append(status.Errors, fmt.Sprintf("database connection failed: %v", err))
		return status, nil // Return status, not error (we want to report unhealthy state)
	}

	// Get latest block info for indexer status
	var latestTimestamp int64
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(height), 0), COALESCE(MAX(timestamp), 0), COALESCE(MAX(updated_at), NOW())
		FROM blocks
		WHERE orphaned = FALSE
	`).Scan(&status.IndexerLastBlock, &latestTimestamp, &status.IndexerLastUpdated)
	if err != nil {
		// Not a critical error, just log it
		status.Errors = append(status.Errors, fmt.Sprintf("failed to get indexer status: %v", err))
	} else {
		currentTime := time.Now().Unix()
		status.IndexerLagSeconds = currentTime - latestTimestamp
	}

	return status, nil
}

// MarkBlocksOrphaned marks all blocks in the height range [startHeight, endHeight] as orphaned
// This method is used during chain reorganization (reorg) handling to mark blocks that are no
// longer part of the canonical chain. Uses a database transaction for atomicity.
// Story 1.5: Chain Reorganization Detection and Recovery - AC3
func (s *Store) MarkBlocksOrphaned(ctx context.Context, startHeight, endHeight uint64) error {
	// Begin transaction for atomic UPDATE operation
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if commit not reached

	// Execute UPDATE statement to mark blocks as orphaned
	// Soft delete pattern: SET orphaned = true (never DELETE)
	query := `UPDATE blocks SET orphaned = true, updated_at = NOW() WHERE height >= $1 AND height <= $2`
	result, err := tx.Exec(ctx, query, startHeight, endHeight)
	if err != nil {
		return fmt.Errorf("failed to mark blocks as orphaned: %w", err)
	}

	// Verify that blocks were actually updated
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		// This is not necessarily an error - blocks may already be orphaned or not exist
		// Log warning but don't fail the transaction
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
