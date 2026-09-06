CREATE TABLE IF NOT EXISTS ingest_state (
    id SERIAL PRIMARY KEY,
    network TEXT NOT NULL UNIQUE,
    last_ledger BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO ingest_state (network, last_ledger)
VALUES ('testnet', 0)
ON CONFLICT (network) DO NOTHING;

INSERT INTO ingest_state (network, last_ledger)
VALUES ('mainnet', 0)
ON CONFLICT (network) DO NOTHING;
