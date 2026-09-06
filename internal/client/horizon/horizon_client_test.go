package horizon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransactionsByLedgerRangePaginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transactions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"_embedded": {"records": [
				{"hash":"h1","ledger":100,"successful":true,"operation_count":1},
				{"hash":"h2","ledger":101,"successful":true,"operation_count":1}
			]},
			"_links": {"next": {"href": ""}}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	txs, err := client.TransactionsByLedgerRange(context.Background(), 100, 101, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 || txs[0].Hash != "h1" {
		t.Fatalf("got %+v", txs)
	}
}

func TestTransactionsByLedgerRangeStopsAtUpperBound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"_embedded": {"records": [
				{"hash":"h1","ledger":100,"successful":true,"operation_count":1},
				{"hash":"h2","ledger":200,"successful":true,"operation_count":1}
			]},
			"_links": {"next": {"href": ""}}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	txs, err := client.TransactionsByLedgerRange(context.Background(), 100, 150, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 {
		t.Fatalf("got %d txs", len(txs))
	}
}

func TestFetchPageHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.TransactionsByLedgerRange(context.Background(), 1, 2, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}
