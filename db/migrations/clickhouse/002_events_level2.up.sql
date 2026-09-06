ALTER TABLE events
    ADD COLUMN IF NOT EXISTS topics_json String AFTER value_xdr,
    ADD COLUMN IF NOT EXISTS value_json String AFTER topics_json,
    ADD COLUMN IF NOT EXISTS semantic_decoded UInt8 DEFAULT 0 AFTER value_json;
