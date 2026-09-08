package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
)

type fakeExploreService struct {
	stats     model.NetworkStats
	statsErr  error
	events    []model.EventItem
	eventsErr error
	contracts []model.ContractItem
	contractsErr error
	home      model.HomePayload
	homeErr   error
	event     model.EventDetail
	eventErr  error
	contract  model.ContractDetail
	contractErr error
	contractEvents model.PaginatedListResponse[model.EventItem]
	contractEventsErr error

	lastNetwork       string
	lastEventLimit    int
	lastContractLimit int
	lastHomeLimits    [2]int
	lastContractID    string
	lastRecentLimit   int
	lastContractEventsPage int
	lastContractEventsSize int
	lastEventsPage         int
	lastEventsSize         int
	lastContractsPage      int
	lastContractsSize      int
}

func (f *fakeExploreService) GetStats(_ context.Context, network string) (model.NetworkStats, error) {
	f.lastNetwork = network
	return f.stats, f.statsErr
}

func (f *fakeExploreService) ListRecentEvents(_ context.Context, network string, limit int) ([]model.EventItem, error) {
	f.lastNetwork = network
	f.lastEventLimit = limit
	return f.events, f.eventsErr
}

func (f *fakeExploreService) ListActiveContracts(_ context.Context, network string, limit int) ([]model.ContractItem, error) {
	f.lastNetwork = network
	f.lastContractLimit = limit
	return f.contracts, f.contractsErr
}

func (f *fakeExploreService) GetHome(_ context.Context, network string, recentLimit, contractLimit int) (model.HomePayload, error) {
	f.lastNetwork = network
	f.lastHomeLimits = [2]int{recentLimit, contractLimit}
	return f.home, f.homeErr
}

func (f *fakeExploreService) GetEvent(_ context.Context, network, id string) (model.EventDetail, error) {
	f.lastNetwork = network
	if f.event.ID == "" && f.eventErr == nil {
		f.event.ID = id
	}
	return f.event, f.eventErr
}

func (f *fakeExploreService) GetContract(_ context.Context, network, contractID string) (model.ContractDetail, error) {
	f.lastNetwork = network
	f.lastContractID = contractID
	if f.contractErr != nil {
		return model.ContractDetail{}, f.contractErr
	}
	out := f.contract
	if out.ContractID == "" {
		out.ContractID = contractID
	}
	return out, f.contractErr
}

func (f *fakeExploreService) ListEvents(_ context.Context, network string, page, pageSize int, _, _, _ string) (model.PaginatedListResponse[model.EventItem], error) {
	f.lastNetwork = network
	f.lastEventsPage = page
	f.lastEventsSize = pageSize
	if f.eventsErr != nil {
		return model.PaginatedListResponse[model.EventItem]{}, f.eventsErr
	}
	return model.PaginatedListResponse[model.EventItem]{
		Items:    f.events,
		Total:    uint64(len(f.events)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (f *fakeExploreService) ListContracts(_ context.Context, network string, page, pageSize int, _, _ string) (model.PaginatedListResponse[model.ContractItem], error) {
	f.lastNetwork = network
	f.lastContractsPage = page
	f.lastContractsSize = pageSize
	if f.contractsErr != nil {
		return model.PaginatedListResponse[model.ContractItem]{}, f.contractsErr
	}
	return model.PaginatedListResponse[model.ContractItem]{
		Items:    f.contracts,
		Total:    uint64(len(f.contracts)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (f *fakeExploreService) ListContractEvents(_ context.Context, network, contractID string, page, pageSize int, _, _, _ string) (model.PaginatedListResponse[model.EventItem], error) {
	f.lastNetwork = network
	f.lastContractID = contractID
	f.lastContractEventsPage = page
	f.lastContractEventsSize = pageSize
	if f.contractErr != nil {
		return model.PaginatedListResponse[model.EventItem]{}, f.contractErr
	}
	if f.contractEventsErr != nil {
		return model.PaginatedListResponse[model.EventItem]{}, f.contractEventsErr
	}
	if len(f.contractEvents.Items) > 0 || f.contractEvents.Total > 0 {
		return f.contractEvents, nil
	}
	return model.PaginatedListResponse[model.EventItem]{
		Items:    f.events,
		Total:    uint64(len(f.events)),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func TestHandleHealth(t *testing.T) {
	h := NewExploreHandler(&fakeExploreService{}, "testnet")
	out, err := h.HandleHealth(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Body.Status != "ok" {
		t.Fatalf("status=%q", out.Body.Status)
	}
}

func TestHandleStatsUsesDefaultNetwork(t *testing.T) {
	svc := &fakeExploreService{stats: model.NetworkStats{Network: "testnet", TotalEvents: 10}}
	h := NewExploreHandler(svc, "testnet")

	out, err := h.HandleStats(context.Background(), &StatsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastNetwork != "testnet" {
		t.Fatalf("network=%q", svc.lastNetwork)
	}
	if out.Body.TotalEvents != 10 {
		t.Fatalf("events=%d", out.Body.TotalEvents)
	}
}

func TestHandleStatsError(t *testing.T) {
	h := NewExploreHandler(&fakeExploreService{statsErr: errors.New("boom")}, "testnet")
	_, err := h.HandleStats(context.Background(), &StatsInput{NetworkQuery: NetworkQuery{Network: "mainnet"}})
	if err == nil {
		t.Fatal("expected error")
	}
	var se huma.StatusError
	if !errors.As(err, &se) || se.GetStatus() != 500 {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleRecentEventsClampsPageSize(t *testing.T) {
	svc := &fakeExploreService{events: []model.EventItem{{ID: "e1"}}}
	h := NewExploreHandler(svc, "testnet")

	out, err := h.HandleRecentEvents(context.Background(), &EventsInput{PageSize: 200})
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastEventsSize != 100 {
		t.Fatalf("page_size=%d", svc.lastEventsSize)
	}
	if len(out.Body.Items) != 1 {
		t.Fatalf("items=%d", len(out.Body.Items))
	}
}

func TestHandleActiveContractsError(t *testing.T) {
	h := NewExploreHandler(&fakeExploreService{contractsErr: errors.New("db down")}, "testnet")
	_, err := h.HandleActiveContracts(context.Background(), &ContractsInput{PageSize: 5})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandleHomeSuccess(t *testing.T) {
	svc := &fakeExploreService{
		home: model.HomePayload{
			Stats:         model.NetworkStats{Network: "testnet", TotalEvents: 3},
			RecentEvents:  []model.EventItem{{ID: "e1"}},
			ActiveContracts: []model.ContractItem{{ContractID: "C1"}},
		},
	}
	h := NewExploreHandler(svc, "testnet")

	out, err := h.HandleHome(context.Background(), &HomeInput{
		RecentLimit:   0,
		ContractLimit: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastHomeLimits != [2]int{8, 12} {
		t.Fatalf("limits=%v", svc.lastHomeLimits)
	}
	if out.Body.Stats.TotalEvents != 3 || len(out.Body.RecentEvents) != 1 || len(out.Body.ActiveContracts) != 1 {
		t.Fatalf("payload=%+v", out.Body)
	}
}

func TestHandleGetEventSuccess(t *testing.T) {
	svc := &fakeExploreService{
		event: model.EventDetail{
			ID:         "evt-1",
			Network:    "testnet",
			EventType:  "transfer",
			EventKind:  "contract",
			DecodeStatus: "decoded",
		},
	}
	h := NewExploreHandler(svc, "testnet")

	out, err := h.HandleGetEvent(context.Background(), &EventDetailInput{ID: "evt-1"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Body.ID != "evt-1" || out.Body.EventType != "transfer" {
		t.Fatalf("body=%+v", out.Body)
	}
}

func TestHandleGetEventNotFound(t *testing.T) {
	h := NewExploreHandler(&fakeExploreService{eventErr: repository.ErrEventNotFound}, "testnet")
	_, err := h.HandleGetEvent(context.Background(), &EventDetailInput{ID: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
	var se huma.StatusError
	if !errors.As(err, &se) || se.GetStatus() != 404 {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleGetContractSuccess(t *testing.T) {
	svc := &fakeExploreService{
		contract: model.ContractDetail{
			ContractID: "C1",
			Network:    "testnet",
			EventCount: 42,
		},
	}
	h := NewExploreHandler(svc, "testnet")

	out, err := h.HandleGetContract(context.Background(), &ContractDetailInput{ID: "C1"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Body.ContractID != "C1" || out.Body.EventCount != 42 {
		t.Fatalf("body=%+v", out.Body)
	}
}

func TestHandleGetContractNotFound(t *testing.T) {
	h := NewExploreHandler(&fakeExploreService{contractErr: repository.ErrContractNotFound}, "testnet")
	_, err := h.HandleGetContract(context.Background(), &ContractDetailInput{ID: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
	var se huma.StatusError
	if !errors.As(err, &se) || se.GetStatus() != 404 {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleListContractEventsSuccess(t *testing.T) {
	svc := &fakeExploreService{
		contract: model.ContractDetail{ContractID: "C1", EventCount: 42},
		contractEvents: model.PaginatedListResponse[model.EventItem]{
			Items:    []model.EventItem{{ID: "e1"}},
			Total:    1,
			Page:     1,
			PageSize: 20,
		},
	}
	h := NewExploreHandler(svc, "testnet")

	out, err := h.HandleListContractEvents(context.Background(), &ContractEventsInput{ID: "C1"})
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastContractEventsPage != 1 || svc.lastContractEventsSize != 20 {
		t.Fatalf("page=%d size=%d", svc.lastContractEventsPage, svc.lastContractEventsSize)
	}
	if len(out.Body.Items) != 1 || out.Body.Total != 1 {
		t.Fatalf("body=%+v", out.Body)
	}
}

func TestHandleListContractEventsNotFound(t *testing.T) {
	h := NewExploreHandler(&fakeExploreService{contractErr: repository.ErrContractNotFound}, "testnet")
	_, err := h.HandleListContractEvents(context.Background(), &ContractEventsInput{ID: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
	var se huma.StatusError
	if !errors.As(err, &se) || se.GetStatus() != 404 {
		t.Fatalf("err=%v", err)
	}
}

func TestClampPageSize(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 20},
		{-1, 1},
		{25, 25},
		{200, 100},
	}
	for _, tc := range tests {
		if got := clampPageSize(tc.in); got != tc.want {
			t.Fatalf("clampPageSize(%d)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestClampLimit(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 8},
		{-1, 1},
		{50, 50},
		{500, 100},
	}
	for _, tc := range tests {
		if got := clampLimit(tc.in); got != tc.want {
			t.Fatalf("clampLimit(%d)=%d want %d", tc.in, got, tc.want)
		}
	}
}
