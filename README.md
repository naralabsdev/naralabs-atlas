# NaraLabs Atlas

**NaraLabs Atlas** is the Soroban data platform backend for NaraLabs: **indexer worker** (write path) + **HTTP read API** (explorer-facing) in one Go repo.

| Command | Role |
|---------|------|
| `atlas worker` | Poll RPC, persist L1 raw XDR + L2 generic JSON, derived tables, cursor |
| `atlas server` | REST API: `/v1/home`, `/v1/stats`, `/v1/events`, `/v1/contracts` |
| `atlas replay` / `atlas backfill` | Maintenance CLIs |

Semantic decode (level 3 via SEP-0048 registry) is planned as an Atlas module in M2 — not a separate repo.

## Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.25 |
| CLI | Cobra (`worker`, `server`, `replay`, `backfill`) |
| HTTP | chi + Huma v2 (OpenAPI 3.1) + Scalar docs |
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
├── worker/          # ingest loop
├── server/          # HTTP API
├── openapi/         # OpenAPI YAML export
├── replay/
└── backfill/
internal/
├── module/
│   ├── ingest/      # write path
│   └── explore/     # read API
├── client/stellar/
└── shared/response/
lib/
├── db, clickhouse, logger
├── scval/ (+ token/, address/)
└── rpcchain/
db/migrations/
```

## CLI

```bash
./bin/atlas worker                              # live ingest worker
./bin/atlas server                              # HTTP API on :8080
./bin/atlas replay --from-ledger 1000 --to-ledger 2000
./bin/atlas backfill --from-ledger 1 --to-ledger 50000
./bin/atlas --version
```

## HTTP API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/v1/home` | Stats + recent events + active contracts |
| GET | `/v1/stats` | Network overview + 14d activity |
| GET | `/v1/events?limit=8` | Recent events (L2 preview) |
| GET | `/v1/contracts?limit=8` | Active contracts |

Query param `network` defaults to `NETWORK` env.

**OpenAPI & docs (dev):**

| URL | Description |
|-----|-------------|
| http://localhost:8080/docs | Scalar interactive API docs (auth token persisted in browser) |
| http://localhost:8080/openapi.yaml | OpenAPI 3.1 spec |
| `./bin/atlas openapi > openapi.yaml` | Export spec to file |

```bash
make run-server
curl http://localhost:8080/v1/home
open http://localhost:8080/docs
```

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

## Quick start (Docker Compose)

Prerequisites: **Docker Desktop** (or Docker Engine + Compose v2).

```bash
git clone https://github.com/naralabsdev/naralabs-atlas.git
cd naralabs-atlas

docker compose up --build -d
docker compose ps
curl http://localhost:8080/health
curl http://localhost:8080/v1/home
```

Semua service (Postgres, ClickHouse, worker, API, CH-UI) jalan dari satu perintah — **tanpa Makefile**. Env default sudah ada di `docker-compose.yml`; override opsional lewat file `.env` atau `docker compose up -e`.

| Service | URL |
|---------|-----|
| API | http://localhost:8080/health |
| OpenAPI docs | http://localhost:8080/docs |
| OpenAPI spec | http://localhost:8080/openapi.yaml |
| CH-UI | http://localhost:3488 |

Stop stack:

```bash
docker compose down
```

Logs:

```bash
docker compose logs -f server atlas
```

## Monitor ClickHouse (CH-UI)

The compose stack includes [CH-UI](https://github.com/caioricciuti/ch-ui) — a web UI for browsing tables, running SQL, and viewing dashboards (similar spirit to phpMyAdmin, but ClickHouse-native).

| Service | URL |
|---------|-----|
| API | http://localhost:8080/health |
| CH-UI | http://localhost:3488 |
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

## Local dev (Go on host, DB in Docker)

Hanya untuk iterasi kode Go tanpa rebuild image:

```bash
docker compose up -d postgres clickhouse
go run . worker
go run . server
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
| Port `5432` / `9000` / `8123` / `8080` already in use | Set custom **expose** ports in `.env` (`POSTGRES_EXPOSE_PORT`, `CLICKHOUSE_HTTP_EXPOSE_PORT`, `CLICKHOUSE_NATIVE_EXPOSE_PORT`, `HTTP_EXPOSE_PORT`, `CH_UI_EXPOSE_PORT`) and update `POSTGRES_URL` / `CLICKHOUSE_URL` / `PUBLISH_URL` port to match. Service ports inside containers stay fixed. Compose binds host ports on `127.0.0.1` only. |
| CH-UI login fails | Use ClickHouse credentials (`atlas` / `atlas` by default), not Postgres. |
| Worker restart loop / migration error | Check `docker logs naralabs-atlas-worker`. Migrations run automatically on startup. |
| No events ingested | Confirm `RPC_URL` is reachable and `WATCHED_CONTRACTS` is empty (index all contracts) or lists valid contract IDs. |

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
| `HTTP_BIND` | Loopback bind address for `./bin/atlas server` (port fixed at `8080`, not `0.0.0.0`) |
| `*_EXPOSE_PORT` | Host ports for Docker Compose only (loopback-only publish; container ports stay fixed) |

## Make targets (optional)

Makefile tersedia untuk CI/dev lokal, tapi **deploy VPS cukup `docker compose` saja**.

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/atlas` |
| `make test` | Run tests |
| `make docker-up` | Alias `docker compose up --build -d` |

## License

Apache-2.0
