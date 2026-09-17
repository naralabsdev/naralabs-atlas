package service

import (
	"context"
	"errors"
	"testing"
	"time"

	exploremodel "github.com/naralabs/naralabs-atlas/internal/module/explore/model"
	explorerepo "github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

type mockBundleSchemaRepo struct {
	byContract []model.EventSchema
	byNetwork  []model.EventSchema
}

func (m *mockBundleSchemaRepo) ListPublishedByNetwork(_ context.Context, _, _ string) ([]model.EventSchema, error) {
	return m.byNetwork, nil
}

func (m *mockBundleSchemaRepo) ListPublishedForContract(_ context.Context, _, _ string) ([]model.EventSchema, error) {
	return m.byContract, nil
}

type mockActivityReader struct {
	detail exploremodel.ContractDetail
	err    error
}

func (m *mockActivityReader) GetContractByID(_ context.Context, _, _ string) (exploremodel.ContractDetail, error) {
	if m.err != nil {
		return exploremodel.ContractDetail{}, m.err
	}
	return m.detail, nil
}

func TestGetContractProfileUnindexedContractStillReturns200Shape(t *testing.T) {
	svc := NewSchemaBundleService(
		&mockBundleSchemaRepo{byContract: nil},
		&mockActivityReader{err: explorerepo.ErrContractNotFound},
		"testnet",
	)

	profile, err := svc.GetContractProfile(context.Background(), "testnet", "CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Indexed {
		t.Fatal("expected indexed=false")
	}
	if profile.Activity != nil {
		t.Fatal("expected nil activity")
	}
	if len(profile.Bundles) != 0 {
		t.Fatalf("expected empty bundles, got %d", len(profile.Bundles))
	}
}

func TestGetContractProfileIncludesActivityWhenIndexed(t *testing.T) {
	lastSeen := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	svc := NewSchemaBundleService(
		&mockBundleSchemaRepo{byContract: nil},
		&mockActivityReader{
			detail: exploremodel.ContractDetail{
				ContractID: "C1",
				Network:    "testnet",
				EventCount: 12,
				LastSeen:   lastSeen,
				SchemaStatus: "partial",
			},
		},
		"testnet",
	)

	profile, err := svc.GetContractProfile(context.Background(), "testnet", "CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4")
	if err != nil {
		t.Fatal(err)
	}
	if !profile.Indexed {
		t.Fatal("expected indexed=true")
	}
	if profile.Activity == nil || profile.Activity.EventCount != 12 {
		t.Fatalf("expected activity event count 12, got %#v", profile.Activity)
	}
}

func TestGetRegistrySummaryAggregatesContractsAndTrust(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	svc := NewSchemaBundleService(
		&mockBundleSchemaRepo{
			byNetwork: []model.EventSchema{
				{
					ContractID: "C1",
					Network:    "testnet",
					EventName:  "transfer",
					Version:    1,
					Status:     "published",
					TrustTier:  "verified",
					UpdatedAt:  now,
				},
				{
					ContractID: "C1",
					Network:    "testnet",
					EventName:  "mint",
					Version:    1,
					Status:     "published",
					TrustTier:  "verified",
					UpdatedAt:  now,
				},
				{
					ContractID: "C2",
					Network:    "testnet",
					EventName:  "incr",
					Version:    1,
					Status:     "published",
					TrustTier:  "community",
					UpdatedAt:  now,
				},
			},
		},
		nil,
		"testnet",
	)

	summary, err := svc.GetRegistrySummary(context.Background(), "testnet")
	if err != nil {
		t.Fatal(err)
	}
	if summary.PublishedContracts != 2 {
		t.Fatalf("expected 2 contracts, got %d", summary.PublishedContracts)
	}
	if summary.EventSchemas != 3 {
		t.Fatalf("expected 3 event schemas, got %d", summary.EventSchemas)
	}
	if summary.VerifiedContracts != 1 {
		t.Fatalf("expected 1 verified contract, got %d", summary.VerifiedContracts)
	}
	if summary.CommunityContracts != 1 {
		t.Fatalf("expected 1 community contract, got %d", summary.CommunityContracts)
	}
}

func TestGetContractProfileInvalidContractID(t *testing.T) {
	svc := NewSchemaBundleService(&mockBundleSchemaRepo{}, nil, "testnet")
	_, err := svc.GetContractProfile(context.Background(), "testnet", "invalid")
	if !errors.Is(err, ErrInvalidContractID) {
		t.Fatalf("expected invalid contract id, got %v", err)
	}
}
