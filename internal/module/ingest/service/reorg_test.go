package service

import (
	"context"
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

func TestMaybeRescanReorgSkipsWhenDisabled(t *testing.T) {
	cfg := testConfig()
	cfg.Ingest.ReorgWindow = 0
	w := NewIngestWorkerService(cfg, testLogger(), &fakeStellarClient{}, &fakeCursorRepo{}, &fakeEventRepo{}, &fakeDerivedRepo{})
	if err := w.maybeRescanReorg(context.Background(), cfg, model.IngestState{LastLedger: 100}, 110); err != nil {
		t.Fatal(err)
	}
}

func TestMaybeRescanReorgFetchesWindow(t *testing.T) {
	cfg := testConfig()
	cfg.Ingest.ReorgWindow = 10
	cfg.Ingest.ReorgInterval = time.Millisecond
	stellarClient := &fakeStellarClient{
		events: []stellar.ContractEvent{{ID: "e1", Ledger: 95}},
	}
	events := &fakeEventRepo{}
	w := NewIngestWorkerService(cfg, testLogger(), stellarClient, &fakeCursorRepo{state: model.IngestState{LastLedger: 100}}, events, &fakeDerivedRepo{})
	if err := w.maybeRescanReorg(context.Background(), cfg, model.IngestState{LastLedger: 100}, 110); err != nil {
		t.Fatal(err)
	}
	if stellarClient.fetchCalls == 0 {
		t.Fatal("expected fetch")
	}
	if events.upserts == 0 {
		t.Fatal("expected upsert")
	}
}
