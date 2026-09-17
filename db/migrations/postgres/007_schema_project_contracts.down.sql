DROP TABLE IF EXISTS contract_authorities;
DROP TABLE IF EXISTS verify_challenges;

ALTER TABLE schema_projects
    DROP COLUMN IF EXISTS contract_id,
    DROP COLUMN IF EXISTS network;
