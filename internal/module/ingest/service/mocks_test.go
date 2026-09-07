package service

import (
	"context"
	"sync"
	"time"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

type fakeStellarClient struct {
	mu sync.Mutex

	latestLedger uint32
	events       []stellar.ContractEvent
	nextLedger   uint32
	fetchErr     error
	latestErr    error
	fetchCalls   int
}

func (f *fakeStellarClient) LatestLedger(ctx context.Context) (uint32, error) {
	return f.latestLedger, f.latestErr
}

func (f *fakeStellarClient) FetchEvents(
	ctx context.Context,
	network string,
	input stellar.FetchEventsInput,
) ([]stellar.ContractEvent, uint32, error) {
	f.mu.Lock()
	f.fetchCalls++
	f.mu.Unlock()
	if f.fetchErr != nil {
		return nil, input.StartLedger, f.fetchErr
	}
	return append([]stellar.ContractEvent(nil), f.events...), f.nextLedger, nil
}

type fakeCursorRepo struct {
	state      model.IngestState
	getErr     error
	upsertErr  error
	lastLedger uint32
}

func (f *fakeCursorRepo) Get(ctx context.Context, network string) (model.IngestState, error) {
	if f.getErr != nil {
		return model.IngestState{}, f.getErr
	}
	if f.state.Network == "" {
		f.state.Network = network
	}
	return f.state, nil
}

func (f *fakeCursorRepo) Upsert(ctx context.Context, network string, lastLedger uint32, rpcCursor string) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.lastLedger = lastLedger
	f.state.LastLedger = lastLedger
	return nil
}

type fakeEventRepo struct {
	events     []model.ContractEvent
	upsertErr  error
	listErr    error
	listResult []model.ContractEvent
	upserts    int
}

func (f *fakeEventRepo) UpsertBatch(ctx context.Context, events []model.ContractEvent) error {
	f.upserts++
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeEventRepo) ListByLedgerRange(
	ctx context.Context,
	network string,
	fromLedger, toLedger uint32,
	limit int,
) ([]model.ContractEvent, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResult != nil {
		return append([]model.ContractEvent(nil), f.listResult...), nil
	}
	out := make([]model.ContractEvent, 0)
	for _, e := range f.events {
		if e.Ledger >= fromLedger && e.Ledger <= toLedger {
			out = append(out, e)
		}
	}
	return out, nil
}

type fakeDerivedRepo struct {
	addresses int
	tokens    int
	addrErr   error
	tokenErr  error
}

func (f *fakeDerivedRepo) UpsertAddresses(ctx context.Context, rows []model.EventAddress) error {
	if f.addrErr != nil {
		return f.addrErr
	}
	f.addresses += len(rows)
	return nil
}

func (f *fakeDerivedRepo) UpsertTokenEvents(ctx context.Context, rows []model.TokenEvent) error {
	if f.tokenErr != nil {
		return f.tokenErr
	}
	f.tokens += len(rows)
	return nil
}

type fakeBackfillRepo struct {
	state       model.BackfillState
	startErr    error
	updateErr   error
	completeErr error
	updates     []uint32
	completed   bool
}

func (f *fakeBackfillRepo) Get(ctx context.Context, network string) (model.BackfillState, error) {
	return f.state, nil
}

func (f *fakeBackfillRepo) Start(ctx context.Context, network string, fromLedger, toLedger uint32) (model.BackfillState, error) {
	if f.startErr != nil {
		return model.BackfillState{}, f.startErr
	}
	f.state = model.BackfillState{
		Network:    network,
		FromLedger: fromLedger,
		ToLedger:   toLedger,
		NextLedger: fromLedger,
		Status:     "running",
	}
	return f.state, nil
}

func (f *fakeBackfillRepo) UpdateProgress(ctx context.Context, network string, nextLedger uint32) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updates = append(f.updates, nextLedger)
	f.state.NextLedger = nextLedger
	return nil
}

func (f *fakeBackfillRepo) Complete(ctx context.Context, network string) error {
	if f.completeErr != nil {
		return f.completeErr
	}
	f.completed = true
	return nil
}

func testConfig() *config.Config {
	return &config.Config{
		ServiceName: "test-atlas",
		Stellar: config.StellarConfig{
			Network: "testnet",
			RPCURL:  "https://rpc.example",
		},
		Ingest: config.IngestConfig{
			PollInterval:     10 * time.Millisecond,
			PollIntervalMin:  5 * time.Millisecond,
			PollIntervalMax:  20 * time.Millisecond,
			BatchSize:        2,
			PageLimit:        1000,
			RetentionLedgers: 100,
			ReorgWindow:      0,
		},
		Backfill: config.BackfillConfig{
			BatchSize: 2,
			Interval:  time.Millisecond,
		},
		Replay: config.ReplayConfig{BatchSize: 10},
	}
}
