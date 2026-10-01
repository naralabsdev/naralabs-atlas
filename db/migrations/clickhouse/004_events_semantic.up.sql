ALTER TABLE events
    ADD COLUMN IF NOT EXISTS decoded_event_name LowCardinality(String) DEFAULT '' AFTER semantic_decoded,
    ADD COLUMN IF NOT EXISTS schema_version UInt16 DEFAULT 0 AFTER decoded_event_name,
    ADD COLUMN IF NOT EXISTS decode_summary String DEFAULT '' AFTER schema_version,
    ADD COLUMN IF NOT EXISTS decoded_fields_json String DEFAULT '' AFTER decode_summary;
