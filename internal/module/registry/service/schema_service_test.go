package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

type mockSchemaRepo struct {
	nextVersion int
	items       []model.EventSchema
}

func (m *mockSchemaRepo) NextVersion(_ context.Context, _, _, _, _ string) (int, error) {
	m.nextVersion++
	return m.nextVersion, nil
}

func (m *mockSchemaRepo) Insert(
	_ context.Context,
	publisherUserID, contractID, network, eventName string,
	version int,
	schemaBody []byte,
	author *string,
) (model.EventSchema, error) {
	schema := model.EventSchema{
		ID:              "schema-1",
		ContractID:      contractID,
		Network:         network,
		EventName:       eventName,
		Version:         version,
		SchemaBody:      append(json.RawMessage(nil), schemaBody...),
		Author:          author,
		PublisherUserID: &publisherUserID,
		TrustTier:       "community",
		Status:          "published",
	}
	m.items = append(m.items, schema)
	return schema, nil
}

func (m *mockSchemaRepo) List(_ context.Context, _, _ string, _, _ int) ([]model.EventSchema, int, error) {
	return m.items, len(m.items), nil
}

func (m *mockSchemaRepo) ListByPublisher(_ context.Context, publisherUserID string, _, _ int) ([]model.EventSchema, int, error) {
	var out []model.EventSchema
	for _, item := range m.items {
		if item.PublisherUserID != nil && *item.PublisherUserID == publisherUserID {
			out = append(out, item)
		}
	}
	return out, len(out), nil
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

func (m *mockSchemaRepo) ListVersionsByEvent(_ context.Context, _, contractID, eventName string) ([]model.EventSchema, error) {
	var out []model.EventSchema
	for _, item := range m.items {
		if item.ContractID == contractID && item.EventName == eventName && item.Status == "published" {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *mockSchemaRepo) GetByIDForPublisher(_ context.Context, schemaID, publisherUserID string) (model.EventSchema, error) {
	for _, item := range m.items {
		if item.ID == schemaID && item.PublisherUserID != nil && *item.PublisherUserID == publisherUserID {
			return item, nil
		}
	}
	return model.EventSchema{}, ErrSchemaNotFound
}

func (m *mockSchemaRepo) GetPublishedByID(_ context.Context, schemaID string) (model.EventSchema, error) {
	for _, item := range m.items {
		if item.ID == schemaID && item.Status == "published" {
			return item, nil
		}
	}
	return model.EventSchema{}, ErrSchemaNotFound
}

func (m *mockSchemaRepo) MarkVerified(_ context.Context, schemaID, wallet string, verifiedAt time.Time) (model.EventSchema, error) {
	for i, item := range m.items {
		if item.ID == schemaID {
			m.items[i].TrustTier = "verified"
			m.items[i].VerifiedWallet = &wallet
			m.items[i].VerifiedAt = &verifiedAt
			return m.items[i], nil
		}
	}
	return model.EventSchema{}, ErrSchemaNotFound
}

func TestPublishIncrementsVersion(t *testing.T) {
	repo := &mockSchemaRepo{}
	svc := NewSchemaService(repo, "testnet")

	first, err := svc.Publish(context.Background(), model.PublishInput{
		PublisherUserID: "user-1",
		ContractID:      "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
		EventName:       "transfer",
		SchemaBody:      validTransferSchema(),
		Author:          "Blend Team",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 {
		t.Fatalf("expected version 1, got %d", first.Version)
	}

	second, err := svc.Publish(context.Background(), model.PublishInput{
		PublisherUserID: "user-1",
		ContractID:      "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
		EventName:       "transfer",
		SchemaBody:      validTransferSchema(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 {
		t.Fatalf("expected version 2, got %d", second.Version)
	}
}

func TestGetByIDReturnsPublishedSchema(t *testing.T) {
	repo := &mockSchemaRepo{
		items: []model.EventSchema{
			{
				ID:         "schema-abc",
				ContractID: "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
				Network:    "testnet",
				EventName:  "transfer",
				Version:    1,
				Status:     "published",
				TrustTier:  "community",
			},
		},
	}
	svc := NewSchemaService(repo, "testnet")

	got, err := svc.GetByID(context.Background(), "schema-abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "schema-abc" || got.EventName != "transfer" {
		t.Fatalf("unexpected schema: %+v", got)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	svc := NewSchemaService(&mockSchemaRepo{}, "testnet")
	_, err := svc.GetByID(context.Background(), "missing")
	if !errors.Is(err, ErrSchemaNotFound) {
		t.Fatalf("expected ErrSchemaNotFound, got %v", err)
	}
}

func TestListEventVersions(t *testing.T) {
	repo := &mockSchemaRepo{
		items: []model.EventSchema{
			{
				ID:         "schema-v2",
				ContractID: "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
				Network:    "testnet",
				EventName:  "transfer",
				Version:    2,
				Status:     "published",
				TrustTier:  "community",
			},
			{
				ID:         "schema-v1",
				ContractID: "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
				Network:    "testnet",
				EventName:  "transfer",
				Version:    1,
				Status:     "published",
				TrustTier:  "verified",
			},
		},
	}
	svc := NewSchemaService(repo, "testnet")

	result, err := svc.ListEventVersions(
		context.Background(),
		"testnet",
		"CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
		"transfer",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("expected 2 versions, got %+v", result)
	}
}
