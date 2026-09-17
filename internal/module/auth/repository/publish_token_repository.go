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

var ErrPublishTokenNotFound = errors.New("publish token not found")

type PublishTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPublishTokenRepository(pool *pgxpool.Pool) *PublishTokenRepository {
	return &PublishTokenRepository{pool: pool}
}

func (r *PublishTokenRepository) CountActiveByUser(ctx context.Context, userID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM publish_tokens
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	var count int
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count publish tokens: %w", err)
	}
	return count, nil
}

func (r *PublishTokenRepository) Insert(
	ctx context.Context,
	userID, tokenHash, tokenPrefix, label string,
) (model.PublishToken, error) {
	const query = `
		INSERT INTO publish_tokens (user_id, token_hash, token_prefix, label)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, token_prefix, label, last_used_at, revoked_at, created_at
	`

	var token model.PublishToken
	err := r.pool.QueryRow(ctx, query, userID, tokenHash, tokenPrefix, label).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenPrefix,
		&token.Label,
		&token.LastUsedAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		return model.PublishToken{}, fmt.Errorf("insert publish token: %w", err)
	}
	return token, nil
}

func (r *PublishTokenRepository) ListByUser(ctx context.Context, userID string) ([]model.PublishToken, error) {
	const query = `
		SELECT id, user_id, token_prefix, label, last_used_at, revoked_at, created_at
		FROM publish_tokens
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list publish tokens: %w", err)
	}
	defer rows.Close()

	return scanPublishTokens(rows)
}

func (r *PublishTokenRepository) Revoke(ctx context.Context, userID, tokenID string, revokedAt time.Time) error {
	const query = `
		UPDATE publish_tokens
		SET revoked_at = $3
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`

	tag, err := r.pool.Exec(ctx, query, tokenID, userID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke publish token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPublishTokenNotFound
	}
	return nil
}

func (r *PublishTokenRepository) ResolveActive(
	ctx context.Context,
	tokenHash string,
) (model.PublishToken, error) {
	const query = `
		SELECT id, user_id, token_prefix, label, last_used_at, revoked_at, created_at
		FROM publish_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	var token model.PublishToken
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenPrefix,
		&token.Label,
		&token.LastUsedAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.PublishToken{}, ErrPublishTokenNotFound
	}
	if err != nil {
		return model.PublishToken{}, fmt.Errorf("resolve publish token: %w", err)
	}
	return token, nil
}

func (r *PublishTokenRepository) TouchLastUsed(ctx context.Context, tokenID string, usedAt time.Time) error {
	const query = `
		UPDATE publish_tokens
		SET last_used_at = $2
		WHERE id = $1
	`

	if _, err := r.pool.Exec(ctx, query, tokenID, usedAt); err != nil {
		return fmt.Errorf("touch publish token: %w", err)
	}
	return nil
}

func scanPublishTokens(rows pgx.Rows) ([]model.PublishToken, error) {
	var items []model.PublishToken
	for rows.Next() {
		var token model.PublishToken
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.TokenPrefix,
			&token.Label,
			&token.LastUsedAt,
			&token.RevokedAt,
			&token.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan publish token: %w", err)
		}
		items = append(items, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate publish tokens: %w", err)
	}
	return items, nil
}

func NormalizePublishTokenLabel(label string) string {
	return strings.TrimSpace(label)
}
