package service

import (
	"context"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/client/horizon"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
)

func TestBackfillServiceValidatesRange(t *testing.T) {
	svc := NewBackfillService(
		testConfig(), testLogger(), &fakeStellarClient{}, nil,
		&fakeEventRepo{}, &fakeDerivedRepo{}, &fakeBackfillRepo{},
	)
	if err := svc.Run(context.Background(), 300, 100); err == nil {
		t.Fatal("expected error")
	}
}

func TestBackfillServiceCompletesSmallRange(t *testing.T) {
	cfg := testConfig()
	cfg.Backfill.Interval = 0
	stellarClient := &fakeStellarClient{
		latestLedger: 10,
		nextLedger:   2,
		events: []stellar.ContractEvent{{
			ID: "evt-1", Network: "testnet", ContractID: "C1", Ledger: 1,
		}},
	}
	events := &fakeEventRepo{}
	backfill := &fakeBackfillRepo{}
	svc := NewBackfillService(
		cfg, testLogger(), stellarClient, nil,
		events, &fakeDerivedRepo{}, backfill,
	)
	if err := svc.Run(context.Background(), 1, 2); err != nil {
		t.Fatal(err)
	}
	if !backfill.completed {
		t.Fatal("expected completed")
	}
	if len(backfill.updates) == 0 {
		t.Fatal("expected progress updates")
	}
	if events.upserts == 0 {
		t.Fatal("expected event upserts")
	}
}

func TestBackfillServiceUsesLatestWhenToLedgerZero(t *testing.T) {
	cfg := testConfig()
	cfg.Backfill.Interval = 0
	cfg.Backfill.BatchSize = 100
	stellarClient := &fakeStellarClient{latestLedger: 5}
	backfill := &fakeBackfillRepo{}
	svc := NewBackfillService(
		cfg, testLogger(), stellarClient, nil,
		&fakeEventRepo{}, &fakeDerivedRepo{}, backfill,
	)
	if err := svc.Run(context.Background(), 1, 0); err != nil {
		t.Fatal(err)
	}
	if backfill.state.ToLedger != 5 {
		t.Fatalf("to_ledger=%d", backfill.state.ToLedger)
	}
}

func TestBackfillLedgerWindowHorizonWarningNonFatal(t *testing.T) {
	cfg := testConfig()
	server := horizon.NewClient("http://127.0.0.1:1") // unreachable
	svc := NewBackfillService(
		cfg, testLogger(),
		&fakeStellarClient{events: []stellar.ContractEvent{{ID: "e1", Ledger: 1}}},
		server,
		&fakeEventRepo{}, &fakeDerivedRepo{}, &fakeBackfillRepo{},
	)
	if err := svc.backfillLedgerWindow(context.Background(), 1, 1); err != nil {
		t.Fatal(err)
	}
}
