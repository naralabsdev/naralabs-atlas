package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/lib/rpcchain"
)

func TestIngestOnceCaughtUp(t *testing.T) {
	cfg := testConfig()
	stellarClient := &fakeStellarClient{latestLedger: 100}
	cursor := &fakeCursorRepo{state: model.IngestState{LastLedger: 100}}
	events := &fakeEventRepo{}
	derived := &fakeDerivedRepo{}

	w := NewIngestWorkerService(cfg, testLogger(), stellarClient, cursor, events, derived)
	caughtUp, err := w.ingestOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !caughtUp {
		t.Fatal("expected caught up")
	}
	if events.upserts != 0 {
		t.Fatalf("upserts=%d", events.upserts)
	}
}

func TestIngestOncePersistsEventsAndUpdatesCursor(t *testing.T) {
	cfg := testConfig()
	stellarClient := &fakeStellarClient{
		latestLedger: 200,
		nextLedger:   150,
		events: []stellar.ContractEvent{{
			ID: "evt-1", Network: "testnet", ContractID: "C1", Ledger: 120,
		}},
	}
	cursor := &fakeCursorRepo{state: model.IngestState{LastLedger: 100}}
	events := &fakeEventRepo{}
	derived := &fakeDerivedRepo{}

	w := NewIngestWorkerService(cfg, testLogger(), stellarClient, cursor, events, derived)
	caughtUp, err := w.ingestOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caughtUp {
		t.Fatal("expected backlog")
	}
	if cursor.lastLedger != 150 {
		t.Fatalf("cursor=%d", cursor.lastLedger)
	}
	if events.upserts != 1 {
		t.Fatalf("upserts=%d", events.upserts)
	}
}

func TestIngestOnceReanchorReturnsWithoutError(t *testing.T) {
	cfg := testConfig()
	stellarClient := &fakeStellarClient{
		latestLedger: 200,
		fetchErr:     rpcchain.ErrReanchorCursor,
	}
	cursor := &fakeCursorRepo{state: model.IngestState{LastLedger: 100}}

	w := NewIngestWorkerService(cfg, testLogger(), stellarClient, cursor, &fakeEventRepo{}, &fakeDerivedRepo{})
	caughtUp, err := w.ingestOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caughtUp {
		t.Fatal("expected not caught up after reanchor")
	}
}

func TestPersistChunksByBatchSize(t *testing.T) {
	cfg := testConfig()
	cfg.Ingest.BatchSize = 2
	events := &fakeEventRepo{}
	derived := &fakeDerivedRepo{}
	w := NewIngestWorkerService(cfg, testLogger(), &fakeStellarClient{}, &fakeCursorRepo{}, events, derived)

	input := []model.ContractEvent{
		{ID: "1", Ledger: 1},
		{ID: "2", Ledger: 1},
		{ID: "3", Ledger: 1},
	}
	if err := w.persist(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if events.upserts != 2 {
		t.Fatalf("upserts=%d", events.upserts)
	}
}

func TestSetConfigNilIsNoop(t *testing.T) {
	w := NewIngestWorkerService(testConfig(), testLogger(), &fakeStellarClient{}, &fakeCursorRepo{}, &fakeEventRepo{}, &fakeDerivedRepo{})
	w.SetConfig(nil)
	if w.cfgSnapshot() == nil {
		t.Fatal("config cleared")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
