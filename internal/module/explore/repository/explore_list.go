package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
)

func (r *ExploreRepositoryImpl) ListEvents(
	ctx context.Context,
	network string,
	page, pageSize int,
	search, eventType, decodeStatusFilter string,
) (model.PaginatedListResponse[model.EventItem], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	whereParts := []string{"network = ?"}
	args := []any{network}

	search = strings.TrimSpace(search)
	if search != "" {
		whereParts = append(whereParts,
			"(positionCaseInsensitive(id, ?) > 0 OR positionCaseInsensitive(contract_id, ?) > 0 OR positionCaseInsensitive(txn_hash, ?) > 0 OR positionCaseInsensitive(topics_json, ?) > 0 OR positionCaseInsensitive(value_json, ?) > 0)",
		)
		args = append(args, search, search, search, search, search)
	}

	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if eventType != "" {
		whereParts = append(whereParts, "positionCaseInsensitive(topics_json, ?) > 0")
		args = append(args, fmt.Sprintf(`"symbol":"%s"`, eventType))
	}

	switch strings.ToLower(strings.TrimSpace(decodeStatusFilter)) {
	case "decoded":
		whereParts = append(whereParts, "semantic_decoded > 0")
	case "raw":
		whereParts = append(whereParts, "semantic_decoded = 0")
	}

	whereClause := strings.Join(whereParts, " AND ")

	countRow := r.ch.QueryRow(ctx, fmt.Sprintf(`
		SELECT count()
		FROM events
		WHERE %s
	`, whereClause), args...)
	var total uint64
	if err := countRow.Scan(&total); err != nil {
		return model.PaginatedListResponse[model.EventItem]{}, fmt.Errorf("count events: %w", err)
	}

	offset := (page - 1) * pageSize
	dataArgs := append(append([]any{}, args...), pageSize, offset)
	rows, err := r.ch.Query(ctx, fmt.Sprintf(`
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
		WHERE %s
		ORDER BY ingested_at DESC, ledger DESC, id DESC
		LIMIT ? OFFSET ?
	`, whereClause), dataArgs...)
	if err != nil {
		return model.PaginatedListResponse[model.EventItem]{}, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	items := make([]model.EventItem, 0, pageSize)
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
			return model.PaginatedListResponse[model.EventItem]{}, fmt.Errorf("scan event: %w", err)
		}
		item.EventType = eventTypeFromTopics(topicsJSON)
		item.SummaryPreview = summaryPreview(item.EventType, topicsJSON, valueJSON)
		item.DecodeStatus = decodeStatus(semanticDecoded)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return model.PaginatedListResponse[model.EventItem]{}, fmt.Errorf("iterate events: %w", err)
	}

	return model.PaginatedListResponse[model.EventItem]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (r *ExploreRepositoryImpl) ListContracts(
	ctx context.Context,
	network string,
	page, pageSize int,
	search, schemaStatusFilter string,
) (model.PaginatedListResponse[model.ContractItem], error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	whereParts := []string{"network = ?"}
	args := []any{network}

	search = strings.TrimSpace(search)
	if search != "" {
		whereParts = append(whereParts, "positionCaseInsensitive(contract_id, ?) > 0")
		args = append(args, search)
	}

	whereClause := strings.Join(whereParts, " AND ")

	havingParts := make([]string, 0, 1)
	switch strings.ToLower(strings.TrimSpace(schemaStatusFilter)) {
	case "decoded":
		havingParts = append(havingParts, "max(semantic_decoded) > 0")
	case "raw_only":
		havingParts = append(havingParts, "max(semantic_decoded) = 0")
	}
	havingClause := ""
	if len(havingParts) > 0 {
		havingClause = " HAVING " + strings.Join(havingParts, " AND ")
	}

	countRow := r.ch.QueryRow(ctx, fmt.Sprintf(`
		SELECT uniqExact(contract_id)
		FROM events
		WHERE %s
	`, whereClause), args...)
	var total uint64
	if err := countRow.Scan(&total); err != nil {
		return model.PaginatedListResponse[model.ContractItem]{}, fmt.Errorf("count contracts: %w", err)
	}

	if schemaStatusFilter != "" {
		countFilteredRow := r.ch.QueryRow(ctx, fmt.Sprintf(`
			SELECT count()
			FROM (
				SELECT contract_id
				FROM events
				WHERE %s
				GROUP BY contract_id
				%s
			)
		`, whereClause, havingClause), args...)
		if err := countFilteredRow.Scan(&total); err != nil {
			return model.PaginatedListResponse[model.ContractItem]{}, fmt.Errorf("count filtered contracts: %w", err)
		}
	}

	offset := (page - 1) * pageSize
	dataArgs := append(append([]any{}, args...), pageSize, offset)
	rows, err := r.ch.Query(ctx, fmt.Sprintf(`
		SELECT
			contract_id,
			count() AS event_count,
			min(ledger) AS first_ledger,
			max(ledger) AS last_ledger,
			max(ingested_at) AS last_seen,
			max(semantic_decoded) AS any_decoded
		FROM events
		WHERE %s
		GROUP BY contract_id
		%s
		ORDER BY last_seen DESC, event_count DESC
		LIMIT ? OFFSET ?
	`, whereClause, havingClause), dataArgs...)
	if err != nil {
		return model.PaginatedListResponse[model.ContractItem]{}, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	items := make([]model.ContractItem, 0, pageSize)
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
			return model.PaginatedListResponse[model.ContractItem]{}, fmt.Errorf("scan contract: %w", err)
		}
		if anyDecoded > 0 {
			item.SchemaStatus = "partial"
		} else {
			item.SchemaStatus = "raw_only"
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return model.PaginatedListResponse[model.ContractItem]{}, fmt.Errorf("iterate contracts: %w", err)
	}

	return model.PaginatedListResponse[model.ContractItem]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
