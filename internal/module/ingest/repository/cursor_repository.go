package repository

import (
	"context"

	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

type CursorRepository interface {
	Get(ctx context.Context, network string) (model.IngestState, error)
	Upsert(ctx context.Context, network string, lastLedger uint32, rpcCursor string) error
}

type EventRepository interface {
	UpsertBatch(ctx context.Context, events []model.ContractEvent) error
	ListByLedgerRange(ctx context.Context, network string, fromLedger, toLedger uint32, limit int) ([]model.ContractEvent, error)
}

type DerivedRepository interface {
	UpsertAddresses(ctx context.Context, rows []model.EventAddress) error
	UpsertTokenEvents(ctx context.Context, rows []model.TokenEvent) error
}

type BackfillRepository interface {
	Get(ctx context.Context, network string) (model.BackfillState, error)
	Start(ctx context.Context, network string, fromLedger, toLedger uint32) (model.BackfillState, error)
	UpdateProgress(ctx context.Context, network string, nextLedger uint32) error
	Complete(ctx context.Context, network string) error
}
