package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

var ErrSchemaNotFound = errors.New("schema not found")

type SchemaRepository struct {
	pool *pgxpool.Pool
}

func NewSchemaRepository(pool *pgxpool.Pool) *SchemaRepository {
	return &SchemaRepository{pool: pool}
}

func (r *SchemaRepository) NextVersion(
	ctx context.Context,
	contractID, network, eventName string,
) (int, error) {
	const query = `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM event_schemas
		WHERE contract_id = $1 AND network = $2 AND event_name = $3
	`

	var version int
	if err := r.pool.QueryRow(ctx, query, contractID, network, eventName).Scan(&version); err != nil {
		return 0, fmt.Errorf("next schema version: %w", err)
	}
	return version, nil
}

func (r *SchemaRepository) Insert(
	ctx context.Context,
	contractID, network, eventName string,
	version int,
	schemaBody []byte,
	author *string,
) (model.EventSchema, error) {
	const query = `
		INSERT INTO event_schemas (contract_id, network, event_name, version, schema_body, author)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, contract_id, network, event_name, version, schema_body, author, created_at, updated_at
	`

	var schema model.EventSchema
	err := r.pool.QueryRow(
		ctx,
		query,
		contractID,
		network,
		eventName,
		version,
		schemaBody,
		author,
	).Scan(
		&schema.ID,
		&schema.ContractID,
		&schema.Network,
		&schema.EventName,
		&schema.Version,
		&schema.SchemaBody,
		&schema.Author,
		&schema.CreatedAt,
		&schema.UpdatedAt,
	)
	if err != nil {
		return model.EventSchema{}, fmt.Errorf("insert schema: %w", err)
	}
	return schema, nil
}

func (r *SchemaRepository) List(
	ctx context.Context,
	network, search string,
	limit, offset int,
) ([]model.EventSchema, int, error) {
	where := []string{"network = $1"}
	args := []any{network}
	argN := 2

	if search = strings.TrimSpace(search); search != "" {
		where = append(where, fmt.Sprintf("contract_id ILIKE $%d", argN))
		args = append(args, "%"+search+"%")
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM event_schemas WHERE %s`, whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count schemas: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, contract_id, network, event_name, version, schema_body, author, created_at, updated_at
		FROM event_schemas
		WHERE %s
		ORDER BY created_at DESC, version DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list schemas: %w", err)
	}
	defer rows.Close()

	items, err := scanSchemas(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *SchemaRepository) ListByContract(
	ctx context.Context,
	network, contractID string,
) ([]model.EventSchema, error) {
	const query = `
		SELECT DISTINCT ON (event_name)
			id, contract_id, network, event_name, version, schema_body, author, created_at, updated_at
		FROM event_schemas
		WHERE network = $1 AND contract_id = $2
		ORDER BY event_name, version DESC
	`

	rows, err := r.pool.Query(ctx, query, network, contractID)
	if err != nil {
		return nil, fmt.Errorf("list schemas by contract: %w", err)
	}
	defer rows.Close()

	return scanSchemas(rows)
}

func (r *SchemaRepository) GetEventSchema(
	ctx context.Context,
	network, contractID, eventName string,
	version int,
) (model.EventSchema, error) {
	var query string
	var args []any

	if version > 0 {
		query = `
			SELECT id, contract_id, network, event_name, version, schema_body, author, created_at, updated_at
			FROM event_schemas
			WHERE network = $1 AND contract_id = $2 AND event_name = $3 AND version = $4
		`
		args = []any{network, contractID, eventName, version}
	} else {
		query = `
			SELECT id, contract_id, network, event_name, version, schema_body, author, created_at, updated_at
			FROM event_schemas
			WHERE network = $1 AND contract_id = $2 AND event_name = $3
			ORDER BY version DESC
			LIMIT 1
		`
		args = []any{network, contractID, eventName}
	}

	var schema model.EventSchema
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&schema.ID,
		&schema.ContractID,
		&schema.Network,
		&schema.EventName,
		&schema.Version,
		&schema.SchemaBody,
		&schema.Author,
		&schema.CreatedAt,
		&schema.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.EventSchema{}, ErrSchemaNotFound
	}
	if err != nil {
		return model.EventSchema{}, fmt.Errorf("get event schema: %w", err)
	}
	return schema, nil
}

func scanSchemas(rows pgx.Rows) ([]model.EventSchema, error) {
	var items []model.EventSchema
	for rows.Next() {
		var schema model.EventSchema
		if err := rows.Scan(
			&schema.ID,
			&schema.ContractID,
			&schema.Network,
			&schema.EventName,
			&schema.Version,
			&schema.SchemaBody,
			&schema.Author,
			&schema.CreatedAt,
			&schema.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schema: %w", err)
		}
		items = append(items, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schemas: %w", err)
	}
	return items, nil
}
