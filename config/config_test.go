package config

import (
	"testing"
	"time"
)

func TestParseStartLedgerAbsolute(t *testing.T) {
	got, err := ParseStartLedger("12345", 99999)
	if err != nil {
		t.Fatal(err)
	}
	if got != 12345 {
		t.Fatalf("got %d want 12345", got)
	}
}

func TestParseStartLedgerLatestOffset(t *testing.T) {
	got, err := ParseStartLedger("latest-100", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 4900 {
		t.Fatalf("got %d want 4900", got)
	}
}

func TestParseStartLedgerEmpty(t *testing.T) {
	got, err := ParseStartLedger("", 100)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func TestParseStartLedgerInvalid(t *testing.T) {
	_, err := ParseStartLedger("latest-abc", 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigValidateClampsPollInterval(t *testing.T) {
	cfg := &Config{
		Ingest: IngestConfig{
			PollInterval:    500 * time.Millisecond,
			PollIntervalMin: time.Second,
			PollIntervalMax: 5 * time.Second,
			BatchSize:       100,
			PageLimit:       1000,
		},
		RPC: RPCResilienceConfig{MaxAttempts: 3, RequestsPerSec: 5},
	}
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Ingest.PollInterval != time.Second {
		t.Fatalf("poll interval=%s", cfg.Ingest.PollInterval)
	}
}

func TestConfigValidateRejectsInvalidBatchSize(t *testing.T) {
	cfg := &Config{
		Ingest: IngestConfig{
			PollIntervalMin: time.Second,
			PollIntervalMax: 5 * time.Second,
			BatchSize:       0,
		},
	}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRPCURLsDedupesFallbacks(t *testing.T) {
	cfg := &Config{
		Stellar: StellarConfig{
			RPCURL:       "https://rpc-a.example",
			RPCFallbacks: "https://rpc-b.example, https://rpc-a.example",
		},
	}
	urls := cfg.RPCURLs()
	if len(urls) != 2 {
		t.Fatalf("got %d urls: %v", len(urls), urls)
	}
}

func TestWatchedContractIDs(t *testing.T) {
	cfg := &Config{Ingest: IngestConfig{WatchedContracts: " C1 , C2 , ,C3 "}}
	ids := cfg.WatchedContractIDs()
	if len(ids) != 3 || ids[0] != "C1" {
		t.Fatalf("got %v", ids)
	}
}

func TestConfigValidateDefaultsRPCAndPageLimit(t *testing.T) {
	cfg := &Config{
		Ingest: IngestConfig{
			PollIntervalMin: time.Second,
			PollIntervalMax: 5 * time.Second,
			BatchSize:       100,
			PageLimit:       0,
		},
		RPC: RPCResilienceConfig{MaxAttempts: 0, RequestsPerSec: 0},
	}
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Ingest.PageLimit != 1000 {
		t.Fatalf("page limit=%d", cfg.Ingest.PageLimit)
	}
	if cfg.RPC.MaxAttempts != 1 || cfg.RPC.RequestsPerSec != 10 {
		t.Fatalf("rpc=%+v", cfg.RPC)
	}
}

func TestParseStartLedgerLatestWhenOffsetTooLarge(t *testing.T) {
	got, err := ParseStartLedger("latest-500", 100)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("got %d want 1", got)
	}
}

func TestReloadAndGet(t *testing.T) {
	t.Setenv("RPC_URL", "https://rpc.test")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost:5432/atlas?sslmode=disable")
	t.Setenv("CLICKHOUSE_URL", "clickhouse://localhost:9000/atlas")
	t.Setenv("POLL_INTERVAL_MIN", "1s")
	t.Setenv("POLL_INTERVAL_MAX", "5s")
	t.Setenv("INGEST_BATCH_SIZE", "100")
	t.Setenv("INGEST_PAGE_LIMIT", "1000")

	if err := Reload(); err != nil {
		t.Fatal(err)
	}
	cfg := Get()
	if cfg == nil || cfg.Stellar.RPCURL != "https://rpc.test" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestEnvHelpers(t *testing.T) {
	if !(&Config{Env: "dev"}).IsDevelopment() {
		t.Fatal("expected dev")
	}
	if !(&Config{Env: "production"}).IsProduction() {
		t.Fatal("expected prod")
	}
}
