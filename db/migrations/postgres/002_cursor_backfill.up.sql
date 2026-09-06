ALTER TABLE ingest_state ADD COLUMN IF NOT EXISTS last_rpc_cursor TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS backfill_state (
    id SERIAL PRIMARY KEY,
    network TEXT NOT NULL UNIQUE,
    from_ledger BIGINT NOT NULL,
    to_ledger BIGINT NOT NULL,
    next_ledger BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
