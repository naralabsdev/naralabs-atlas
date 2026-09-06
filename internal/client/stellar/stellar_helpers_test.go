package stellar

import (
	"encoding/json"
	"testing"

	rpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
)

func TestBuildFilterBatchesEmpty(t *testing.T) {
	batches := BuildFilterBatches(nil, 25)
	if len(batches) != 1 || batches[0] != nil {
		t.Fatalf("got %v", batches)
	}
}

func TestBuildFilterBatchesSplits(t *testing.T) {
	ids := make([]string, 30)
	for i := range ids {
		ids[i] = "C" + string(rune('A'+i%26))
	}
	batches := BuildFilterBatches(ids, 25)
	if len(batches) != 2 {
		t.Fatalf("got %d batches", len(batches))
	}
	if len(batches[0]) != 25 || len(batches[1]) != 5 {
		t.Fatalf("batch sizes %d %d", len(batches[0]), len(batches[1]))
	}
}

func TestFilterBatchLabel(t *testing.T) {
	if FilterBatchLabel(nil) != "all-contracts" {
		t.Fatal()
	}
	if FilterBatchLabel([]string{"C1"}) != "C1" {
		t.Fatal()
	}
	if FilterBatchLabel([]string{"C1", "C2", "C3"}) != "C1+2" {
		t.Fatal()
	}
}

func TestParseContractIDs(t *testing.T) {
	if ParseContractIDs("  ") != nil {
		t.Fatal()
	}
	got := ParseContractIDs("a, b,,c")
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}

func TestEndpointLabel(t *testing.T) {
	if endpointLabel("https://soroban-testnet.stellar.org") != "soroban-testnet.stellar.org" {
		t.Fatal()
	}
	if endpointLabel("not-a-url") != "not-a-url" {
		t.Fatal()
	}
}

func TestParseCursorLedger(t *testing.T) {
	ledger, err := parseCursorLedger("12345.1.2.3.4")
	if err != nil || ledger != 12345 {
		t.Fatalf("ledger=%d err=%v", ledger, err)
	}
	_, err = parseCursorLedger("bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildEventFilters(t *testing.T) {
	all := buildEventFilters(nil)
	if len(all) != 1 {
		t.Fatalf("got %d", len(all))
	}
	byContract := buildEventFilters([]string{"C1", "C2"})
	if len(byContract) != 2 {
		t.Fatalf("got %d", len(byContract))
	}
}

func TestMapEventAndEventType(t *testing.T) {
	event := mapEvent("testnet", rpc.EventInfo{
		ID:              "evt-1",
		ContractID:      "C1",
		Ledger:          100,
		TransactionHash: "txhash",
		EventType:       rpc.EventTypeContract,
		TopicXDR:        []string{"topic"},
		ValueXDR:        "value",
		TopicJSON:       []json.RawMessage{json.RawMessage(`"transfer"`)},
		ValueJSON:       json.RawMessage(`{"u64":1}`),
	})
	if event.Network != "testnet" || event.EventType != 1 || event.Ledger != 100 {
		t.Fatalf("event=%+v", event)
	}
	if eventTypeToUint32(rpc.EventTypeDiagnostic) != 2 {
		t.Fatal()
	}
	if eventTypeToUint32("unknown") != 0 {
		t.Fatal()
	}
}
