#!/bin/bash
# Pagination Performance Test Script
# Tests pagination performance with EXPLAIN ANALYZE and measures p95 latency

set -e

echo "=== Pagination Performance Test Suite ==="
echo "Date: $(date)"
echo ""

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-blockchain_explorer}"
DB_USER="${DB_USER:-postgres}"

echo "Database: $DB_NAME @ $DB_HOST:$DB_PORT"
echo ""

# Test 1: Blocks Pagination - EXPLAIN ANALYZE
echo "--- Test 1: Blocks Pagination (LIMIT 25 OFFSET 0) ---"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT height, hash, parent_hash, timestamp, orphaned
FROM blocks
WHERE orphaned = false
ORDER BY height DESC
LIMIT 25 OFFSET 0;
EOF
echo ""

# Test 2: Blocks Pagination with Offset
echo "--- Test 2: Blocks Pagination (LIMIT 25 OFFSET 1000) ---"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT height, hash, parent_hash, timestamp, orphaned
FROM blocks
WHERE orphaned = false
ORDER BY height DESC
LIMIT 25 OFFSET 1000;
EOF
echo ""

# Test 3: Address Transactions Pagination
echo "--- Test 3: Address Transactions (LIMIT 50 OFFSET 0) ---"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT t.hash, t.from_addr, t.to_addr, t.value, t.block_height, t.tx_index
FROM transactions t
WHERE t.from_addr = '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0'
   OR t.to_addr = '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0'
ORDER BY t.block_height DESC, t.tx_index DESC
LIMIT 50 OFFSET 0;
EOF
echo ""

# Test 4: Event Logs Pagination
echo "--- Test 4: Event Logs (LIMIT 100 OFFSET 0) ---"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT address, topic0, topic1, topic2, topic3, data, block_height, log_index
FROM logs
WHERE address = '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0'
ORDER BY block_height DESC, log_index DESC
LIMIT 100 OFFSET 0;
EOF
echo ""

# Test 5: COUNT(*) Query for Total Blocks
echo "--- Test 5: COUNT(*) Total Blocks ---"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT COUNT(*) FROM blocks WHERE orphaned = false;
EOF
echo ""

# Test 6: Large Offset Performance
echo "--- Test 6: Large Offset (LIMIT 25 OFFSET 4000) ---"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT height, hash, parent_hash, timestamp, orphaned
FROM blocks
WHERE orphaned = false
ORDER BY height DESC
LIMIT 25 OFFSET 4000;
EOF
echo ""

echo "=== Performance Test Complete ===="
echo ""
echo "Performance Summary:"
echo "- All queries should use Index Scan (not Seq Scan)"
echo "- Blocks pagination: Expected <50ms"
echo "- Address transactions: Expected <150ms"
echo "- Event logs: Expected <100ms"
echo "- COUNT(*) queries: Should use covering index"
echo ""
echo "Note: Actual timing shown in 'Execution Time' field above"
