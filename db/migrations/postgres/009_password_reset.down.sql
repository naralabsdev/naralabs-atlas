DROP INDEX IF EXISTS password_reset_tokens_active_user_idx;
DROP INDEX IF EXISTS password_reset_tokens_token_hash_idx;
DROP INDEX IF EXISTS password_reset_tokens_user_id_idx;
DROP TABLE IF EXISTS password_reset_tokens;
