package main

import (
	"context"
	"fmt"
	"log"
	"os"
	
	"github.com/hieutt50/go-blockchain-explorer/internal/rpc"
)

func main() {
	// Override config to use public Sepolia node
	os.Setenv("RPC_URL", "https://rpc.sepolia.org")
	
	cfg, err := rpc.NewConfig()
	if err != nil {
		log.Fatal("Config error:", err)
	}
	
	fmt.Printf("🔗 Connecting to: %s\n", cfg.RPCURL)
	
	client, err := rpc.NewClient(cfg)
	if err != nil {
		log.Fatal("Client error:", err)
	}
	defer client.Close()
	
	fmt.Println("✅ Connected to Sepolia testnet")
	
	// Verify it's Sepolia (chain ID 11155111)
	ctx := context.Background()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatal("ChainID error:", err)
	}
	fmt.Printf("📡 Chain ID: %s ", chainID.String())
	if chainID.String() == "11155111" {
		fmt.Println("✅ (Correct - Sepolia)")
	} else {
		fmt.Println("❌ (Wrong network!)")
	}
	
	// Fetch a known Sepolia block
	fmt.Println("\n🔍 Fetching Sepolia block 5000000...")
	block, err := client.GetBlockByNumber(ctx, 5000000)
	if err != nil {
		log.Fatal("GetBlock error:", err)
	}
	
	fmt.Printf("✅ Block #%d\n", block.NumberU64())
	fmt.Printf("   Hash: %s\n", block.Hash().Hex())
	fmt.Printf("   Transactions: %d\n", len(block.Transactions()))
	fmt.Printf("\n🎉 RPC client is working perfectly!\n")
}
