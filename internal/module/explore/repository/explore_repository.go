package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
)

var ErrEventNotFound = errors.New("event not found")
var ErrContractNotFound = errors.New("contract not found")

type ExploreRepository interface {
	GetStats(ctx context.Context, network string) (model.NetworkStats, error)
	ListRecentEvents(ctx context.Context, network string, limit int) ([]model.EventItem, error)
	ListActiveContracts(ctx context.Context, network string, limit int) ([]model.ContractItem, error)
	GetEventByID(ctx context.Context, network, id string) (model.EventDetail, error)
	GetContractByID(ctx context.Context, network, contractID string) (model.ContractDetail, error)
	ListContractEvents(
		ctx context.Context,
		network, contractID string,
		page, pageSize int,
		search, eventType, decodeStatus string,
	) (model.PaginatedListResponse[model.EventItem], error)
}

type ExploreRepositoryImpl struct {
	ch  clickhouse.Conn
	pg  *pgxpool.Pool
	now func() time.Time
}

func NewExploreRepository(ch clickhouse.Conn, pg *pgxpool.Pool) ExploreRepository {
	return &ExploreRepositoryImpl{ch: ch, pg: pg, now: time.Now}
}

func (r *ExploreRepositoryImpl) GetStats(ctx context.Context, network string) (model.NetworkStats, error) {
	var stats model.NetworkStats
	stats.Network = network

	row := r.ch.QueryRow(ctx, `
		SELECT
			count(),
			uniqExact(contract_id),
			min(ledger),
			max(ledger),
			max(ingested_at),
			countIf(ingested_at >= now() - INTERVAL 24 HOUR)
		FROM events
		WHERE network = ?
	`, network)
	var oldestLedger, newestLedger uint32
	if err := row.Scan(
		&stats.TotalEvents,
		&stats.ContractCount,
		&oldestLedger,
		&newestLedger,
		&stats.LastIndexedAt,
		&stats.Events24h,
	); err != nil {
		return model.NetworkStats{}, fmt.Errorf("aggregate stats: %w", err)
	}
	if oldestLedger > 0 {
		v := uint64(oldestLedger)
		stats.OldestStoredLedger = &v
	}
	stats.LastIngestedLedger = uint64(newestLedger)

	const pgQuery = `
		SELECT last_ledger, updated_at
		FROM ingest_state
		WHERE network = $1
	`
	var pgLedger uint64
	var updatedAt time.Time
	err := r.pg.QueryRow(ctx, pgQuery, network).Scan(&pgLedger, &updatedAt)
	if err == nil {
		stats.LastIngestedLedger = pgLedger
		stats.LastIndexedAt = &updatedAt
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return model.NetworkStats{}, fmt.Errorf("load ingest cursor: %w", err)
	}

	rows, err := r.ch.Query(ctx, `
		SELECT toStartOfHour(ingested_at) AS bucket, count() AS c
		FROM events
		WHERE network = ?
		  AND ingested_at >= now() - INTERVAL 14 DAY
		GROUP BY bucket
		ORDER BY bucket ASC
	`, network)
	if err != nil {
		return model.NetworkStats{}, fmt.Errorf("activity buckets: %w", err)
	}
	defer rows.Close()

	stats.Activity = make([]model.ActivityBucket, 0, 336)
	for rows.Next() {
		var bucket model.ActivityBucket
		if err := rows.Scan(&bucket.Bucket, &bucket.Count); err != nil {
			return model.NetworkStats{}, fmt.Errorf("scan activity bucket: %w", err)
		}
		stats.Activity = append(stats.Activity, bucket)
	}
	if err := rows.Err(); err != nil {
		return model.NetworkStats{}, fmt.Errorf("iterate activity: %w", err)
	}

	return stats, nil
}

func (r *ExploreRepositoryImpl) ListRecentEvents(ctx context.Context, network string, limit int) ([]model.EventItem, error) {
	if limit <= 0 {
		limit = 8
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.ch.Query(ctx, `
		SELECT
			id,
			contract_id,
			ledger,
			txn_hash,
			topics_json,
			value_json,
			semantic_decoded,
			ingested_at
		FROM events
		WHERE network = ?
		ORDER BY ingested_at DESC, ledger DESC, id DESC
		LIMIT ?
	`, network, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent events: %w", err)
	}
	defer rows.Close()

	out := make([]model.EventItem, 0, limit)
	for rows.Next() {
		var item model.EventItem
		var topicsJSON, valueJSON string
		var semanticDecoded uint8
		if err := rows.Scan(
			&item.ID,
			&item.ContractID,
			&item.Ledger,
			&item.TxnHash,
			&topicsJSON,
			&valueJSON,
			&semanticDecoded,
			&item.IngestedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		item.EventType = eventTypeFromTopics(topicsJSON)
		item.SummaryPreview = summaryPreview(item.EventType, topicsJSON, valueJSON)
		item.DecodeStatus = decodeStatus(semanticDecoded)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return out, nil
}

func (r *ExploreRepositoryImpl) ListActiveContracts(ctx context.Context, network string, limit int) ([]model.ContractItem, error) {
	if limit <= 0 {
		limit = 8
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.ch.Query(ctx, `
		SELECT
			contract_id,
			count() AS event_count,
			min(ledger) AS first_ledger,
			max(ledger) AS last_ledger,
			max(ingested_at) AS last_seen,
			max(semantic_decoded) AS any_decoded
		FROM events
		WHERE network = ?
		GROUP BY contract_id
		ORDER BY last_seen DESC, event_count DESC
		LIMIT ?
	`, network, limit)
	if err != nil {
		return nil, fmt.Errorf("list active contracts: %w", err)
	}
	defer rows.Close()

	out := make([]model.ContractItem, 0, limit)
	for rows.Next() {
		var item model.ContractItem
		var anyDecoded uint8
		if err := rows.Scan(
			&item.ContractID,
			&item.EventCount,
			&item.FirstLedger,
			&item.LastLedger,
			&item.LastSeen,
			&anyDecoded,
		); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		if anyDecoded > 0 {
			item.SchemaStatus = "partial"
		} else {
			item.SchemaStatus = "raw_only"
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contracts: %w", err)
	}
	return out, nil
}

func (r *ExploreRepositoryImpl) GetEventByID(ctx context.Context, network, id string) (model.EventDetail, error) {
	row := r.ch.QueryRow(ctx, `
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
		WHERE network = ? AND id = ?
		ORDER BY ingested_at DESC
		LIMIT 1
	`, network, id)

	var detail model.EventDetail
	var topicsJSON, valueJSON string
	var semanticDecoded uint8
	if err := row.Scan(
		&detail.ID,
		&detail.Network,
		&detail.ContractID,
		&detail.Ledger,
		&detail.TxnHash,
		&detail.EventTypeCode,
		&detail.TopicsXDR,
		&detail.ValueXDR,
		&topicsJSON,
		&valueJSON,
		&semanticDecoded,
		&detail.IngestedAt,
	); err != nil {
		if isEmptyScan(err) {
			return model.EventDetail{}, ErrEventNotFound
		}
		return model.EventDetail{}, fmt.Errorf("get event: %w", err)
	}
	if detail.ID == "" {
		return model.EventDetail{}, ErrEventNotFound
	}

	detail.EventType = eventTypeFromTopics(topicsJSON)
	detail.EventKind = eventKindFromCode(detail.EventTypeCode)
	detail.SummaryPreview = summaryPreview(detail.EventType, topicsJSON, valueJSON)
	detail.DecodeStatus = decodeStatus(semanticDecoded)
	detail.Topics = normalizeJSONPayload(topicsJSON, "[]")
	detail.Value = normalizeJSONPayload(valueJSON, "{}")

	return detail, nil
}

func (r *ExploreRepositoryImpl) GetContractByID(
	ctx context.Context,
	network, contractID string,
) (model.ContractDetail, error) {
	row := r.ch.QueryRow(ctx, `
		SELECT
			count() AS event_count,
			uniqExact(txn_hash) AS transaction_count,
			countIf(semantic_decoded > 0) AS decoded_count,
			countIf(ingested_at >= now() - INTERVAL 24 HOUR) AS events_24h,
			min(ledger) AS first_ledger,
			max(ledger) AS last_ledger,
			max(ingested_at) AS last_seen,
			max(semantic_decoded) AS any_decoded
		FROM events
		WHERE network = ? AND contract_id = ?
	`, network, contractID)

	var detail model.ContractDetail
	var anyDecoded uint8
	if err := row.Scan(
		&detail.EventCount,
		&detail.TransactionCount,
		&detail.DecodedCount,
		&detail.Events24h,
		&detail.FirstLedger,
		&detail.LastLedger,
		&detail.LastSeen,
		&anyDecoded,
	); err != nil {
		return model.ContractDetail{}, fmt.Errorf("get contract stats: %w", err)
	}
	if detail.EventCount == 0 {
		return model.ContractDetail{}, ErrContractNotFound
	}

	detail.ContractID = contractID
	detail.Network = network
	if anyDecoded > 0 {
		detail.SchemaStatus = "partial"
	} else {
		detail.SchemaStatus = "raw_only"
	}

	breakdownRows, err := r.ch.Query(ctx, `
		SELECT topics_json, count() AS c
		FROM events
		WHERE network = ? AND contract_id = ?
		GROUP BY topics_json
		ORDER BY c DESC
		LIMIT 20
	`, network, contractID)
	if err != nil {
		return model.ContractDetail{}, fmt.Errorf("contract type breakdown: %w", err)
	}
	defer breakdownRows.Close()

	typeCounts := make(map[string]uint64)
	for breakdownRows.Next() {
		var topicsJSON string
		var count uint64
		if err := breakdownRows.Scan(&topicsJSON, &count); err != nil {
			return model.ContractDetail{}, fmt.Errorf("scan type breakdown: %w", err)
		}
		eventType := eventTypeFromTopics(topicsJSON)
		typeCounts[eventType] += count
	}
	if err := breakdownRows.Err(); err != nil {
		return model.ContractDetail{}, fmt.Errorf("iterate type breakdown: %w", err)
	}

	detail.TypeBreakdown = make([]model.EventTypeCount, 0, len(typeCounts))
	for eventType, count := range typeCounts {
		detail.TypeBreakdown = append(detail.TypeBreakdown, model.EventTypeCount{
			EventType: eventType,
			Count:     count,
		})
	}
	sortEventTypeCounts(detail.TypeBreakdown)

	return detail, nil
}

func sortEventTypeCounts(items []model.EventTypeCount) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].EventType < items[j].EventType
		}
		return items[i].Count > items[j].Count
	})
}

func normalizeJSONPayload(raw, fallback string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return json.RawMessage(fallback)
	}
	if json.Valid([]byte(raw)) {
		return json.RawMessage(raw)
	}
	return json.RawMessage(fallback)
}

func eventKindFromCode(code uint32) string {
	switch code {
	case 0:
		return "system"
	case 1:
		return "contract"
	case 2:
		return "diagnostic"
	default:
		return "unknown"
	}
}

func isEmptyScan(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || err.Error() == "sql: no rows in result set")
}
