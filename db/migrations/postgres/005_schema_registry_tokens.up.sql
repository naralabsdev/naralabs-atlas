ALTER TABLE event_schemas
    ADD COLUMN IF NOT EXISTS publisher_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS trust_tier TEXT NOT NULL DEFAULT 'community',
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published',
    ADD COLUMN IF NOT EXISTS verified_wallet TEXT,
    ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;

ALTER TABLE event_schemas DROP CONSTRAINT IF EXISTS event_schemas_version_unique;

ALTER TABLE event_schemas DROP CONSTRAINT IF EXISTS event_schemas_trust_tier_check;
ALTER TABLE event_schemas
    ADD CONSTRAINT event_schemas_trust_tier_check CHECK (trust_tier IN ('community', 'verified'));

ALTER TABLE event_schemas DROP CONSTRAINT IF EXISTS event_schemas_status_check;
ALTER TABLE event_schemas
    ADD CONSTRAINT event_schemas_status_check CHECK (status IN ('draft', 'published', 'archived'));

CREATE UNIQUE INDEX IF NOT EXISTS event_schemas_publisher_version_unique
    ON event_schemas (publisher_user_id, contract_id, network, event_name, version)
    WHERE publisher_user_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS event_schemas_verified_unique
    ON event_schemas (contract_id, network, event_name)
    WHERE trust_tier = 'verified' AND status = 'published';

CREATE INDEX IF NOT EXISTS event_schemas_publisher_user_idx
    ON event_schemas (publisher_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS publish_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS publish_tokens_user_idx ON publish_tokens (user_id, created_at DESC);
