package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hieutt50/go-blockchain-explorer/internal/db"
	"github.com/hieutt50/go-blockchain-explorer/internal/store"
)

// TestIntegrationAPI runs integration tests for all API endpoints
// These tests require a running PostgreSQL database with test data
func TestIntegrationAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup test database connection
	pool := setupTestDB(t)
	if pool == nil {
		t.Skip("skipping integration test - database not available")
	}
	defer pool.Close()

	// Create test server
	config := &Config{
		Port:            8080,
		CORSOrigins:     "*",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}

	dbPool := &db.Pool{Pool: pool}
	server := NewServer(dbPool, config)
	router := server.Router()

	// Run sub-tests
	t.Run("Health Check", func(t *testing.T) {
		testHealthCheck(t, router)
	})

	t.Run("List Blocks", func(t *testing.T) {
		testListBlocks(t, router)
	})

	t.Run("Get Block by Height", func(t *testing.T) {
		testGetBlockByHeight(t, router)
	})

	t.Run("Get Block by Hash", func(t *testing.T) {
		testGetBlockByHash(t, router)
	})

	t.Run("Get Chain Stats", func(t *testing.T) {
		testGetChainStats(t, router)
	})

	t.Run("Pagination", func(t *testing.T) {
		testPagination(t, router)
	})

	t.Run("CORS Headers", func(t *testing.T) {
		testCORSHeaders(t, router)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testErrorHandling(t, router)
	})
}

// setupTestDB creates a test database connection
// Returns nil if database is not available (test will be skipped)
func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Try to connect to test database
	// Use environment variables or defaults for test database
	config, err := db.NewConfig()
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, config)
	if err != nil {
		return nil
	}

	// Test connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil
	}

	return pool.Pool
}

func testHealthCheck(t *testing.T, router http.Handler) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "health check should return 200")

	var health store.HealthStatus
	err := json.Unmarshal(w.Body.Bytes(), &health)
	require.NoError(t, err, "should parse health response")

	assert.Contains(t, []string{"healthy", "unhealthy"}, health.Status)
	assert.NotEmpty(t, health.Version)
}

func testListBlocks(t *testing.T, router http.Handler) {
	req := httptest.NewRequest("GET", "/v1/blocks?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "list blocks should return 200")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "should parse response")

	assert.Contains(t, response, "blocks")
	assert.Contains(t, response, "total")
	assert.Contains(t, response, "limit")
	assert.Contains(t, response, "offset")

	blocks, ok := response["blocks"].([]interface{})
	require.True(t, ok, "blocks should be an array")

	if len(blocks) > 0 {
		block := blocks[0].(map[string]interface{})
		assert.Contains(t, block, "height")
		assert.Contains(t, block, "hash")
		assert.Contains(t, block, "timestamp")
	}
}

func testGetBlockByHeight(t *testing.T, router http.Handler) {
	// First get a block from list to know a valid height
	req := httptest.NewRequest("GET", "/v1/blocks?limit=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("no blocks available for testing")
	}

	var listResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResponse)
	blocks := listResponse["blocks"].([]interface{})

	if len(blocks) == 0 {
		t.Skip("no blocks available for testing")
	}

	block := blocks[0].(map[string]interface{})
	height := int64(block["height"].(float64))

	// Test getting block by height
	req = httptest.NewRequest("GET", "/v1/blocks/"+string(rune(height)), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var blockResponse store.Block
	err := json.Unmarshal(w.Body.Bytes(), &blockResponse)
	require.NoError(t, err)
	assert.Equal(t, height, blockResponse.Height)
}

func testGetBlockByHash(t *testing.T, router http.Handler) {
	// First get a block from list to know a valid hash
	req := httptest.NewRequest("GET", "/v1/blocks?limit=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("no blocks available for testing")
	}

	var listResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResponse)
	blocks := listResponse["blocks"].([]interface{})

	if len(blocks) == 0 {
		t.Skip("no blocks available for testing")
	}

	block := blocks[0].(map[string]interface{})
	hash := block["hash"].(string)

	// Test getting block by hash
	req = httptest.NewRequest("GET", "/v1/blocks/"+hash, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var blockResponse store.Block
	err := json.Unmarshal(w.Body.Bytes(), &blockResponse)
	require.NoError(t, err)
	assert.Equal(t, hash, blockResponse.Hash)
}

func testGetChainStats(t *testing.T, router http.Handler) {
	req := httptest.NewRequest("GET", "/v1/stats/chain", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats store.ChainStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, stats.LatestBlock, int64(0))
	assert.GreaterOrEqual(t, stats.TotalBlocks, int64(0))
	assert.NotZero(t, stats.LastUpdated)
}

func testPagination(t *testing.T, router http.Handler) {
	// Test with different pagination parameters
	tests := []struct {
		name       string
		url        string
		expectCode int
	}{
		{"default pagination", "/v1/blocks", http.StatusOK},
		{"custom limit", "/v1/blocks?limit=5", http.StatusOK},
		{"custom offset", "/v1/blocks?offset=10", http.StatusOK},
		{"both params", "/v1/blocks?limit=5&offset=10", http.StatusOK},
		{"zero limit uses default", "/v1/blocks?limit=0", http.StatusOK},
		{"negative offset uses zero", "/v1/blocks?offset=-1", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "blocks")
				assert.Contains(t, response, "total")
			}
		})
	}
}

func testCORSHeaders(t *testing.T, router http.Handler) {
	req := httptest.NewRequest("GET", "/v1/blocks", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "*")
}

func testErrorHandling(t *testing.T, router http.Handler) {
	tests := []struct {
		name       string
		url        string
		expectCode int
	}{
		{"invalid block height", "/v1/blocks/invalid", http.StatusBadRequest},
		{"invalid block hash", "/v1/blocks/0xinvalid", http.StatusBadRequest},
		{"block not found", "/v1/blocks/999999999", http.StatusNotFound},
		{"invalid address format", "/v1/address/invalid/txs", http.StatusBadRequest},
		{"invalid tx hash", "/v1/txs/invalid", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			var errorResponse map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			require.NoError(t, err)
			assert.Contains(t, errorResponse, "error")
		})
	}
}
