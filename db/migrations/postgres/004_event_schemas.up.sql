CREATE TABLE IF NOT EXISTS event_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id TEXT NOT NULL,
    network TEXT NOT NULL,
    event_name TEXT NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    schema_body JSONB NOT NULL,
    author TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT event_schemas_version_unique UNIQUE (contract_id, network, event_name, version),
    CONSTRAINT event_schemas_network_check CHECK (network IN ('testnet', 'mainnet', 'futurenet'))
);

CREATE INDEX IF NOT EXISTS event_schemas_contract_network_idx
    ON event_schemas (contract_id, network);

CREATE INDEX IF NOT EXISTS event_schemas_contract_network_event_idx
    ON event_schemas (contract_id, network, event_name, version DESC);

CREATE INDEX IF NOT EXISTS event_schemas_contract_id_search_idx
    ON event_schemas (contract_id text_pattern_ops);
