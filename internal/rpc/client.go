package rpc

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// Client wraps go-ethereum's ethclient with retry logic and structured logging
type Client struct {
	ethClient *ethclient.Client
	config    *Config
}

// NewClient creates a new RPC client with the provided configuration
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create context with connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	util.Info("connecting to ethereum rpc",
		"url_length", len(config.RPCURL), // Don't log full URL (may contain API key)
		"connection_timeout", config.ConnectionTimeout.String(),
	)

	// Connect to Ethereum RPC endpoint
	ethClient, err := ethclient.DialContext(ctx, config.RPCURL)
	if err != nil {
		util.Error("failed to connect to rpc endpoint",
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to connect to RPC endpoint: %w", err)
	}

	util.Info("successfully connected to ethereum rpc")

	return &Client{
		ethClient: ethClient,
		config:    config,
	}, nil
}

// Close closes the RPC client connection
func (c *Client) Close() {
	if c.ethClient != nil {
		c.ethClient.Close()
		util.Info("rpc client connection closed")
	}
}

// GetBlockByNumber fetches a block by its height with automatic retry logic
func (c *Client) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
	// Input validation
	if height < 0 {
		return nil, fmt.Errorf("block height cannot be negative: %d", height)
	}

	startTime := time.Now()

	util.Info("fetching block",
		"method", "eth_getBlockByNumber",
		"block_height", height,
	)

	var block *types.Block
	var lastError error

	// Create operation closure for retry logic
	operation := func() error {
		// Create context with request timeout
		reqCtx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
		defer cancel()

		// Fetch block with transactions
		blk, err := c.ethClient.BlockByNumber(reqCtx, big.NewInt(int64(height)))
		if err != nil {
			lastError = err
			return err
		}

		block = blk
		return nil
	}

	// Execute with retry logic
	retryCfg := &retryConfig{
		maxRetries: c.config.MaxRetries,
		baseDelay:  c.config.RetryBaseDelay,
	}

	err := retryWithBackoff(
		ctx,
		retryCfg,
		operation,
		util.GlobalLogger,
		fmt.Sprintf("GetBlockByNumber(height=%d)", height),
	)

	duration := time.Since(startTime)

	if err != nil {
		// Record RPC error metrics
		if lastError != nil {
			errorType := classifyError(lastError)
			metricsErrorType := errorTypeToMetricsLabel(errorType)
			util.RecordRPCError(metricsErrorType)
		}

		util.Error("failed to fetch block",
			"method", "eth_getBlockByNumber",
			"block_height", height,
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	util.Info("successfully fetched block",
		"method", "eth_getBlockByNumber",
		"block_height", height,
		"block_hash", block.Hash().Hex(),
		"tx_count", len(block.Transactions()),
		"duration_ms", duration.Milliseconds(),
	)

	return block, nil
}

// GetTransactionReceipt fetches a transaction receipt by hash with automatic retry logic
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	// Input validation
	if txHash == (common.Hash{}) {
		return nil, fmt.Errorf("transaction hash cannot be empty")
	}

	startTime := time.Now()

	util.Info("fetching transaction receipt",
		"method", "eth_getTransactionReceipt",
		"tx_hash", txHash.Hex(),
	)

	var receipt *types.Receipt

	// Create operation closure for retry logic
	operation := func() error {
		// Create context with request timeout
		reqCtx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
		defer cancel()

		// Fetch transaction receipt
		rcpt, err := c.ethClient.TransactionReceipt(reqCtx, txHash)
		if err != nil {
			// Check if transaction not found (not an error to retry)
			if err == ethereum.NotFound {
				return NewRPCError("transaction not found", err)
			}
			return err
		}

		receipt = rcpt
		return nil
	}

	// Execute with retry logic
	retryCfg := &retryConfig{
		maxRetries: c.config.MaxRetries,
		baseDelay:  c.config.RetryBaseDelay,
	}

	err := retryWithBackoff(
		ctx,
		retryCfg,
		operation,
		util.GlobalLogger,
		fmt.Sprintf("GetTransactionReceipt(hash=%s)", txHash.Hex()),
	)

	duration := time.Since(startTime)

	if err != nil {
		util.Error("failed to fetch transaction receipt",
			"method", "eth_getTransactionReceipt",
			"tx_hash", txHash.Hex(),
			"error", err.Error(),
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	util.Info("successfully fetched transaction receipt",
		"method", "eth_getTransactionReceipt",
		"tx_hash", txHash.Hex(),
		"block_number", receipt.BlockNumber.Uint64(),
		"status", receipt.Status,
		"gas_used", receipt.GasUsed,
		"duration_ms", duration.Milliseconds(),
	)

	return receipt, nil
}

// ChainID returns the chain ID of the connected network
// Useful for verifying we're connected to the correct network
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	chainID, err := c.ethClient.ChainID(reqCtx)
	if err != nil {
		util.Error("failed to fetch chain id",
			"error", err.Error(),
		)
		return nil, err
	}

	util.Info("fetched chain id",
		"chain_id", chainID.String(),
	)

	return chainID, nil
}
