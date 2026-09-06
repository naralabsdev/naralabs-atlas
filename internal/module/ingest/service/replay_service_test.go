package service

import (
	"context"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

func TestReplayServiceValidatesRange(t *testing.T) {
	svc := NewReplayService(testConfig(), testLogger(), &fakeEventRepo{}, &fakeDerivedRepo{})
	if err := svc.Run(context.Background(), 200, 100); err == nil {
		t.Fatal("expected range error")
	}
}

func TestReplayServiceRematerializesEvents(t *testing.T) {
	cfg := testConfig()
	events := &fakeEventRepo{
		listResult: []model.ContractEvent{{
			ID: "evt-1", Network: "testnet", ContractID: "C1", Ledger: 100,
			TopicsXDR: []string{}, ValueXDR: "",
		}},
	}
	derived := &fakeDerivedRepo{}
	svc := NewReplayService(cfg, testLogger(), events, derived)
	if err := svc.Run(context.Background(), 100, 100); err != nil {
		t.Fatal(err)
	}
	if events.upserts != 1 {
		t.Fatalf("upserts=%d", events.upserts)
	}
}

func TestRematerializePreservesIdentity(t *testing.T) {
	in := []model.ContractEvent{{
		ID: "evt-1", Network: "testnet", ContractID: "C1", Ledger: 5,
		TopicsXDR: []string{"xdr-topic"}, ValueXDR: "",
	}}
	out := rematerialize(in)
	if out[0].ID != in[0].ID || out[0].Ledger != 5 {
		t.Fatalf("out=%+v", out[0])
	}
	if out[0].TopicsJSON == "" {
		t.Fatal("expected topics json")
	}
}
