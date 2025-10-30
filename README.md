# Blockchain Explorer

A production-grade Ethereum blockchain indexer and query platform built in Go.

## Overview

This project indexes blockchain data from Ethereum Sepolia testnet, providing APIs for querying blocks, transactions, and logs with real-time updates.

## Features

- **Historical Block Indexing**: Parallel backfill for configurable block ranges
- **Real-Time Monitoring**: Live-tail mechanism for new blocks as they're produced
- **Chain Reorganization Handling**: Automatic detection and recovery
- **REST API**: Query blocks, transactions, and logs
- **WebSocket Streaming**: Real-time updates for new blocks and transactions
- **Minimal Frontend**: Single-page application for demonstration
- **Observability**: Prometheus metrics and structured JSON logging

## Project Structure

```
go-blockchain-explorer/
├── internal/
│   └── rpc/              # RPC client with retry logic (Story 1.1) ✅
│       ├── client.go     # Main client implementation
│       ├── config.go     # Configuration management
│       ├── errors.go     # Error classification
│       ├── retry.go      # Exponential backoff logic
│       └── *_test.go     # Unit tests
├── docs/                 # Documentation
│   ├── stories/          # Story files
│   ├── PRD.md           # Product requirements
│   ├── solution-architecture.md  # Architecture design
│   └── tech-spec-*.md   # Technical specifications
├── go.mod                # Go module definition
└── README.md             # This file
```

## Getting Started

### Prerequisites

- Go 1.24+ (required by go-ethereum v1.16.5)
- Ethereum RPC endpoint (Alchemy, Infura, or public node)
- PostgreSQL 16 (for future stories)

### Installation

```bash
# Clone the repository
git clone https://github.com/hieutt50/go-blockchain-explorer.git
cd go-blockchain-explorer

# Install dependencies
go mod download

# Set up environment variables
export RPC_URL="https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY"

# Run tests
go test ./... -v
```

### Configuration

The RPC client requires the `RPC_URL` environment variable:

```bash
export RPC_URL="https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY"
```

Supported RPC providers:
- **Alchemy**: `https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY`
- **Infura**: `https://sepolia.infura.io/v3/YOUR_API_KEY`
- **Public nodes**: `https://rpc.sepolia.org` (rate limited)

### Usage Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hieutt50/go-blockchain-explorer/internal/rpc"
)

func main() {
	// Create configuration from environment
	cfg, err := rpc.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create RPC client
	client, err := rpc.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Fetch a block
	ctx := context.Background()
	block, err := client.GetBlockByNumber(ctx, 1000000)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Block #%d: %s\\n", block.NumberU64(), block.Hash().Hex())
	fmt.Printf("Transactions: %d\\n", len(block.Transactions()))
}
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./internal/rpc -v

# Run with race detector
go test ./internal/rpc -race

# Run with coverage
go test ./internal/rpc -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require a real Ethereum RPC endpoint:

```bash
# Run integration tests (requires RPC_URL environment variable)
export RPC_URL="https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY"
go test ./internal/rpc -v -run Integration

# Skip integration tests
go test ./internal/rpc -v -short
```

## Architecture

### RPC Client (Story 1.1)

The RPC client provides a robust interface to Ethereum nodes with:

- **Automatic Retry Logic**: Exponential backoff for transient failures (1s, 2s, 4s, 8s, 16s)
- **Error Classification**: Distinguishes transient vs permanent errors
- **Structured Logging**: JSON-formatted logs with operation context
- **Timeout Management**: 10s connection timeout, 30s request timeout
- **Rate Limit Handling**: Automatic backoff for HTTP 429 responses

### Error Handling

Errors are classified into three types:

1. **Transient** (retry): Network timeouts, connection refused, DNS errors
2. **Permanent** (fail immediately): Invalid parameters, method not found
3. **Rate Limit** (backoff + retry): HTTP 429, quota exceeded

## Development Status

- ✅ **Story 1.1**: RPC Client with Retry Logic (Complete)
- 📋 **Story 1.2**: PostgreSQL Schema and Migrations (Next)
- 📋 **Story 1.3**: Parallel Backfill Worker Pool
- 📋 **Story 1.4**: Live-Tail Mechanism
- 📋 **Story 1.5**: Chain Reorganization Handling
- 📋 **Stories 1.6-2.6**: Additional features

See `docs/sprint-status.yaml` for detailed progress tracking.

## Contributing

This is a portfolio project demonstrating production-grade Go development. Contributions are welcome!

## License

MIT License - See LICENSE file for details

## Author

Hieu - [GitHub](https://github.com/hieutt50)

## Acknowledgments

- Built with [go-ethereum](https://github.com/ethereum/go-ethereum) v1.16.5
- Uses [Sepolia testnet](https://sepolia.etherscan.io) for testing
- Follows BMad Method for structured development
