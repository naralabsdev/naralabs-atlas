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

var ErrSchemaProjectNotFound = errors.New("schema project not found")

const projectSelectColumns = `
	id, user_id, name, slug, description, status, publish_token_id, publish_token,
	contract_id, network, created_at, updated_at
`

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) Insert(
	ctx context.Context,
	userID, name, slug, description, status, publishTokenID, publishToken string,
) (model.SchemaProject, error) {
	query := `
		INSERT INTO schema_projects (
			user_id, name, slug, description, status, publish_token_id, publish_token
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING` + projectSelectColumns

	var project model.SchemaProject
	err := scanProjectFields(&project, r.pool.QueryRow(
		ctx,
		query,
		userID,
		name,
		slug,
		description,
		status,
		publishTokenID,
		publishToken,
	))
	if err != nil {
		return model.SchemaProject{}, fmt.Errorf("insert schema project: %w", err)
	}
	return project, nil
}

func (r *ProjectRepository) ListByUser(ctx context.Context, userID string) ([]model.SchemaProject, error) {
	query := `SELECT` + projectSelectColumns + `
		FROM schema_projects
		WHERE user_id = $1 AND status <> 'archived'
		ORDER BY updated_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list schema projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

func (r *ProjectRepository) GetByIDForUser(
	ctx context.Context,
	projectID, userID string,
) (model.SchemaProject, error) {
	query := `SELECT` + projectSelectColumns + `
		FROM schema_projects
		WHERE id = $1 AND user_id = $2 AND status <> 'archived'
	`

	return r.scanOne(ctx, query, projectID, userID)
}

func (r *ProjectRepository) MarkPublishedByPublishTokenID(
	ctx context.Context,
	publishTokenID, contractID, network string,
) error {
	const query = `
		UPDATE schema_projects
		SET status = CASE WHEN status = 'draft' THEN 'published' ELSE status END,
		    contract_id = COALESCE(contract_id, NULLIF($2, '')),
		    network = COALESCE(network, NULLIF($3, '')),
		    updated_at = NOW()
		WHERE publish_token_id = $1
	`

	_, err := r.pool.Exec(ctx, query, publishTokenID, contractID, network)
	if err != nil {
		return fmt.Errorf("mark schema project published: %w", err)
	}
	return nil
}

func (r *ProjectRepository) SyncPublishedStatus(
	ctx context.Context,
	projectID, userID string,
) (model.SchemaProject, error) {
	query := `
		UPDATE schema_projects sp
		SET status = 'published',
		    updated_at = NOW()
		WHERE sp.id = $1
		  AND sp.user_id = $2
		  AND sp.status = 'draft'
		  AND EXISTS (
		    SELECT 1
		    FROM publish_tokens pt
		    WHERE pt.id = sp.publish_token_id
		      AND pt.last_used_at IS NOT NULL
		  )
		RETURNING` + projectSelectColumns

	return r.scanOne(ctx, query, projectID, userID)
}

func (r *ProjectRepository) Update(
	ctx context.Context,
	projectID, userID, name, slug, description string,
) (model.SchemaProject, error) {
	query := `
		UPDATE schema_projects
		SET name = $3,
		    slug = $4,
		    description = $5,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status <> 'archived'
		RETURNING` + projectSelectColumns

	return r.scanOne(ctx, query, projectID, userID, name, slug, description)
}

func (r *ProjectRepository) SlugExists(ctx context.Context, userID, slug string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM schema_projects WHERE user_id = $1 AND slug = $2 AND status <> 'archived'
		)
	`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, userID, slug).Scan(&exists); err != nil {
		return false, fmt.Errorf("check schema project slug: %w", err)
	}
	return exists, nil
}

func scanProjectFields(project *model.SchemaProject, scanner interface {
	Scan(dest ...any) error
}) error {
	return scanner.Scan(
		&project.ID,
		&project.UserID,
		&project.Name,
		&project.Slug,
		&project.Description,
		&project.Status,
		&project.PublishTokenID,
		&project.PublishToken,
		&project.ContractID,
		&project.Network,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
}

func (r *ProjectRepository) scanOne(ctx context.Context, query string, args ...any) (model.SchemaProject, error) {
	var project model.SchemaProject
	err := scanProjectFields(&project, r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SchemaProject{}, ErrSchemaProjectNotFound
	}
	if err != nil {
		return model.SchemaProject{}, fmt.Errorf("scan schema project: %w", err)
	}
	return project, nil
}

func scanProjects(rows pgx.Rows) ([]model.SchemaProject, error) {
	var items []model.SchemaProject
	for rows.Next() {
		var project model.SchemaProject
		if err := scanProjectFields(&project, rows); err != nil {
			return nil, fmt.Errorf("scan schema project row: %w", err)
		}
		items = append(items, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema projects: %w", err)
	}
	return items, nil
}

func NormalizeProjectName(name string) string {
	return strings.TrimSpace(name)
}
