-- Indexes for blocks table
CREATE INDEX idx_blocks_orphaned_height ON blocks(orphaned, height DESC);
CREATE INDEX idx_blocks_timestamp ON blocks(timestamp DESC);

-- Indexes for transactions table
CREATE INDEX idx_tx_block_height ON transactions(block_height);
CREATE INDEX idx_tx_from_addr_block ON transactions(from_addr, block_height DESC);
CREATE INDEX idx_tx_to_addr_block ON transactions(to_addr, block_height DESC) WHERE to_addr IS NOT NULL;
CREATE INDEX idx_tx_block_index ON transactions(block_height, tx_index);

-- Indexes for logs table
CREATE INDEX idx_logs_tx_hash ON logs(tx_hash);
CREATE INDEX idx_logs_address_topic0 ON logs(address, topic0) WHERE topic0 IS NOT NULL;
CREATE INDEX idx_logs_address ON logs(address);
