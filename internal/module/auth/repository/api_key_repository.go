package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/internal/module/auth/model"
)

var ErrAPIKeyNotFound = errors.New("api key not found")

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) CountActiveByUser(ctx context.Context, userID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM api_keys
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	var count int
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count api keys: %w", err)
	}
	return count, nil
}

func (r *APIKeyRepository) Insert(
	ctx context.Context,
	userID, tokenHash, tokenPrefix, label string,
) (model.APIKey, error) {
	const query = `
		INSERT INTO api_keys (user_id, token_hash, token_prefix, label)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, token_prefix, label, last_used_at, revoked_at, created_at
	`

	var key model.APIKey
	err := r.pool.QueryRow(ctx, query, userID, tokenHash, tokenPrefix, label).Scan(
		&key.ID,
		&key.UserID,
		&key.TokenPrefix,
		&key.Label,
		&key.LastUsedAt,
		&key.RevokedAt,
		&key.CreatedAt,
	)
	if err != nil {
		return model.APIKey{}, fmt.Errorf("insert api key: %w", err)
	}
	return key, nil
}

func (r *APIKeyRepository) ListByUser(ctx context.Context, userID string) ([]model.APIKey, error) {
	const query = `
		SELECT id, user_id, token_prefix, label, last_used_at, revoked_at, created_at
		FROM api_keys
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	return scanAPIKeys(rows)
}

func (r *APIKeyRepository) Revoke(ctx context.Context, userID, keyID string, revokedAt time.Time) error {
	const query = `
		UPDATE api_keys
		SET revoked_at = $3
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`

	tag, err := r.pool.Exec(ctx, query, keyID, userID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

func (r *APIKeyRepository) ResolveActive(ctx context.Context, tokenHash string) (model.APIKey, error) {
	const query = `
		SELECT id, user_id, token_prefix, label, last_used_at, revoked_at, created_at
		FROM api_keys
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	var key model.APIKey
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&key.ID,
		&key.UserID,
		&key.TokenPrefix,
		&key.Label,
		&key.LastUsedAt,
		&key.RevokedAt,
		&key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.APIKey{}, ErrAPIKeyNotFound
	}
	if err != nil {
		return model.APIKey{}, fmt.Errorf("resolve api key: %w", err)
	}
	return key, nil
}

func (r *APIKeyRepository) TouchLastUsed(ctx context.Context, keyID string, usedAt time.Time) error {
	const query = `
		UPDATE api_keys
		SET last_used_at = $2
		WHERE id = $1
	`

	if _, err := r.pool.Exec(ctx, query, keyID, usedAt); err != nil {
		return fmt.Errorf("touch api key: %w", err)
	}
	return nil
}

func scanAPIKeys(rows pgx.Rows) ([]model.APIKey, error) {
	var items []model.APIKey
	for rows.Next() {
		var key model.APIKey
		if err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.TokenPrefix,
			&key.Label,
			&key.LastUsedAt,
			&key.RevokedAt,
			&key.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		items = append(items, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return items, nil
}

func NormalizeAPIKeyLabel(label string) string {
	return strings.TrimSpace(label)
}
