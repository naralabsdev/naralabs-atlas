package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

type BackfillRepositoryImpl struct {
	pool *pgxpool.Pool
}

func NewBackfillRepository(pool *pgxpool.Pool) BackfillRepository {
	return &BackfillRepositoryImpl{pool: pool}
}

func (r *BackfillRepositoryImpl) Get(ctx context.Context, network string) (model.BackfillState, error) {
	const query = `
		SELECT id, network, from_ledger, to_ledger, next_ledger, status, started_at, updated_at, completed_at
		FROM backfill_state
		WHERE network = $1
	`
	var state model.BackfillState
	err := r.pool.QueryRow(ctx, query, network).Scan(
		&state.ID,
		&state.Network,
		&state.FromLedger,
		&state.ToLedger,
		&state.NextLedger,
		&state.Status,
		&state.StartedAt,
		&state.UpdatedAt,
		&state.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.BackfillState{Network: network}, nil
	}
	if err != nil {
		return model.BackfillState{}, fmt.Errorf("get backfill state: %w", err)
	}
	return state, nil
}

func (r *BackfillRepositoryImpl) Start(ctx context.Context, network string, fromLedger, toLedger uint32) (model.BackfillState, error) {
	const query = `
		INSERT INTO backfill_state (network, from_ledger, to_ledger, next_ledger, status, started_at, updated_at)
		VALUES ($1, $2, $3, $2, 'running', NOW(), NOW())
		ON CONFLICT (network)
		DO UPDATE SET
			from_ledger = EXCLUDED.from_ledger,
			to_ledger = EXCLUDED.to_ledger,
			next_ledger = EXCLUDED.next_ledger,
			status = 'running',
			started_at = NOW(),
			updated_at = NOW(),
			completed_at = NULL
		RETURNING id, network, from_ledger, to_ledger, next_ledger, status, started_at, updated_at, completed_at
	`
	var state model.BackfillState
	err := r.pool.QueryRow(ctx, query, network, fromLedger, toLedger).Scan(
		&state.ID,
		&state.Network,
		&state.FromLedger,
		&state.ToLedger,
		&state.NextLedger,
		&state.Status,
		&state.StartedAt,
		&state.UpdatedAt,
		&state.CompletedAt,
	)
	if err != nil {
		return model.BackfillState{}, fmt.Errorf("start backfill: %w", err)
	}
	return state, nil
}

func (r *BackfillRepositoryImpl) UpdateProgress(ctx context.Context, network string, nextLedger uint32) error {
	const query = `
		UPDATE backfill_state
		SET next_ledger = $2, updated_at = NOW()
		WHERE network = $1 AND status = 'running'
	`
	tag, err := r.pool.Exec(ctx, query, network, nextLedger)
	if err != nil {
		return fmt.Errorf("update backfill progress: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("backfill not running for network %s", network)
	}
	return nil
}

func (r *BackfillRepositoryImpl) Complete(ctx context.Context, network string) error {
	const query = `
		UPDATE backfill_state
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE network = $1
	`
	if _, err := r.pool.Exec(ctx, query, network); err != nil {
		return fmt.Errorf("complete backfill: %w", err)
	}
	return nil
}
