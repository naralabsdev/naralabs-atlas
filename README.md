# NaraLabs Atlas

**NaraLabs Atlas** is the Soroban event indexer worker for the NaraLabs platform. It polls Stellar RPC (`getEvents`), persists **level-1 raw XDR** and **level-2 generic JSON** to ClickHouse, extracts derived address/token tables, and tracks ingest cursor state in Postgres. Semantic decode (level 3) is handled by `naralabs-api`.

Atlas is deployed as a **separate service** from the NaraLabs API platform.

## Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.25 |
| CLI | Cobra |
| Stellar RPC | `github.com/stellar/go-stellar-sdk` |
| Cursor store | Postgres (`pgx`) |
| Events store | ClickHouse |
| Config | `cleanenv` (root `config/`) |
| Logging | `log/slog` (JSON) |

## Project structure

```
main.go
config/
cmd/
├── root.go
├── worker/
├── replay/
└── backfill/
lib/
├── db, clickhouse, logger
├── scval/ (+ token/, address/)
└── rpcchain/                     # retry, throttle, endpoint pool
internal/
├── client/stellar/               # resilient RPC wrapper
├── client/horizon/               # historical tx lookup for backfill
└── module/ingest/
db/migrations/
├── postgres/
└── clickhouse/
```

## CLI

```bash
./bin/atlas worker                              # live ingest worker
./bin/atlas replay --from-ledger 1000 --to-ledger 2000
./bin/atlas backfill --from-ledger 1 --to-ledger 50000
./bin/atlas --version
```

Send `SIGHUP` to the worker process to hot-reload safe config fields (poll interval, watched contracts, RPC settings).

## Production features (M1.1)

| Feature | Implementation |
|---------|----------------|
| Cold start | `RETENTION_LEDGERS`, `START_LEDGER` (`latest-N` supported) |
| RPC filter batching | Contract filters split into batches of 25 |
| Adaptive poll | Shrinks interval on backlog, grows when caught up |
| SIGHUP reload | Re-reads `.env` / env without restart |
| Token parsing | SEP-41 style hints → `token_events` table |
| Address extraction | Level-2 JSON walk → `event_addresses` table |
| Decoder replay | `atlas replay` re-materializes level-2 JSON |
| Idempotent upsert | ClickHouse `ReplacingMergeTree(ingested_at)` + PG `GREATEST` cursor |
| Query-rich indexes | Bloom/minmax/set skip indexes on events + derived tables |
| RPC resilience | Retry/backoff, throttle, multi-URL failover (`lib/rpcchain`) |
| Reorg rescan | Periodic re-fetch over `REORG_WINDOW` ledgers |
| Backfill | `atlas backfill` with persisted `backfill_state` |

## Quick start

```bash
cp .env.example .env
make docker-up
make docker-logs
```

Local dev:

```bash
docker compose up -d postgres clickhouse
make run
```

## Environment variables

See [`.env.example`](.env.example).

Key variables:

| Variable | Description |
|----------|-------------|
| `RPC_URL` / `RPC_FALLBACK_URLS` | Primary + failover Soroban RPC endpoints |
| `RETENTION_LEDGERS` | Cold-start window when no cursor exists |
| `START_LEDGER` | Override cold start (`12345` or `latest-1000`) |
| `POLL_INTERVAL_MIN/MAX` | Adaptive poll bounds |
| `REORG_WINDOW` | Ledgers to re-scan periodically |
| `HORIZON_URL` | Horizon base URL for backfill metadata |

## Make targets

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/atlas` |
| `make run` | Run worker locally |
| `make test` | Run tests |
| `make docker-up` | Build & start compose stack |
| `make docker-down` | Stop compose stack |

## License

Apache-2.0
