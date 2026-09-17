package service

import (
	"context"

	registermodel "github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

type SchemaRepositoryAdapter struct {
	repo SchemaReader
}

type SchemaReader interface {
	ListByContract(ctx context.Context, network, contractID string) ([]registermodel.EventSchema, error)
	GetEventSchema(ctx context.Context, network, contractID, eventName string, version int) (registermodel.EventSchema, error)
}

func NewSchemaRepositoryAdapter(repo SchemaReader) *SchemaRepositoryAdapter {
	return &SchemaRepositoryAdapter{repo: repo}
}

func (a *SchemaRepositoryAdapter) ListByContract(
	ctx context.Context,
	network, contractID string,
) ([]registermodel.EventSchema, error) {
	return a.repo.ListByContract(ctx, network, contractID)
}

func (a *SchemaRepositoryAdapter) GetEventSchema(
	ctx context.Context,
	network, contractID, eventName string,
	version int,
) (registermodel.EventSchema, error) {
	return a.repo.GetEventSchema(ctx, network, contractID, eventName, version)
}
