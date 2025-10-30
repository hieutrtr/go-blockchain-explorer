-- Drop indexes for logs table
DROP INDEX IF EXISTS idx_logs_address;
DROP INDEX IF EXISTS idx_logs_address_topic0;
DROP INDEX IF EXISTS idx_logs_tx_hash;

-- Drop indexes for transactions table
DROP INDEX IF EXISTS idx_tx_block_index;
DROP INDEX IF EXISTS idx_tx_to_addr_block;
DROP INDEX IF EXISTS idx_tx_from_addr_block;
DROP INDEX IF EXISTS idx_tx_block_height;

-- Drop indexes for blocks table
DROP INDEX IF EXISTS idx_blocks_timestamp;
DROP INDEX IF EXISTS idx_blocks_orphaned_height;
