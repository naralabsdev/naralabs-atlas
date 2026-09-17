CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS password_reset_tokens_user_id_idx
    ON password_reset_tokens (user_id);

CREATE INDEX IF NOT EXISTS password_reset_tokens_token_hash_idx
    ON password_reset_tokens (token_hash);

CREATE INDEX IF NOT EXISTS password_reset_tokens_active_user_idx
    ON password_reset_tokens (user_id, created_at DESC)
    WHERE consumed_at IS NULL;
