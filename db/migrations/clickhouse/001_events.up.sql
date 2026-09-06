CREATE TABLE IF NOT EXISTS events (
    id String,
    network LowCardinality(String),
    contract_id String,
    ledger UInt32,
    txn_hash String,
    event_type UInt32,
    topics_xdr Array(String),
    value_xdr String,
    topics_json String,
    value_json String,
    semantic_decoded UInt8 DEFAULT 0,
    ingested_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(ingested_at)
ORDER BY (network, contract_id, ledger, id);
