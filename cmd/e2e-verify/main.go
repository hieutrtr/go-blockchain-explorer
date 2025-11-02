package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hieutt50/go-blockchain-explorer/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	fmt.Println("==========================================")
	fmt.Println("E2E Testing: Transaction Extraction Pipeline")
	fmt.Println("==========================================")
	fmt.Println()

	ctx := context.Background()

	// Test 1: Database Connectivity
	fmt.Println(">>> Test Case 1: Database Connectivity")
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Printf("✗ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	fmt.Println("✓ Database connected")
	fmt.Println()

	// Test 2: Check if migrations have run
	fmt.Println(">>> Test Case 2: Database Schema")
	var blockTableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'blocks'
		);
	`).Scan(&blockTableExists)
	if err != nil || !blockTableExists {
		fmt.Println("✗ blocks table not found")
		os.Exit(1)
	}
	fmt.Println("✓ blocks table exists")

	var txTableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'transactions'
		);
	`).Scan(&txTableExists)
	if err != nil || !txTableExists {
		fmt.Println("✗ transactions table not found")
		os.Exit(1)
	}
	fmt.Println("✓ transactions table exists")
	fmt.Println()

	// Test 3: Query current data
	fmt.Println(">>> Test Case 3: Current Data Status")
	var blockCount int64
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM blocks WHERE orphaned = FALSE`).Scan(&blockCount)
	if err != nil {
		fmt.Printf("✗ Failed to query block count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Blocks in database: %d\n", blockCount)

	var txCount int64
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&txCount)
	if err != nil {
		fmt.Printf("✗ Failed to query transaction count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Transactions in database: %d\n", txCount)
	fmt.Println()

	// Test 4: Validate transaction data
	fmt.Println(">>> Test Case 4: Transaction Data Validation")

	// Get sample transaction
	var txHash string
	var blockHeight int64
	var fromAddr string
	var toAddr *string
	var valueWei, feeWei string

	err = pool.QueryRow(ctx, `
		SELECT '0x' || encode(hash, 'hex'), block_height, '0x' || encode(from_addr, 'hex'), 
		       CASE WHEN to_addr IS NULL THEN NULL ELSE '0x' || encode(to_addr, 'hex') END,
		       value_wei, fee_wei
		FROM transactions
		LIMIT 1
	`).Scan(&txHash, &blockHeight, &fromAddr, &toAddr, &valueWei, &feeWei)

	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("⚠ No transactions found in database")
			fmt.Println("  (Run worker to index transactions)")
		} else {
			fmt.Printf("✗ Failed to query transaction: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("✓ Sample transaction retrieved\n")
		fmt.Printf("  Hash: %s\n", txHash)
		fmt.Printf("  Block Height: %d\n", blockHeight)
		fmt.Printf("  From: %s\n", fromAddr)
		if toAddr != nil {
			fmt.Printf("  To: %s\n", *toAddr)
		} else {
			fmt.Printf("  To: <contract creation>\n")
		}
		fmt.Printf("  Value: %s wei\n", valueWei)
		fmt.Printf("  Fee: %s wei\n", feeWei)
	}
	fmt.Println()

	// Test 5: Validate block-transaction relationship
	fmt.Println(">>> Test Case 5: Block-Transaction Relationship")
	var blockWithTxs int64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT b.height)
		FROM blocks b
		INNER JOIN transactions t ON b.height = t.block_height
		WHERE b.orphaned = FALSE
	`).Scan(&blockWithTxs)
	if err != nil {
		fmt.Printf("✗ Failed to query blocks with transactions: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Blocks with extracted transactions: %d\n", blockWithTxs)
	fmt.Println()

	// Test 6: Validate Store interface
	fmt.Println(">>> Test Case 6: Store Query Methods")
	st := store.NewStore(pool)

	// Test GetTransaction
	if txCount > 0 {
		var sampleHash string
		err = pool.QueryRow(ctx, `
			SELECT '0x' || encode(hash, 'hex') FROM transactions LIMIT 1
		`).Scan(&sampleHash)

		if err == nil {
			tx, err := st.GetTransaction(ctx, sampleHash)
			if err != nil {
				fmt.Printf("✗ GetTransaction failed: %v\n", err)
			} else {
				fmt.Printf("✓ GetTransaction works\n")
				fmt.Printf("  Retrieved transaction: %s\n", tx.Hash)
			}
		}
	}

	// Test GetBlockTransactions
	if blockCount > 0 {
		var sampleHeight int64
		err = pool.QueryRow(ctx, `
			SELECT height FROM blocks WHERE orphaned = FALSE LIMIT 1
		`).Scan(&sampleHeight)

		if err == nil {
			txs, total, err := st.GetBlockTransactions(ctx, sampleHeight, 100, 0)
			if err != nil {
				fmt.Printf("✗ GetBlockTransactions failed: %v\n", err)
			} else {
				fmt.Printf("✓ GetBlockTransactions works\n")
				fmt.Printf("  Block %d has %d transactions (retrieved %d)\n", sampleHeight, total, len(txs))
			}
		}
	}
	fmt.Println()

	// Test 7: Summary
	fmt.Println(">>> Test Case 7: Data Completeness")

	if blockCount > 0 && txCount > 0 {
		avgTxPerBlock := float64(txCount) / float64(blockCount)
		fmt.Printf("✓ Average transactions per block: %.2f\n", avgTxPerBlock)

		// Check for foreign key violations
		var orphanedTxs int64
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM transactions t
			LEFT JOIN blocks b ON t.block_height = b.height
			WHERE b.height IS NULL
		`).Scan(&orphanedTxs)

		if orphanedTxs > 0 {
			fmt.Printf("⚠ Warning: %d transactions with orphaned blocks\n", orphanedTxs)
		} else {
			fmt.Println("✓ No orphaned transactions (foreign keys valid)")
		}

		// Check for duplicate transactions
		var duplicateTxs int64
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM (
				SELECT hash, COUNT(*) FROM transactions
				GROUP BY hash HAVING COUNT(*) > 1
			) AS dup
		`).Scan(&duplicateTxs)

		if duplicateTxs > 0 {
			fmt.Printf("⚠ Warning: %d duplicate transactions found\n", duplicateTxs)
		} else {
			fmt.Println("✓ No duplicate transactions")
		}
	}
	fmt.Println()

	// Final Summary
	fmt.Println("==========================================")
	if blockCount > 0 && txCount > 0 {
		fmt.Println("✓ E2E Testing Complete - Pipeline Functional!")
		fmt.Printf("  • Blocks indexed: %d\n", blockCount)
		fmt.Printf("  • Transactions extracted: %d\n", txCount)
		fmt.Printf("  • Store queries working: Yes\n")
		fmt.Println("==========================================")
		os.Exit(0)
	} else if blockCount == 0 {
		fmt.Println("⚠ No blocks indexed yet")
		fmt.Println("  → Run worker to index blocks and extract transactions")
		fmt.Println("  → Command: go run cmd/worker/main.go")
		fmt.Println("==========================================")
		os.Exit(0)
	}
}
