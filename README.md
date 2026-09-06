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

## Quick start (Docker — recommended)

Prerequisites: **Docker Desktop** (or Docker Engine + Compose v2).

```bash
git clone https://github.com/naralabsdev/naralabs-atlas.git
cd naralabs-atlas

# 1. Copy env template (edit RPC_URL / watched contracts if needed)
cp .env.example .env

# 2. Start Postgres + ClickHouse + Atlas worker
make docker-up

# 3. Tail worker logs — look for "ingest cycle completed"
make docker-logs
```

Verify the stack:

```bash
docker compose ps
curl -s "http://localhost:8123/?user=atlas&password=atlas" --data-binary "SELECT count() FROM events"
docker exec naralabs-atlas-postgres psql -U atlas -d atlas -c "SELECT network, last_ledger FROM ingest_state;"
```

Stop everything:

```bash
make docker-down
```

## Monitor ClickHouse (CH-UI)

The compose stack includes [CH-UI](https://github.com/caioricciuti/ch-ui) — a web UI for browsing tables, running SQL, and viewing dashboards (similar spirit to phpMyAdmin, but ClickHouse-native).

| | |
|---|---|
| URL | http://localhost:3488 |
| Login | ClickHouse user / password from `.env` (default: `atlas` / `atlas`) |
| Database | `atlas` (tables: `events`, `event_addresses`, `token_events`) |

Example queries:

```sql
SELECT count() FROM events;
SELECT network, contract_id, ledger, id FROM events ORDER BY ingested_at DESC LIMIT 20;
SELECT action, count() FROM token_events GROUP BY action;
```

Start only the UI (if ClickHouse is already running):

```bash
docker compose up -d ch-ui
```

## Local dev (worker on host, DB in Docker)

Useful when iterating on Go code without rebuilding the image every time.

```bash
cp .env.example .env

# Start only databases
docker compose up -d postgres clickhouse

# Run worker locally (loads .env automatically)
make run
```

Build the binary without running:

```bash
make build
./bin/atlas worker
./bin/atlas --version
```

## CLI commands

```bash
./bin/atlas worker                              # live ingest worker
./bin/atlas replay --from-ledger 1000 --to-ledger 2000
./bin/atlas backfill --from-ledger 1 --to-ledger 50000
./bin/atlas --version
```

Send `SIGHUP` to the worker process to hot-reload safe config fields (poll interval, watched contracts, RPC settings).

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `clickhouse is unhealthy` on first start | Wait ~30s; compose healthcheck uses `127.0.0.1:8123`. Run `docker compose up -d --force-recreate clickhouse`. |
| Port `5432` / `9000` / `8123` already in use | Stop conflicting services or change host ports in `docker-compose.yml`. |
| CH-UI login fails | Use ClickHouse credentials (`atlas` / `atlas` by default), not Postgres. |
| Worker restart loop / migration error | Check `docker logs naralabs-atlas-worker`. Migrations run automatically on startup. |
| No events ingested | Confirm `RPC_URL` is reachable and `WATCHED_CONTRACTS` is empty (index all contracts) or lists valid contract IDs. |

## Quick start (legacy one-liner)

```bash
cp .env.example .env
make docker-up
make docker-logs
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
| `make test-cover` | Run tests with per-package coverage gate |
| `make docker-up` | Build & start compose stack |
| `make docker-down` | Stop compose stack |

## License

Apache-2.0
