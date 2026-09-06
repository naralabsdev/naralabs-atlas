package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

type EventRepositoryImpl struct {
	conn clickhouse.Conn
}

func NewEventRepository(conn clickhouse.Conn) EventRepository {
	return &EventRepositoryImpl{conn: conn}
}

func (r *EventRepositoryImpl) UpsertBatch(ctx context.Context, events []model.ContractEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO events (
			id,
			network,
			contract_id,
			ledger,
			txn_hash,
			event_type,
			topics_xdr,
			value_xdr,
			topics_json,
			value_json,
			semantic_decoded,
			ingested_at
		)
	`)
	if err != nil {
		return fmt.Errorf("prepare clickhouse batch: %w", err)
	}

	now := time.Now().UTC()
	for _, event := range events {
		ingestedAt := event.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = now
		}

		semanticDecoded := uint8(0)
		if event.SemanticDecoded {
			semanticDecoded = 1
		}

		if err := batch.Append(
			event.ID,
			event.Network,
			event.ContractID,
			event.Ledger,
			event.TxnHash,
			event.EventType,
			event.TopicsXDR,
			event.ValueXDR,
			event.TopicsJSON,
			event.ValueJSON,
			semanticDecoded,
			ingestedAt,
		); err != nil {
			return fmt.Errorf("append event %s: %w", event.ID, err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send clickhouse batch: %w", err)
	}

	return nil
}

func (r *EventRepositoryImpl) ListByLedgerRange(
	ctx context.Context,
	network string,
	fromLedger, toLedger uint32,
	limit int,
) ([]model.ContractEvent, error) {
	if limit <= 0 {
		limit = 500
	}

	rows, err := r.conn.Query(ctx, `
		SELECT
			id,
			network,
			contract_id,
			ledger,
			txn_hash,
			event_type,
			topics_xdr,
			value_xdr,
			topics_json,
			value_json,
			semantic_decoded,
			ingested_at
		FROM events
		WHERE network = ?
		  AND ledger >= ?
		  AND ledger <= ?
		ORDER BY ledger ASC, id ASC
		LIMIT ?
	`, network, fromLedger, toLedger, limit)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	out := make([]model.ContractEvent, 0, limit)
	for rows.Next() {
		var event model.ContractEvent
		var semanticDecoded uint8
		if err := rows.Scan(
			&event.ID,
			&event.Network,
			&event.ContractID,
			&event.Ledger,
			&event.TxnHash,
			&event.EventType,
			&event.TopicsXDR,
			&event.ValueXDR,
			&event.TopicsJSON,
			&event.ValueJSON,
			&semanticDecoded,
			&event.IngestedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		event.SemanticDecoded = semanticDecoded == 1
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return out, nil
}

type DerivedRepositoryImpl struct {
	conn clickhouse.Conn
}

func NewDerivedRepository(conn clickhouse.Conn) DerivedRepository {
	return &DerivedRepositoryImpl{conn: conn}
}

func (r *DerivedRepositoryImpl) UpsertAddresses(ctx context.Context, rows []model.EventAddress) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO event_addresses (
			network, address, contract_id, event_id, ledger, role, ingested_at
		)
	`)
	if err != nil {
		return fmt.Errorf("prepare address batch: %w", err)
	}
	now := time.Now().UTC()
	for _, row := range rows {
		ingestedAt := row.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = now
		}
		if err := batch.Append(
			row.Network,
			row.Address,
			row.ContractID,
			row.EventID,
			row.Ledger,
			row.Role,
			ingestedAt,
		); err != nil {
			return fmt.Errorf("append address: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send address batch: %w", err)
	}
	return nil
}

func (r *DerivedRepositoryImpl) UpsertTokenEvents(ctx context.Context, rows []model.TokenEvent) error {
	if len(rows) == 0 {
		return nil
	}
	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO token_events (
			network, contract_id, event_id, ledger,
			token_symbol, token_amount, from_address, to_address, action, ingested_at
		)
	`)
	if err != nil {
		return fmt.Errorf("prepare token batch: %w", err)
	}
	now := time.Now().UTC()
	for _, row := range rows {
		ingestedAt := row.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = now
		}
		if err := batch.Append(
			row.Network,
			row.ContractID,
			row.EventID,
			row.Ledger,
			row.TokenSymbol,
			row.TokenAmount,
			row.FromAddress,
			row.ToAddress,
			row.Action,
			ingestedAt,
		); err != nil {
			return fmt.Errorf("append token event: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send token batch: %w", err)
	}
	return nil
}
