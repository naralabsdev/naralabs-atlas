package service

import (
	"context"
	"errors"
	"strings"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/repository"
)

type SchemaRepository interface {
	NextVersion(ctx context.Context, contractID, network, eventName string) (int, error)
	Insert(ctx context.Context, contractID, network, eventName string, version int, schemaBody []byte, author *string) (model.EventSchema, error)
	List(ctx context.Context, network, search string, limit, offset int) ([]model.EventSchema, int, error)
	ListByContract(ctx context.Context, network, contractID string) ([]model.EventSchema, error)
	GetEventSchema(ctx context.Context, network, contractID, eventName string, version int) (model.EventSchema, error)
}

type SchemaService struct {
	repo            SchemaRepository
	defaultNetwork  string
}

func NewSchemaService(repo SchemaRepository, defaultNetwork string) *SchemaService {
	return &SchemaService{
		repo:           repo,
		defaultNetwork: strings.ToLower(strings.TrimSpace(defaultNetwork)),
	}
}

func (s *SchemaService) Publish(ctx context.Context, input model.PublishInput) (model.EventSchemaPublic, error) {
	contractID, err := normalizeContractID(input.ContractID)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	network, err := s.resolveNetwork(input.Network)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	eventName, err := normalizeEventName(input.EventName)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	if err := validateSchemaBody(eventName, input.SchemaBody); err != nil {
		return model.EventSchemaPublic{}, err
	}

	version, err := s.repo.NextVersion(ctx, contractID, network, eventName)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	var author *string
	if authorText := strings.TrimSpace(input.Author); authorText != "" {
		author = &authorText
	}

	schema, err := s.repo.Insert(
		ctx,
		contractID,
		network,
		eventName,
		version,
		append([]byte(nil), input.SchemaBody...),
		author,
	)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	return schema.Public(), nil
}

func (s *SchemaService) List(
	ctx context.Context,
	network, search string,
	limit, offset int,
) (model.ListResponse, error) {
	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return model.ListResponse{}, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.repo.List(ctx, resolved, search, limit, offset)
	if err != nil {
		return model.ListResponse{}, err
	}

	out := model.ListResponse{
		Total: total,
		Items: make([]model.EventSchemaPublic, 0, len(items)),
	}
	for _, item := range items {
		out.Items = append(out.Items, item.Public())
	}
	return out, nil
}

func (s *SchemaService) ListByContract(
	ctx context.Context,
	network, contractID string,
) ([]model.EventSchemaPublic, error) {
	id, err := normalizeContractID(contractID)
	if err != nil {
		return nil, err
	}

	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByContract(ctx, resolved, id)
	if err != nil {
		return nil, err
	}

	out := make([]model.EventSchemaPublic, 0, len(items))
	for _, item := range items {
		out = append(out, item.Public())
	}
	return out, nil
}

func (s *SchemaService) GetEventSchema(
	ctx context.Context,
	network, contractID, eventName string,
	version int,
) (model.EventSchemaPublic, error) {
	id, err := normalizeContractID(contractID)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	name, err := normalizeEventName(eventName)
	if err != nil {
		return model.EventSchemaPublic{}, err
	}

	schema, err := s.repo.GetEventSchema(ctx, resolved, id, name, version)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			return model.EventSchemaPublic{}, ErrSchemaNotFound
		}
		return model.EventSchemaPublic{}, err
	}
	return schema.Public(), nil
}

func (s *SchemaService) resolveNetwork(network string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(network))
	if n == "" {
		if s.defaultNetwork == "" {
			return "", ErrInvalidNetwork
		}
		return s.defaultNetwork, nil
	}
	return normalizeNetwork(n)
}
