CREATE TABLE IF NOT EXISTS event_addresses (
    network LowCardinality(String),
    address String,
    contract_id String,
    event_id String,
    ledger UInt32,
    role LowCardinality(String),
    ingested_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (network, address, contract_id, event_id, role);

CREATE TABLE IF NOT EXISTS token_events (
    network LowCardinality(String),
    contract_id String,
    event_id String,
    ledger UInt32,
    token_symbol LowCardinality(String),
    token_amount String,
    from_address String,
    to_address String,
    action LowCardinality(String),
    ingested_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (network, contract_id, action, ledger, event_id);

ALTER TABLE events ADD INDEX IF NOT EXISTS idx_txn_hash txn_hash TYPE bloom_filter GRANULARITY 4;
ALTER TABLE events ADD INDEX IF NOT EXISTS idx_ledger ledger TYPE minmax GRANULARITY 1;
ALTER TABLE events ADD INDEX IF NOT EXISTS idx_event_type event_type TYPE set(0) GRANULARITY 4;

ALTER TABLE event_addresses ADD INDEX IF NOT EXISTS idx_address address TYPE bloom_filter GRANULARITY 4;
ALTER TABLE token_events ADD INDEX IF NOT EXISTS idx_from from_address TYPE bloom_filter GRANULARITY 4;
ALTER TABLE token_events ADD INDEX IF NOT EXISTS idx_to to_address TYPE bloom_filter GRANULARITY 4;
