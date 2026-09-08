package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

type mockSchemaRepo struct {
	nextVersion int
	items       []model.EventSchema
}

func (m *mockSchemaRepo) NextVersion(_ context.Context, _, _, _ string) (int, error) {
	m.nextVersion++
	return m.nextVersion, nil
}

func (m *mockSchemaRepo) Insert(
	_ context.Context,
	contractID, network, eventName string,
	version int,
	schemaBody []byte,
	author *string,
) (model.EventSchema, error) {
	schema := model.EventSchema{
		ID:         "schema-1",
		ContractID: contractID,
		Network:    network,
		EventName:  eventName,
		Version:    version,
		SchemaBody: append(json.RawMessage(nil), schemaBody...),
		Author:     author,
	}
	m.items = append(m.items, schema)
	return schema, nil
}

func (m *mockSchemaRepo) List(_ context.Context, _, _ string, _, _ int) ([]model.EventSchema, int, error) {
	return m.items, len(m.items), nil
}

func (m *mockSchemaRepo) ListByContract(_ context.Context, _, contractID string) ([]model.EventSchema, error) {
	var out []model.EventSchema
	for _, item := range m.items {
		if item.ContractID == contractID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *mockSchemaRepo) GetEventSchema(_ context.Context, _, contractID, eventName string, version int) (model.EventSchema, error) {
	for _, item := range m.items {
		if item.ContractID == contractID && item.EventName == eventName && (version == 0 || item.Version == version) {
			return item, nil
		}
	}
	return model.EventSchema{}, ErrSchemaNotFound
}

func TestPublishIncrementsVersion(t *testing.T) {
	repo := &mockSchemaRepo{}
	svc := NewSchemaService(repo, "testnet")

	first, err := svc.Publish(context.Background(), model.PublishInput{
		ContractID: "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
		EventName:  "transfer",
		SchemaBody: validTransferSchema(),
		Author:     "Blend Team",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 {
		t.Fatalf("expected version 1, got %d", first.Version)
	}

	second, err := svc.Publish(context.Background(), model.PublishInput{
		ContractID: "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
		EventName:  "transfer",
		SchemaBody: validTransferSchema(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 {
		t.Fatalf("expected version 2, got %d", second.Version)
	}
}
