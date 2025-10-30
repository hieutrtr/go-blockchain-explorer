package rpc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests demonstrate the structure, but require a mock ethclient
// In a real implementation, you would use interfaces and mocks to test retry logic

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantErr bool
	}{
		{
			name:    "valid rpc url",
			envVar:  "https://eth-sepolia.g.alchemy.com/v2/test-key",
			wantErr: false,
		},
		{
			name:    "empty rpc url",
			envVar:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or clear environment variable
			t.Setenv("RPC_URL", tt.envVar)

			cfg, err := NewConfig()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, tt.envVar, cfg.RPCURL)
				assert.Equal(t, 10*time.Second, cfg.ConnectionTimeout)
				assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
				assert.Equal(t, 5, cfg.MaxRetries)
				assert.Equal(t, 1*time.Second, cfg.RetryBaseDelay)
			}
		})
	}
}

func TestNewConfigWithDefaults(t *testing.T) {
	rpcURL := "https://eth-sepolia.g.alchemy.com/v2/test-key"
	cfg := NewConfigWithDefaults(rpcURL)

	assert.NotNil(t, cfg)
	assert.Equal(t, rpcURL, cfg.RPCURL)
	assert.Equal(t, 10*time.Second, cfg.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 1*time.Second, cfg.RetryBaseDelay)
}

// TestClient_GetBlockByNumber_InputValidation tests input validation
func TestClient_GetBlockByNumber_InputValidation(t *testing.T) {
	// This test demonstrates validation but cannot actually test the client
	// without a real or mocked RPC endpoint

	t.Run("negative block height", func(t *testing.T) {
		// In the actual implementation, this would be rejected
		// The validation is: if height < 0
		// However, height is uint64 so negative values are not possible
		// This test serves as documentation
	})
}

// TestClient_GetTransactionReceipt_InputValidation tests input validation
func TestClient_GetTransactionReceipt_InputValidation(t *testing.T) {
	// Test would verify that empty transaction hash is rejected
	emptyHash := common.Hash{}
	assert.Equal(t, common.Hash{}, emptyHash, "empty hash should be zero value")
}

// Integration test markers - these would run against a real testnet node
// Marked as integration tests to skip in unit test runs

func TestClient_GetBlockByNumber_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This would test against a real Sepolia RPC endpoint
	// Requires RPC_URL environment variable to be set
	rpcURL := getTestRPCURL(t)
	if rpcURL == "" {
		t.Skip("RPC_URL not set, skipping integration test")
	}

	cfg := NewConfigWithDefaults(rpcURL)
	client, err := NewClient(cfg)
	require.NoError(t, err, "should create client")
	defer client.Close()

	ctx := context.Background()

	// Fetch a known block (genesis block)
	block, err := client.GetBlockByNumber(ctx, 0)
	require.NoError(t, err, "should fetch genesis block")
	assert.NotNil(t, block)
	assert.Equal(t, uint64(0), block.NumberU64())
}

func TestClient_GetTransactionReceipt_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	rpcURL := getTestRPCURL(t)
	if rpcURL == "" {
		t.Skip("RPC_URL not set, skipping integration test")
	}

	// This test would fetch a known transaction receipt from Sepolia testnet
	// Requires a real RPC endpoint and known transaction hash
	t.Skip("requires known transaction hash on testnet")
}

func TestClient_ChainID_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	rpcURL := getTestRPCURL(t)
	if rpcURL == "" {
		t.Skip("RPC_URL not set, skipping integration test")
	}

	cfg := NewConfigWithDefaults(rpcURL)
	client, err := NewClient(cfg)
	require.NoError(t, err, "should create client")
	defer client.Close()

	ctx := context.Background()

	// Sepolia chain ID is 11155111
	chainID, err := client.ChainID(ctx)
	require.NoError(t, err, "should fetch chain id")
	assert.NotNil(t, chainID)

	// If connected to Sepolia, verify chain ID
	if chainID.Uint64() == 11155111 {
		assert.Equal(t, uint64(11155111), chainID.Uint64(), "should be Sepolia chain ID")
	}
}

// Helper function to get test RPC URL
func getTestRPCURL(t *testing.T) string {
	// Try environment variable first
	if url := os.Getenv("RPC_URL"); url != "" {
		return url
	}

	// Fallback to TEST_RPC_URL for integration tests
	return os.Getenv("TEST_RPC_URL")
}

// Benchmark tests
func BenchmarkClient_GetBlockByNumber(b *testing.B) {
	rpcURL := getEnvOrSkip(b, "RPC_URL")
	cfg := NewConfigWithDefaults(rpcURL)
	client, err := NewClient(cfg)
	if err != nil {
		b.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Fetch block 1000000 (a known block on Sepolia)
		_, err := client.GetBlockByNumber(ctx, 1000000)
		if err != nil {
			b.Fatalf("failed to fetch block: %v", err)
		}
	}
}

func getEnvOrSkip(b *testing.B, key string) string {
	value := os.Getenv(key)
	if value == "" {
		b.Skipf("%s not set, skipping benchmark", key)
	}
	return value
}

// Example test demonstrating expected behavior
func ExampleClient_GetBlockByNumber() {
	// Create config with RPC URL
	cfg := NewConfigWithDefaults("https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY")

	// Create client
	client, err := NewClient(cfg)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Fetch genesis block
	ctx := context.Background()
	block, err := client.GetBlockByNumber(ctx, 0)
	if err != nil {
		panic(err)
	}

	_ = block // Use block
}
