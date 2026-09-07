// Package config provides environment-based configuration for NaraLabs Atlas.
package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// DefaultHTTPPort is the fixed listen port for the HTTP API (not configurable).
const DefaultHTTPPort = 8080

type LogConfig struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
	JSON  bool   `env:"LOG_JSON" env-default:"true"`
}

type HTTPConfig struct {
	// Bind is the network interface for the HTTP server (default loopback only).
	Bind            string        `env:"HTTP_BIND" env-default:"127.0.0.1"`
	Addr            string        `env:"HTTP_ADDR"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"30s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
	CORSOrigins     string        `env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000,http://127.0.0.1:3000"`
}

type StellarConfig struct {
	Network string `env:"NETWORK" env-default:"testnet"`
	RPCURL  string `env:"RPC_URL" env-required:"true"`
	// Comma-separated fallback RPC URLs (failover order after RPC_URL).
	RPCFallbacks string `env:"RPC_FALLBACK_URLS" env-default:""`
	HorizonURL   string `env:"HORIZON_URL" env-default:"https://horizon-testnet.stellar.org"`
}

type RPCResilienceConfig struct {
	MaxAttempts    int           `env:"RPC_MAX_ATTEMPTS" env-default:"5"`
	BaseBackoff    time.Duration `env:"RPC_BASE_BACKOFF" env-default:"500ms"`
	MaxBackoff     time.Duration `env:"RPC_MAX_BACKOFF" env-default:"30s"`
	RequestsPerSec float64       `env:"RPC_REQUESTS_PER_SEC" env-default:"10"`
	MaxRetryAfter  time.Duration `env:"RPC_MAX_RETRY_AFTER" env-default:"60s"`
}

type DatabaseConfig struct {
	URI string `env:"POSTGRES_URL" env-required:"true"`
}

type ClickHouseConfig struct {
	URI string `env:"CLICKHOUSE_URL" env-required:"true"`
}

type IngestConfig struct {
	PollInterval     time.Duration `env:"POLL_INTERVAL" env-default:"5s"`
	PollIntervalMin  time.Duration `env:"POLL_INTERVAL_MIN" env-default:"1s"`
	PollIntervalMax  time.Duration `env:"POLL_INTERVAL_MAX" env-default:"5s"`
	BatchSize        int           `env:"INGEST_BATCH_SIZE" env-default:"1000"`
	PageLimit        int           `env:"INGEST_PAGE_LIMIT" env-default:"1000"`
	WatchedContracts string        `env:"WATCHED_CONTRACTS" env-default:""`
	RetentionLedgers uint32        `env:"RETENTION_LEDGERS" env-default:"17280"`
	// Absolute ledger or relative offset, e.g. "latest-1000".
	StartLedgerRaw string        `env:"START_LEDGER" env-default:""`
	ReorgWindow    uint32        `env:"REORG_WINDOW" env-default:"12"`
	ReorgInterval  time.Duration `env:"REORG_RESCAN_INTERVAL" env-default:"5m"`
}

type BackfillConfig struct {
	BatchSize int           `env:"BACKFILL_BATCH_SIZE" env-default:"200"`
	Interval  time.Duration `env:"BACKFILL_REQUEST_INTERVAL" env-default:"100ms"`
}

type ReplayConfig struct {
	BatchSize int `env:"REPLAY_BATCH_SIZE" env-default:"500"`
}

// Config holds all runtime configuration for the Atlas worker.
type Config struct {
	ServiceName     string        `env:"SERVICE_NAME" env-default:"naralabs-atlas"`
	Env             string        `env:"ENV" env-default:"dev"`
	PublishURL      string        `env:"PUBLISH_URL"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"10s"`

	Log        LogConfig
	HTTP       HTTPConfig
	Stellar    StellarConfig
	RPC        RPCResilienceConfig
	DB         DatabaseConfig
	ClickHouse ClickHouseConfig
	Ingest     IngestConfig
	Backfill   BackfillConfig
	Replay     ReplayConfig
}

var (
	mu   sync.RWMutex
	conf *Config
)

// Get returns the current configuration snapshot.
func Get() *Config {
	mu.RLock()
	if conf != nil {
		c := conf
		mu.RUnlock()
		return c
	}
	mu.RUnlock()
	if err := Reload(); err != nil {
		panic(fmt.Sprintf("load config: %v", err))
	}
	mu.RLock()
	defer mu.RUnlock()
	return conf
}

// Reload re-reads configuration from .env and environment.
func Reload() error {
	next := &Config{}
	if err := cleanenv.ReadConfig(".env", next); err != nil {
		if err := cleanenv.ReadEnv(next); err != nil {
			return fmt.Errorf("read env config: %w", err)
		}
	}
	if err := next.validate(); err != nil {
		return err
	}
	mu.Lock()
	conf = next
	mu.Unlock()
	return nil
}

func (c *Config) validate() error {
	if c.Ingest.PollIntervalMin <= 0 {
		return fmt.Errorf("POLL_INTERVAL_MIN must be positive")
	}
	if c.Ingest.PollIntervalMax < c.Ingest.PollIntervalMin {
		return fmt.Errorf("POLL_INTERVAL_MAX must be >= POLL_INTERVAL_MIN")
	}
	if c.Ingest.PollInterval < c.Ingest.PollIntervalMin {
		c.Ingest.PollInterval = c.Ingest.PollIntervalMin
	}
	if c.Ingest.PollIntervalMax == 0 {
		c.Ingest.PollIntervalMax = c.Ingest.PollInterval
	}
	if c.Ingest.BatchSize <= 0 {
		return fmt.Errorf("INGEST_BATCH_SIZE must be positive")
	}
	if c.Ingest.PageLimit <= 0 {
		c.Ingest.PageLimit = 1000
	}
	if c.RPC.MaxAttempts < 1 {
		c.RPC.MaxAttempts = 1
	}
	if c.RPC.RequestsPerSec <= 0 {
		c.RPC.RequestsPerSec = 10
	}
	if err := c.resolveHTTP(); err != nil {
		return err
	}
	if strings.TrimSpace(c.PublishURL) == "" {
		c.PublishURL = c.DefaultPublishURL()
	}
	return nil
}

func (c *Config) resolveHTTP() error {
	rawAddr := strings.TrimSpace(c.HTTP.Addr)
	if rawAddr == "" {
		c.HTTP.Addr = net.JoinHostPort(strings.TrimSpace(c.HTTP.Bind), strconv.Itoa(DefaultHTTPPort))
		return nil
	}

	// Legacy ":8080" form — apply HTTP_BIND instead of listening on all interfaces.
	if strings.HasPrefix(rawAddr, ":") {
		port := strings.TrimPrefix(rawAddr, ":")
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("invalid HTTP_ADDR port %q: %w", port, err)
		}
		c.HTTP.Addr = net.JoinHostPort(strings.TrimSpace(c.HTTP.Bind), port)
		return nil
	}

	host, port, err := net.SplitHostPort(rawAddr)
	if err != nil {
		return fmt.Errorf("invalid HTTP_ADDR %q: %w", rawAddr, err)
	}
	if strings.TrimSpace(host) == "" || host == "0.0.0.0" {
		c.HTTP.Addr = net.JoinHostPort(strings.TrimSpace(c.HTTP.Bind), port)
	}
	return nil
}

func (c *Config) DefaultPublishURL() string {
	return fmt.Sprintf("http://localhost:%d", DefaultHTTPPort)
}

func (c *Config) RPCURLs() []string {
	urls := []string{strings.TrimSpace(c.Stellar.RPCURL)}
	for _, part := range strings.Split(c.Stellar.RPCFallbacks, ",") {
		u := strings.TrimSpace(part)
		if u != "" {
			urls = append(urls, u)
		}
	}
	return dedupeStrings(urls)
}

func (c *Config) WatchedContractIDs() []string {
	return parseCSV(c.Ingest.WatchedContracts)
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func ParseStartLedger(raw string, latest uint32) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "latest-") {
		offsetStr := strings.TrimPrefix(strings.ToLower(raw), "latest-")
		offset, err := strconv.ParseUint(offsetStr, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("parse START_LEDGER offset: %w", err)
		}
		if latest <= uint32(offset) {
			return 1, nil
		}
		return latest - uint32(offset), nil
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse START_LEDGER: %w", err)
	}
	return uint32(v), nil
}

func (c *Config) IsDevelopment() bool {
	return strings.HasPrefix(strings.ToLower(c.Env), "dev")
}

func (c *Config) IsProduction() bool {
	return strings.HasPrefix(strings.ToLower(c.Env), "prod")
}

func (c *Config) CORSOriginList() []string {
	raw := strings.TrimSpace(c.HTTP.CORSOrigins)
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
