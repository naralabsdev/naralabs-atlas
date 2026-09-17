package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

var ErrSchemaNotFound = errors.New("schema not found")

const schemaSelectColumns = `
	id, contract_id, network, event_name, version, schema_body, author,
	publisher_user_id, trust_tier, status, verified_wallet, verified_at,
	created_at, updated_at
`

type SchemaRepository struct {
	pool *pgxpool.Pool
}

func NewSchemaRepository(pool *pgxpool.Pool) *SchemaRepository {
	return &SchemaRepository{pool: pool}
}

func (r *SchemaRepository) NextVersion(
	ctx context.Context,
	publisherUserID, contractID, network, eventName string,
) (int, error) {
	const query = `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM event_schemas
		WHERE publisher_user_id = $1
		  AND contract_id = $2
		  AND network = $3
		  AND event_name = $4
	`

	var version int
	if err := r.pool.QueryRow(ctx, query, publisherUserID, contractID, network, eventName).Scan(&version); err != nil {
		return 0, fmt.Errorf("next schema version: %w", err)
	}
	return version, nil
}

func (r *SchemaRepository) Insert(
	ctx context.Context,
	publisherUserID, contractID, network, eventName string,
	version int,
	schemaBody []byte,
	author *string,
) (model.EventSchema, error) {
	const query = `
		INSERT INTO event_schemas (
			contract_id, network, event_name, version, schema_body, author,
			publisher_user_id, trust_tier, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'community', 'published')
		RETURNING ` + schemaSelectColumns

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
		publisherUserID,
	).Scan(
		&schema.ID,
		&schema.ContractID,
		&schema.Network,
		&schema.EventName,
		&schema.Version,
		&schema.SchemaBody,
		&schema.Author,
		&schema.PublisherUserID,
		&schema.TrustTier,
		&schema.Status,
		&schema.VerifiedWallet,
		&schema.VerifiedAt,
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
	where := []string{"network = $1", "status = 'published'"}
	args := []any{network}
	argN := 2

	if search = strings.TrimSpace(search); search != "" {
		where = append(where, fmt.Sprintf(
			"(contract_id ILIKE $%d OR event_name ILIKE $%d OR id::text ILIKE $%d)",
			argN, argN, argN,
		))
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
		SELECT %s
		FROM event_schemas
		WHERE %s
		ORDER BY created_at DESC, version DESC
		LIMIT $%d OFFSET $%d
	`, schemaSelectColumns, whereSQL, argN, argN+1)
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

func (r *SchemaRepository) ListByPublisher(
	ctx context.Context,
	publisherUserID string,
	limit, offset int,
) ([]model.EventSchema, int, error) {
	const countQuery = `
		SELECT COUNT(*)
		FROM event_schemas
		WHERE publisher_user_id = $1 AND status = 'published'
	`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, publisherUserID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count publisher schemas: %w", err)
	}

	const listQuery = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE publisher_user_id = $1 AND status = 'published'
		ORDER BY created_at DESC, version DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, listQuery, publisherUserID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list publisher schemas: %w", err)
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
			` + schemaSelectColumns + `
		FROM event_schemas
		WHERE network = $1 AND contract_id = $2 AND status = 'published'
		ORDER BY event_name, trust_tier DESC, version DESC
	`

	rows, err := r.pool.Query(ctx, query, network, contractID)
	if err != nil {
		return nil, fmt.Errorf("list schemas by contract: %w", err)
	}
	defer rows.Close()

	return scanSchemas(rows)
}

func (r *SchemaRepository) ListVersionsByEvent(
	ctx context.Context,
	network, contractID, eventName string,
) ([]model.EventSchema, error) {
	const query = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE network = $1
		  AND contract_id = $2
		  AND event_name = $3
		  AND status = 'published'
		ORDER BY version DESC, created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, network, contractID, eventName)
	if err != nil {
		return nil, fmt.Errorf("list schema versions by event: %w", err)
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
			SELECT ` + schemaSelectColumns + `
			FROM event_schemas
			WHERE network = $1 AND contract_id = $2 AND event_name = $3 AND version = $4
			  AND status = 'published'
		`
		args = []any{network, contractID, eventName, version}
	} else {
		query = `
			SELECT ` + schemaSelectColumns + `
			FROM event_schemas
			WHERE network = $1 AND contract_id = $2 AND event_name = $3
			  AND status = 'published'
			ORDER BY trust_tier DESC, version DESC
			LIMIT 1
		`
		args = []any{network, contractID, eventName}
	}

	return r.scanOne(ctx, query, args...)
}

func (r *SchemaRepository) GetByIDForPublisher(
	ctx context.Context,
	schemaID, publisherUserID string,
) (model.EventSchema, error) {
	const query = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE id = $1 AND publisher_user_id = $2 AND status = 'published'
	`

	return r.scanOne(ctx, query, schemaID, publisherUserID)
}

func (r *SchemaRepository) GetPublishedByID(ctx context.Context, schemaID string) (model.EventSchema, error) {
	const query = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE id = $1 AND status = 'published'
	`

	return r.scanOne(ctx, query, schemaID)
}

func (r *SchemaRepository) ListPublishedForContract(
	ctx context.Context,
	network, contractID string,
) ([]model.EventSchema, error) {
	const query = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE network = $1
		  AND contract_id = $2
		  AND status = 'published'
		ORDER BY event_name ASC, trust_tier DESC, version DESC, created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, network, contractID)
	if err != nil {
		return nil, fmt.Errorf("list published schemas for contract: %w", err)
	}
	defer rows.Close()

	return scanSchemas(rows)
}

func (r *SchemaRepository) ListPublishedByNetwork(
	ctx context.Context,
	network, search string,
) ([]model.EventSchema, error) {
	where := []string{"network = $1", "status = 'published'"}
	args := []any{network}

	if search = strings.TrimSpace(search); search != "" {
		where = append(where, "contract_id ILIKE $2")
		args = append(args, "%"+search+"%")
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM event_schemas
		WHERE %s
		ORDER BY contract_id ASC, created_at ASC
	`, schemaSelectColumns, strings.Join(where, " AND "))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list published schemas by network: %w", err)
	}
	defer rows.Close()

	return scanSchemas(rows)
}

func (r *SchemaRepository) MarkVerified(
	ctx context.Context,
	schemaID, wallet string,
	verifiedAt time.Time,
) (model.EventSchema, error) {
	const query = `
		UPDATE event_schemas
		SET trust_tier = 'verified',
		    verified_wallet = $2,
		    verified_at = $3,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'published'
		RETURNING ` + schemaSelectColumns

	return r.scanOne(ctx, query, schemaID, wallet, verifiedAt)
}

func (r *SchemaRepository) ListByPublisherAndContract(
	ctx context.Context,
	publisherUserID, contractID, network string,
) ([]model.EventSchema, error) {
	const query = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE publisher_user_id = $1
		  AND contract_id = $2
		  AND network = $3
		  AND status = 'published'
		ORDER BY event_name ASC, version DESC, created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, publisherUserID, contractID, network)
	if err != nil {
		return nil, fmt.Errorf("list publisher contract schemas: %w", err)
	}
	defer rows.Close()

	return scanSchemas(rows)
}

func (r *SchemaRepository) ListPublishedByPublisher(
	ctx context.Context,
	publisherUserID string,
) ([]model.EventSchema, error) {
	const query = `
		SELECT ` + schemaSelectColumns + `
		FROM event_schemas
		WHERE publisher_user_id = $1
		  AND status = 'published'
		ORDER BY contract_id ASC, network ASC, event_name ASC, version DESC, created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, publisherUserID)
	if err != nil {
		return nil, fmt.Errorf("list publisher schemas: %w", err)
	}
	defer rows.Close()

	return scanSchemas(rows)
}

func (r *SchemaRepository) VerifyContractForPublisher(
	ctx context.Context,
	publisherUserID, contractID, network, wallet string,
	verifiedAt time.Time,
) ([]model.EventSchema, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin verify contract tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const latestCommunityQuery = `
		SELECT DISTINCT ON (event_name)
			id, event_name, version
		FROM event_schemas
		WHERE publisher_user_id = $1
		  AND contract_id = $2
		  AND network = $3
		  AND trust_tier = 'community'
		  AND status = 'published'
		ORDER BY event_name, version DESC
	`

	rows, err := tx.Query(ctx, latestCommunityQuery, publisherUserID, contractID, network)
	if err != nil {
		return nil, fmt.Errorf("list latest community schemas: %w", err)
	}

	type promoteTarget struct {
		id        string
		eventName string
		version   int
	}
	var targets []promoteTarget
	for rows.Next() {
		var target promoteTarget
		if err := rows.Scan(&target.id, &target.eventName, &target.version); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan latest community schema: %w", err)
		}
		targets = append(targets, target)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest community schemas: %w", err)
	}
	if len(targets) == 0 {
		return nil, ErrSchemaNotFound
	}

	const archiveQuery = `
		UPDATE event_schemas
		SET status = 'archived',
		    updated_at = NOW()
		WHERE contract_id = $1
		  AND network = $2
		  AND event_name = $3
		  AND trust_tier = 'verified'
		  AND status = 'published'
	`

	const promoteQuery = `
		UPDATE event_schemas
		SET trust_tier = 'verified',
		    verified_wallet = $2,
		    verified_at = $3,
		    updated_at = NOW()
		WHERE id = $4
		  AND publisher_user_id = $1
		  AND trust_tier = 'community'
		  AND status = 'published'
		RETURNING ` + schemaSelectColumns

	var promoted []model.EventSchema
	for _, target := range targets {
		if _, err := tx.Exec(ctx, archiveQuery, contractID, network, target.eventName); err != nil {
			return nil, fmt.Errorf("archive verified schema: %w", err)
		}

		var schema model.EventSchema
		err := tx.QueryRow(ctx, promoteQuery, publisherUserID, wallet, verifiedAt, target.id).Scan(
			&schema.ID,
			&schema.ContractID,
			&schema.Network,
			&schema.EventName,
			&schema.Version,
			&schema.SchemaBody,
			&schema.Author,
			&schema.PublisherUserID,
			&schema.TrustTier,
			&schema.Status,
			&schema.VerifiedWallet,
			&schema.VerifiedAt,
			&schema.CreatedAt,
			&schema.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("promote schema to verified: %w", err)
		}
		promoted = append(promoted, schema)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit verify contract tx: %w", err)
	}
	return promoted, nil
}

func (r *SchemaRepository) scanOne(ctx context.Context, query string, args ...any) (model.EventSchema, error) {
	var schema model.EventSchema
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&schema.ID,
		&schema.ContractID,
		&schema.Network,
		&schema.EventName,
		&schema.Version,
		&schema.SchemaBody,
		&schema.Author,
		&schema.PublisherUserID,
		&schema.TrustTier,
		&schema.Status,
		&schema.VerifiedWallet,
		&schema.VerifiedAt,
		&schema.CreatedAt,
		&schema.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.EventSchema{}, ErrSchemaNotFound
	}
	if err != nil {
		return model.EventSchema{}, fmt.Errorf("scan schema: %w", err)
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
			&schema.PublisherUserID,
			&schema.TrustTier,
			&schema.Status,
			&schema.VerifiedWallet,
			&schema.VerifiedAt,
			&schema.CreatedAt,
			&schema.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schema row: %w", err)
		}
		items = append(items, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schemas: %w", err)
	}
	return items, nil
}
