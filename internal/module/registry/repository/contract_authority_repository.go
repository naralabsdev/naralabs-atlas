package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContractAuthorityRepository struct {
	pool *pgxpool.Pool
}

func NewContractAuthorityRepository(pool *pgxpool.Pool) *ContractAuthorityRepository {
	return &ContractAuthorityRepository{pool: pool}
}

func (r *ContractAuthorityRepository) Get(
	ctx context.Context,
	contractID, network string,
) (string, bool, error) {
	const query = `
		SELECT deployer_wallet
		FROM contract_authorities
		WHERE contract_id = $1 AND network = $2
	`

	var wallet string
	err := r.pool.QueryRow(ctx, query, contractID, network).Scan(&wallet)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get contract authority: %w", err)
	}
	return wallet, true, nil
}

func (r *ContractAuthorityRepository) Upsert(
	ctx context.Context,
	contractID, network, wallet string,
) error {
	const query = `
		INSERT INTO contract_authorities (contract_id, network, deployer_wallet, resolved_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (contract_id, network)
		DO UPDATE SET deployer_wallet = EXCLUDED.deployer_wallet, resolved_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query, contractID, network, wallet)
	if err != nil {
		return fmt.Errorf("upsert contract authority: %w", err)
	}
	return nil
}
