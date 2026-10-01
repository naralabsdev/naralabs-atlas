ALTER TABLE events
    DROP COLUMN IF EXISTS decoded_fields_json,
    DROP COLUMN IF EXISTS decode_summary,
    DROP COLUMN IF EXISTS schema_version,
    DROP COLUMN IF EXISTS decoded_event_name;
