package service

import (
	"encoding/json"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/lib/scval"
)

func TestMaterializeEventsUsesParser(t *testing.T) {
	events := MaterializeEvents([]stellar.ContractEvent{{
		ID:         "evt-1",
		Network:    "testnet",
		ContractID: "C1",
		Ledger:     10,
		TopicsXDR:  []string{},
		ValueXDR:   "",
		TopicsJSON: []json.RawMessage{json.RawMessage(`"transfer"`)},
		ValueJSON:  json.RawMessage(`null`),
	}}, scval.DefaultParser)

	if len(events) != 1 {
		t.Fatalf("got %d", len(events))
	}
	if events[0].TopicsJSON == "" || events[0].SemanticDecoded {
		t.Fatalf("event=%+v", events[0])
	}
}

func TestExtractDerivedProducesRows(t *testing.T) {
	topics := `["transfer","GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"]`
	value := `{"amount":"10","symbol":"USDC"}`
	addrs, tokens := ExtractDerived([]model.ContractEvent{{
		ID: "evt-1", Network: "testnet", ContractID: "C1", Ledger: 10,
		TopicsJSON: topics,
		ValueJSON:  value,
	}})
	if len(tokens) != 1 {
		t.Fatalf("tokens=%d", len(tokens))
	}
	if len(addrs) == 0 {
		t.Fatal("expected addresses")
	}
}
