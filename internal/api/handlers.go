package api

import (
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hieutt50/go-blockchain-explorer/internal/store"
)

var (
	// addressRegex validates Ethereum addresses (0x + 40 hex characters)
	addressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

	// hashRegex validates transaction/block hashes (0x + 64 hex characters)
	hashRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
)

// handleListBlocks handles GET /v1/blocks - List recent blocks with pagination
func (s *Server) handleListBlocks(w http.ResponseWriter, r *http.Request) {
	// Parse pagination (default limit=25, max=100)
	limit, offset := parsePagination(r, 25, 100)

	// Create store
	st := store.NewStore(s.pool.Pool)

	// Query blocks
	blocks, total, err := st.ListBlocks(r.Context(), limit, offset)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	// Build response
	response := map[string]interface{}{
		"blocks": blocks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGetBlock handles GET /v1/blocks/{heightOrHash} - Get block by height or hash
// This handler intelligently routes to either height-based or hash-based lookup
func (s *Server) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	// Parse parameter
	param := chi.URLParam(r, "heightOrHash")

	// Create store
	st := store.NewStore(s.pool.Pool)

	// Try to parse as height (numeric) first
	if height, err := strconv.ParseInt(param, 10, 64); err == nil && height >= 0 {
		// Query block by height
		block, err := st.GetBlockByHeight(r.Context(), height)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeNotFound(w, "block not found")
				return
			}
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, block)
		return
	}

	// Otherwise, treat as hash
	if !hashRegex.MatchString(param) {
		writeBadRequest(w, "invalid parameter: expected block height (number) or block hash (0x + 64 hex characters)")
		return
	}

	// Query block by hash
	block, err := st.GetBlockByHash(r.Context(), param)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeNotFound(w, "block not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, block)
}

// handleGetTransaction handles GET /v1/txs/{hash} - Get transaction by hash
func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	// Parse transaction hash parameter
	txHash := chi.URLParam(r, "hash")

	// Validate hash format
	if !hashRegex.MatchString(txHash) {
		writeBadRequest(w, "invalid transaction hash format (expected 0x + 64 hex characters)")
		return
	}

	// Create store
	st := store.NewStore(s.pool.Pool)

	// Query transaction
	tx, err := st.GetTransaction(r.Context(), txHash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeNotFound(w, "transaction not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tx)
}

// handleGetAddressTransactions handles GET /v1/address/{addr}/txs - Get transactions for address
func (s *Server) handleGetAddressTransactions(w http.ResponseWriter, r *http.Request) {
	// Parse address parameter
	address := chi.URLParam(r, "addr")

	// Validate address format
	if !addressRegex.MatchString(address) {
		writeBadRequest(w, "invalid address format (expected 0x + 40 hex characters)")
		return
	}

	// Parse pagination (default limit=50, max=100)
	limit, offset := parsePagination(r, 50, 100)

	// Create store
	st := store.NewStore(s.pool.Pool)

	// Query transactions
	txs, total, err := st.GetAddressTransactions(r.Context(), address, limit, offset)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	// Build response
	response := map[string]interface{}{
		"address":      address,
		"transactions": txs,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGetBlockTransactions handles GET /v1/blocks/{height}/transactions - Get transactions for a block
func (s *Server) handleGetBlockTransactions(w http.ResponseWriter, r *http.Request) {
	// Parse block height parameter
	heightParam := chi.URLParam(r, "height")

	// Parse height as integer
	height, err := strconv.ParseInt(heightParam, 10, 64)
	if err != nil || height < 0 {
		writeBadRequest(w, "invalid block height (expected non-negative integer)")
		return
	}

	// Parse pagination (default limit=100, max=1000)
	limit, offset := parsePagination(r, 100, 1000)

	// Create store
	st := store.NewStore(s.pool.Pool)

	// Query transactions
	txs, total, err := st.GetBlockTransactions(r.Context(), height, limit, offset)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	// Build response
	response := map[string]interface{}{
		"transactions": txs,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleQueryLogs handles GET /v1/logs - Query event logs with filters
func (s *Server) handleQueryLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	addressParam := r.URL.Query().Get("address")
	topic0Param := r.URL.Query().Get("topic0")

	var address, topic0 *string

	// Validate address if provided
	if addressParam != "" {
		if !addressRegex.MatchString(addressParam) {
			writeBadRequest(w, "invalid address format (expected 0x + 40 hex characters)")
			return
		}
		address = &addressParam
	}

	// Validate topic0 if provided
	if topic0Param != "" {
		if !hashRegex.MatchString(topic0Param) {
			writeBadRequest(w, "invalid topic0 format (expected 0x + 64 hex characters)")
			return
		}
		topic0 = &topic0Param
	}

	// Parse pagination (default limit=100, max=1000)
	limit, offset := parsePagination(r, 100, 1000)

	// Create store
	st := store.NewStore(s.pool.Pool)

	// Query logs
	logs, total, err := st.QueryLogs(r.Context(), address, topic0, limit, offset)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	// Build response
	response := map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleChainStats handles GET /v1/stats/chain - Get chain statistics
func (s *Server) handleChainStats(w http.ResponseWriter, r *http.Request) {
	// Create store
	st := store.NewStore(s.pool.Pool)

	// Query stats
	stats, err := st.GetChainStats(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleHealth handles GET /health - Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Create store
	st := store.NewStore(s.pool.Pool)

	// Check health
	health, err := st.CheckHealth(r.Context())
	if err != nil {
		writeServiceUnavailable(w, "health check failed")
		return
	}

	// Return appropriate status code
	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, health)
}

// validateAddress checks if an address is valid
func validateAddress(address string) bool {
	return addressRegex.MatchString(address)
}

// validateHash checks if a hash is valid
func validateHash(hash string) bool {
	return hashRegex.MatchString(hash)
}

// parseHexBytes parses a 0x-prefixed hex string to bytes
func parseHexBytes(hexStr string) ([]byte, error) {
	if len(hexStr) < 2 || hexStr[:2] != "0x" {
		return nil, errors.New("hex string must start with 0x")
	}
	return hex.DecodeString(hexStr[2:])
}
