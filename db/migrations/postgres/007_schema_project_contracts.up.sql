ALTER TABLE schema_projects
    ADD COLUMN IF NOT EXISTS contract_id TEXT,
    ADD COLUMN IF NOT EXISTS network TEXT;

CREATE TABLE IF NOT EXISTS verify_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES schema_projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contract_id TEXT NOT NULL,
    network TEXT NOT NULL,
    nonce TEXT NOT NULL UNIQUE,
    message TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS verify_challenges_project_idx
    ON verify_challenges (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS contract_authorities (
    contract_id TEXT NOT NULL,
    network TEXT NOT NULL,
    deployer_wallet TEXT NOT NULL,
    resolved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contract_id, network)
);
