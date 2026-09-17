package service

import (
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

func publisher(id string) *string {
	return &id
}

func TestGroupIntoBundlesSamePublishBatch(t *testing.T) {
	created := time.Date(2026, 9, 14, 16, 43, 21, 0, time.UTC)
	schemas := []model.EventSchema{
		{
			ContractID:      "C1",
			Network:         "testnet",
			EventName:       "counter_incremented",
			Version:         1,
			PublisherUserID: publisher("user-1"),
			TrustTier:       "community",
			Status:          "published",
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		{
			ContractID:      "C1",
			Network:         "testnet",
			EventName:       "threshold_reached",
			Version:         1,
			PublisherUserID: publisher("user-1"),
			TrustTier:       "community",
			Status:          "published",
			CreatedAt:       created.Add(2 * time.Second),
			UpdatedAt:       created.Add(2 * time.Second),
		},
	}

	bundles := groupIntoBundles(schemas)
	if len(bundles) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(bundles))
	}
	if bundles[0].Version != 1 {
		t.Fatalf("expected version 1, got %d", bundles[0].Version)
	}
	if len(bundles[0].Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(bundles[0].Events))
	}
}

func TestGroupIntoBundlesDifferentPublishTimes(t *testing.T) {
	first := time.Date(2026, 9, 14, 16, 43, 21, 0, time.UTC)
	second := first.Add(2 * time.Minute)
	schemas := []model.EventSchema{
		{
			ContractID:      "C1",
			Network:         "testnet",
			EventName:       "counter_incremented",
			Version:         1,
			PublisherUserID: publisher("user-1"),
			Status:          "published",
			CreatedAt:       first,
			UpdatedAt:       first,
		},
		{
			ContractID:      "C1",
			Network:         "testnet",
			EventName:       "counter_incremented",
			Version:         2,
			PublisherUserID: publisher("user-1"),
			Status:          "published",
			CreatedAt:       second,
			UpdatedAt:       second,
		},
	}

	bundles := groupIntoBundles(schemas)
	if len(bundles) != 2 {
		t.Fatalf("expected 2 bundles, got %d", len(bundles))
	}
}

func TestContractVerificationAggregate(t *testing.T) {
	verifiedAt := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	schemas := []model.EventSchema{
		{
			EventName:  "counter_incremented",
			TrustTier:  "verified",
			Status:     "published",
			VerifiedAt: &verifiedAt,
		},
		{
			EventName: "threshold_reached",
			TrustTier: "community",
			Status:    "published",
		},
	}

	isVerified, at := contractVerification(schemas)
	if !isVerified {
		t.Fatal("expected contract to be verified")
	}
	if at == nil || !at.Equal(verifiedAt) {
		t.Fatalf("expected verifiedAt %s, got %#v", verifiedAt, at)
	}
}

func TestBuildContractSummaries(t *testing.T) {
	created := time.Date(2026, 9, 14, 16, 43, 21, 0, time.UTC)
	schemas := []model.EventSchema{
		{
			ContractID: "C1",
			Network:    "testnet",
			EventName:  "counter_incremented",
			Version:    1,
			Status:     "published",
			CreatedAt:  created,
			UpdatedAt:  created,
		},
		{
			ContractID: "C1",
			Network:    "testnet",
			EventName:  "threshold_reached",
			Version:    1,
			Status:     "published",
			CreatedAt:  created.Add(time.Second),
			UpdatedAt:  created.Add(time.Second),
		},
	}

	items := buildContractSummaries(schemas)
	if len(items) != 1 {
		t.Fatalf("expected 1 contract summary, got %d", len(items))
	}
	if items[0].SchemaCount != 1 {
		t.Fatalf("expected 1 schema bundle, got %d", items[0].SchemaCount)
	}
	if items[0].EventCount != 2 {
		t.Fatalf("expected 2 events, got %d", items[0].EventCount)
	}
}

func TestBundleTierVerifiedWhenAnyEventVerified(t *testing.T) {
	created := time.Date(2026, 9, 14, 16, 43, 21, 0, time.UTC)
	schemas := []model.EventSchema{
		{
			EventName: "counter_incremented",
			Version:   1,
			TrustTier: "verified",
			Status:    "published",
			CreatedAt: created,
			UpdatedAt: created,
		},
		{
			EventName: "threshold_reached",
			Version:   1,
			TrustTier: "community",
			Status:    "published",
			CreatedAt: created.Add(time.Second),
			UpdatedAt: created.Add(time.Second),
		},
	}

	bundle := finalizeBundle(schemas)
	if bundle.TrustTier != "verified" {
		t.Fatalf("expected verified bundle tier, got %s", bundle.TrustTier)
	}
}
