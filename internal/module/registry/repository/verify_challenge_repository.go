package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrVerifyChallengeNotFound = errors.New("verify challenge not found")
	ErrVerifyChallengeExpired  = errors.New("verify challenge expired")
	ErrVerifyChallengeUsed     = errors.New("verify challenge used")
)

type VerifyChallengeRepository struct {
	pool *pgxpool.Pool
}

func NewVerifyChallengeRepository(pool *pgxpool.Pool) *VerifyChallengeRepository {
	return &VerifyChallengeRepository{pool: pool}
}

func (r *VerifyChallengeRepository) Create(
	ctx context.Context,
	projectID, userID, contractID, network, nonce, message string,
	expiresAt time.Time,
) error {
	const query = `
		INSERT INTO verify_challenges (
			project_id, user_id, contract_id, network, nonce, message, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query, projectID, userID, contractID, network, nonce, message, expiresAt)
	if err != nil {
		return fmt.Errorf("insert verify challenge: %w", err)
	}
	return nil
}

func (r *VerifyChallengeRepository) Consume(
	ctx context.Context,
	projectID, userID, contractID, network, nonce string,
	now time.Time,
) (string, error) {
	const query = `
		UPDATE verify_challenges
		SET used_at = $7
		WHERE project_id = $1
		  AND user_id = $2
		  AND contract_id = $3
		  AND network = $4
		  AND nonce = $5
		  AND used_at IS NULL
		  AND expires_at > $6
		RETURNING message
	`

	var message string
	err := r.pool.QueryRow(
		ctx,
		query,
		projectID,
		userID,
		contractID,
		network,
		nonce,
		now,
		now,
	).Scan(&message)
	if errors.Is(err, pgx.ErrNoRows) {
		const lookupQuery = `
			SELECT message, used_at, expires_at
			FROM verify_challenges
			WHERE project_id = $1
			  AND user_id = $2
			  AND contract_id = $3
			  AND network = $4
			  AND nonce = $5
		`
		var usedAt *time.Time
		var expiresAt time.Time
		lookupErr := r.pool.QueryRow(
			ctx,
			lookupQuery,
			projectID,
			userID,
			contractID,
			network,
			nonce,
		).Scan(&message, &usedAt, &expiresAt)
		if lookupErr != nil {
			if errors.Is(lookupErr, pgx.ErrNoRows) {
				return "", ErrVerifyChallengeNotFound
			}
			return "", fmt.Errorf("lookup verify challenge: %w", lookupErr)
		}
		if usedAt != nil {
			return "", ErrVerifyChallengeUsed
		}
		if !expiresAt.After(now) {
			return "", ErrVerifyChallengeExpired
		}
		return "", ErrVerifyChallengeNotFound
	}
	if err != nil {
		return "", fmt.Errorf("consume verify challenge: %w", err)
	}
	return message, nil
}
