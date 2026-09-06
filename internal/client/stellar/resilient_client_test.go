package stellar

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rpc "github.com/stellar/go-stellar-sdk/protocols/rpc"

	"github.com/naralabs/naralabs-atlas/config"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      any             `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result"`
	ID      any    `json:"id"`
}

func testResilientConfig(rpcURL string) *config.Config {
	return &config.Config{
		Stellar: config.StellarConfig{RPCURL: rpcURL},
		RPC: config.RPCResilienceConfig{
			MaxAttempts:    2,
			BaseBackoff:    time.Millisecond,
			MaxBackoff:     50 * time.Millisecond,
			RequestsPerSec: 100,
			MaxRetryAfter:  time.Second,
		},
		Ingest: config.IngestConfig{
			PollIntervalMin: time.Second,
			PollIntervalMax: 5 * time.Second,
			BatchSize:       100,
			PageLimit:       1000,
		},
	}
}

func TestNewResilientClient(t *testing.T) {
	client, err := NewResilientClient(testResilientConfig("https://rpc-a.example"), slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("nil client")
	}
}

func TestResilientClientLatestLedger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Method != rpc.GetLatestLedgerMethodName {
			t.Fatalf("method=%s", req.Method)
		}
		_ = json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			Result: rpc.GetLatestLedgerResponse{
				Sequence: 4242,
				Hash:     "hash",
			},
			ID: req.ID,
		})
	}))
	defer server.Close()

	client, err := NewResilientClient(testResilientConfig(server.URL), slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	seq, err := client.LatestLedger(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if seq != 4242 {
		t.Fatalf("sequence=%d", seq)
	}
}

func TestResilientClientFetchEventsSinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Method != rpc.GetEventsMethodName {
			t.Fatalf("method=%s", req.Method)
		}
		_ = json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			Result: rpc.GetEventsResponse{
				Events: []rpc.EventInfo{{
					ID:              "evt-1",
					ContractID:      "CABC",
					Ledger:          100,
					TransactionHash: "txhash",
					EventType:       rpc.EventTypeContract,
					TopicJSON:       []json.RawMessage{json.RawMessage(`"transfer"`)},
					ValueJSON:       json.RawMessage(`{"u64":1}`),
				}},
				LatestLedger: 100,
				Cursor:       "",
			},
			ID: req.ID,
		})
	}))
	defer server.Close()

	client, err := NewResilientClient(testResilientConfig(server.URL), slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	events, nextLedger, err := client.FetchEvents(context.Background(), "testnet", FetchEventsInput{
		StartLedger: 1,
		Limit:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "evt-1" {
		t.Fatalf("events=%+v", events)
	}
	if nextLedger != 100 {
		t.Fatalf("nextLedger=%d", nextLedger)
	}
}

func TestResilientClientFetchEventsInvalidCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			Result:  rpc.GetEventsResponse{LatestLedger: 1},
			ID:      req.ID,
		})
	}))
	defer server.Close()

	client, err := NewResilientClient(testResilientConfig(server.URL), slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = client.FetchEvents(context.Background(), "testnet", FetchEventsInput{
		StartLedger: 1,
		Cursor:      "not-a-valid-cursor",
		Limit:       10,
	})
	if err == nil {
		t.Fatal("expected cursor parse error")
	}
}

func TestResilientClientFetchEventsRespectsLimit(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req jsonRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			Result: rpc.GetEventsResponse{
				Events: []rpc.EventInfo{
					{ID: "e1", ContractID: "C1", Ledger: 1, EventType: rpc.EventTypeContract},
					{ID: "e2", ContractID: "C1", Ledger: 1, EventType: rpc.EventTypeContract},
				},
				LatestLedger: 1,
				Cursor:       "2.0.0.0.0",
			},
			ID: req.ID,
		})
	}))
	defer server.Close()

	client, err := NewResilientClient(testResilientConfig(server.URL), slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	events, _, err := client.FetchEvents(context.Background(), "testnet", FetchEventsInput{
		StartLedger: 1,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len=%d calls=%d", len(events), calls)
	}
	if calls != 1 {
		t.Fatalf("expected single RPC page, calls=%d", calls)
	}
}
