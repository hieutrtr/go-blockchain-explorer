-- Create blocks table
CREATE TABLE blocks (
    height BIGINT PRIMARY KEY,
    hash BYTEA NOT NULL UNIQUE,
    parent_hash BYTEA NOT NULL,
    miner BYTEA NOT NULL,
    gas_used NUMERIC NOT NULL,
    gas_limit NUMERIC NOT NULL,
    timestamp BIGINT NOT NULL,
    tx_count INTEGER NOT NULL,
    orphaned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create transactions table
CREATE TABLE transactions (
    hash BYTEA PRIMARY KEY,
    block_height BIGINT NOT NULL REFERENCES blocks(height) ON DELETE CASCADE,
    tx_index INTEGER NOT NULL,
    from_addr BYTEA NOT NULL,
    to_addr BYTEA,  -- NULL for contract creation
    value_wei NUMERIC NOT NULL,
    fee_wei NUMERIC NOT NULL,
    gas_used NUMERIC NOT NULL,
    gas_price NUMERIC NOT NULL,
    nonce BIGINT NOT NULL,
    success BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create logs table
CREATE TABLE logs (
    id BIGSERIAL PRIMARY KEY,
    tx_hash BYTEA NOT NULL REFERENCES transactions(hash) ON DELETE CASCADE,
    log_index INTEGER NOT NULL,
    address BYTEA NOT NULL,
    topic0 BYTEA,
    topic1 BYTEA,
    topic2 BYTEA,
    topic3 BYTEA,
    data BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tx_hash, log_index)
);
