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

var ErrUserNotFound = errors.New("user not found")

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email, passwordHash string) (model.User, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, email_verified_at, created_at, updated_at
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, normalizeEmail(email), passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, email_verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, normalizeEmail(email)).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, id string) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, email_verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	const query = `
		UPDATE users
		SET email_verified_at = COALESCE(email_verified_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1
	`

	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *AuthRepository) InvalidateActiveVerificationTokens(ctx context.Context, userID string, now time.Time) error {
	const query = `
		UPDATE email_verification_tokens
		SET consumed_at = $2
		WHERE user_id = $1
		  AND consumed_at IS NULL
	`

	if _, err := r.pool.Exec(ctx, query, userID, now); err != nil {
		return fmt.Errorf("invalidate verification tokens: %w", err)
	}
	return nil
}

func (r *AuthRepository) CreateVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`

	if _, err := r.pool.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}
	return nil
}

func (r *AuthRepository) LatestActiveVerificationCreatedAt(ctx context.Context, userID string) (*time.Time, error) {
	const query = `
		SELECT created_at
		FROM email_verification_tokens
		WHERE user_id = $1
		  AND consumed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	var createdAt time.Time
	err := r.pool.QueryRow(ctx, query, userID).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest verification token: %w", err)
	}
	return &createdAt, nil
}

func (r *AuthRepository) ConsumeVerificationToken(ctx context.Context, tokenHash string, now time.Time) (model.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const selectQuery = `
		SELECT t.id, t.user_id, t.expires_at
		FROM email_verification_tokens t
		WHERE t.token_hash = $1
		  AND t.consumed_at IS NULL
		LIMIT 1
	`

	var tokenID, userID string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, selectQuery, tokenHash).Scan(&tokenID, &userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("select verification token: %w", err)
	}

	if now.After(expiresAt) {
		return model.User{}, fmt.Errorf("token expired")
	}

	const consumeQuery = `
		UPDATE email_verification_tokens
		SET consumed_at = $2
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, consumeQuery, tokenID, now); err != nil {
		return model.User{}, fmt.Errorf("consume verification token: %w", err)
	}

	const verifyQuery = `
		UPDATE users
		SET email_verified_at = COALESCE(email_verified_at, $2),
		    updated_at = $2
		WHERE id = $1
		RETURNING id, email, password_hash, email_verified_at, created_at, updated_at
	`

	var user model.User
	err = tx.QueryRow(ctx, verifyQuery, userID, now).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("verify user email: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.User{}, fmt.Errorf("commit tx: %w", err)
	}
	return user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
