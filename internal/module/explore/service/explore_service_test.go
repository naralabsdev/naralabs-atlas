package service

import (
	"context"
	"errors"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
)

type fakeExploreRepo struct {
	stats     model.NetworkStats
	statsErr  error
	events    []model.EventItem
	eventsErr error
	contracts []model.ContractItem
	contractsErr error
	event     model.EventDetail
	eventErr  error
	contract  model.ContractDetail
	contractErr error
}

func (f *fakeExploreRepo) GetStats(_ context.Context, network string) (model.NetworkStats, error) {
	if f.statsErr != nil {
		return model.NetworkStats{}, f.statsErr
	}
	out := f.stats
	if out.Network == "" {
		out.Network = network
	}
	return out, nil
}

func (f *fakeExploreRepo) ListRecentEvents(_ context.Context, _ string, _ int) ([]model.EventItem, error) {
	return f.events, f.eventsErr
}

func (f *fakeExploreRepo) ListActiveContracts(_ context.Context, _ string, _ int) ([]model.ContractItem, error) {
	return f.contracts, f.contractsErr
}

func (f *fakeExploreRepo) GetEventByID(_ context.Context, _, id string) (model.EventDetail, error) {
	if f.eventErr != nil {
		return model.EventDetail{}, f.eventErr
	}
	out := f.event
	if out.ID == "" {
		out.ID = id
	}
	return out, f.eventErr
}

func (f *fakeExploreRepo) GetContractByID(_ context.Context, network, contractID string) (model.ContractDetail, error) {
	if f.contractErr != nil {
		return model.ContractDetail{}, f.contractErr
	}
	out := f.contract
	if out.ContractID == "" {
		out.ContractID = contractID
	}
	if out.Network == "" {
		out.Network = network
	}
	return out, nil
}

func (f *fakeExploreRepo) ListContractEvents(_ context.Context, _, _ string, page, pageSize int, _, _, _ string) (model.PaginatedListResponse[model.EventItem], error) {
	if f.contractErr != nil {
		return model.PaginatedListResponse[model.EventItem]{}, f.contractErr
	}
	return model.PaginatedListResponse[model.EventItem]{
		Items:    f.events,
		Total:    uint64(len(f.events)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

type fakeStellarClient struct {
	head    uint32
	headErr error
}

func (f *fakeStellarClient) LatestLedger(_ context.Context) (uint32, error) {
	return f.head, f.headErr
}

func (f *fakeStellarClient) FetchEvents(context.Context, string, stellar.FetchEventsInput) ([]stellar.ContractEvent, uint32, error) {
	return nil, 0, nil
}

func TestGetStatsEnrichesChainHead(t *testing.T) {
	svc := NewExploreService(
		&fakeExploreRepo{stats: model.NetworkStats{LastIngestedLedger: 100}},
		&fakeStellarClient{head: 110},
	)

	stats, err := svc.GetStats(context.Background(), "testnet")
	if err != nil {
		t.Fatal(err)
	}
	if stats.ChainHeadLedger == nil || *stats.ChainHeadLedger != 110 {
		t.Fatalf("head=%v", stats.ChainHeadLedger)
	}
	if stats.IngestLagLedgers == nil || *stats.IngestLagLedgers != 10 {
		t.Fatalf("lag=%v", stats.IngestLagLedgers)
	}
}

func TestGetStatsIgnoresStellarError(t *testing.T) {
	svc := NewExploreService(
		&fakeExploreRepo{stats: model.NetworkStats{LastIngestedLedger: 50}},
		&fakeStellarClient{headErr: errors.New("rpc down")},
	)

	stats, err := svc.GetStats(context.Background(), "testnet")
	if err != nil {
		t.Fatal(err)
	}
	if stats.ChainHeadLedger != nil {
		t.Fatal("expected no chain head when rpc fails")
	}
}

func TestGetHomeAggregatesPayload(t *testing.T) {
	repo := &fakeExploreRepo{
		stats:     model.NetworkStats{TotalEvents: 5},
		events:    []model.EventItem{{ID: "e1"}},
		contracts: []model.ContractItem{{ContractID: "C1"}},
	}
	svc := NewExploreService(repo, &fakeStellarClient{})

	payload, err := svc.GetHome(context.Background(), "testnet", 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Stats.TotalEvents != 5 || len(payload.RecentEvents) != 1 || len(payload.ActiveContracts) != 1 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestGetHomePropagatesStatsError(t *testing.T) {
	svc := NewExploreService(
		&fakeExploreRepo{statsErr: errors.New("stats failed")},
		&fakeStellarClient{},
	)
	_, err := svc.GetHome(context.Background(), "testnet", 8, 8)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetHomePropagatesEventsError(t *testing.T) {
	svc := NewExploreService(
		&fakeExploreRepo{stats: model.NetworkStats{}, eventsErr: errors.New("events failed")},
		&fakeStellarClient{},
	)
	_, err := svc.GetHome(context.Background(), "testnet", 8, 8)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetHomePropagatesContractsError(t *testing.T) {
	svc := NewExploreService(
		&fakeExploreRepo{
			stats:        model.NetworkStats{},
			events:       []model.EventItem{{ID: "e1"}},
			contractsErr: errors.New("contracts failed"),
		},
		&fakeStellarClient{},
	)
	_, err := svc.GetHome(context.Background(), "testnet", 8, 8)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListRecentEventsDelegates(t *testing.T) {
	repo := &fakeExploreRepo{events: []model.EventItem{{ID: "evt"}}}
	svc := NewExploreService(repo, &fakeStellarClient{})

	items, err := svc.ListRecentEvents(context.Background(), "testnet", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "evt" {
		t.Fatalf("items=%+v", items)
	}
}

func TestListActiveContractsDelegates(t *testing.T) {
	repo := &fakeExploreRepo{contracts: []model.ContractItem{{ContractID: "C1"}}}
	svc := NewExploreService(repo, &fakeStellarClient{})

	items, err := svc.ListActiveContracts(context.Background(), "testnet", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ContractID != "C1" {
		t.Fatalf("items=%+v", items)
	}
}
