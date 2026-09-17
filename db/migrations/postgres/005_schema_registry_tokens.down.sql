DROP TABLE IF EXISTS publish_tokens;

DROP INDEX IF EXISTS event_schemas_publisher_user_idx;
DROP INDEX IF EXISTS event_schemas_verified_unique;
DROP INDEX IF EXISTS event_schemas_publisher_version_unique;

ALTER TABLE event_schemas DROP CONSTRAINT IF EXISTS event_schemas_status_check;
ALTER TABLE event_schemas DROP CONSTRAINT IF EXISTS event_schemas_trust_tier_check;

ALTER TABLE event_schemas
    DROP COLUMN IF EXISTS verified_at,
    DROP COLUMN IF EXISTS verified_wallet,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS trust_tier,
    DROP COLUMN IF EXISTS publisher_user_id;

ALTER TABLE event_schemas
    ADD CONSTRAINT event_schemas_version_unique UNIQUE (contract_id, network, event_name, version);
