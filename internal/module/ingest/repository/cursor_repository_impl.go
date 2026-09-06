package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

type CursorRepositoryImpl struct {
	pool *pgxpool.Pool
}

func NewCursorRepository(pool *pgxpool.Pool) CursorRepository {
	return &CursorRepositoryImpl{pool: pool}
}

func (r *CursorRepositoryImpl) Get(ctx context.Context, network string) (model.IngestState, error) {
	const query = `
		SELECT id, network, last_ledger, COALESCE(last_rpc_cursor, ''), updated_at
		FROM ingest_state
		WHERE network = $1
	`

	var state model.IngestState
	err := r.pool.QueryRow(ctx, query, network).Scan(
		&state.ID,
		&state.Network,
		&state.LastLedger,
		&state.LastRPCCursor,
		&state.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.IngestState{Network: network}, nil
	}
	if err != nil {
		return model.IngestState{}, fmt.Errorf("get ingest state: %w", err)
	}

	return state, nil
}

func (r *CursorRepositoryImpl) Upsert(ctx context.Context, network string, lastLedger uint32, rpcCursor string) error {
	const query = `
		INSERT INTO ingest_state (network, last_ledger, last_rpc_cursor, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (network)
		DO UPDATE SET
			last_ledger = GREATEST(ingest_state.last_ledger, EXCLUDED.last_ledger),
			last_rpc_cursor = EXCLUDED.last_rpc_cursor,
			updated_at = NOW()
	`

	if _, err := r.pool.Exec(ctx, query, network, lastLedger, rpcCursor); err != nil {
		return fmt.Errorf("upsert ingest state: %w", err)
	}

	return nil
}
