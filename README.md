# NaraLabs Atlas

**NaraLabs Atlas** is the Soroban data platform backend for NaraLabs: **indexer worker** (write path) + **HTTP read/write API** (explorer, auth, schema registry) in one Go repo.

| Command | Role |
|---------|------|
| `atlas worker` | Poll RPC, persist L1 raw XDR + L2 generic JSON, derived tables, cursor |
| `atlas server` | REST API: explore, auth, SEP-0048 schema registry |
| `atlas replay` / `atlas backfill` | Maintenance CLIs |
| `atlas openapi` | Export OpenAPI 3.1 spec |

**Current version:** `0.3.0`

Ingest today decodes events to **level 2** (tagged JSON via `scval`). The **schema registry** stores SEP-0048 definitions; **semantic decode (level 3)** against those schemas is the next pipeline step (registry CRUD is live, decode worker hookup planned).

## Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.25 |
| CLI | Cobra (`worker`, `server`, `replay`, `backfill`, `openapi`) |
| HTTP | chi + Huma v2 (OpenAPI 3.1) + Scalar docs |
| Stellar RPC | `github.com/stellar/go-stellar-sdk` |
| Cursor + auth + registry | Postgres (`pgx`) |
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
│   ├── ingest/      # write path (worker)
│   ├── explore/     # read API (home, events, contracts)
│   ├── auth/        # register, login, email verification, JWT
│   └── registry/    # SEP-0048 event schema registry (Postgres)
├── client/stellar/
└── shared/response/
lib/
├── db, clickhouse, logger
├── scval/ (+ token/, address/)
└── rpcchain/
db/migrations/
├── postgres/        # users, event_schemas, cursor, backfill_state
└── clickhouse/      # events, token_events, event_addresses
```

## CLI

```bash
./bin/atlas worker                              # live ingest worker
./bin/atlas server                              # HTTP API on :8080
./bin/atlas replay --from-ledger 1000 --to-ledger 2000
./bin/atlas backfill --from-ledger 1 --to-ledger 50000
./bin/atlas openapi > openapi.yaml              # export OpenAPI spec
./bin/atlas --version                           # 0.3.0
```

Send `SIGHUP` to the worker process to hot-reload safe config fields (poll interval, watched contracts, RPC settings).

## HTTP API

Query param `network` defaults to `NETWORK` env (`testnet`, `mainnet`, `futurenet`).

### Explore (public read)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/v1/home` | Stats + recent events + active contracts |
| GET | `/v1/stats` | Network overview + 14d activity |
| GET | `/v1/events` | Paginated Soroban events |
| GET | `/v1/events/{id}` | Event detail (topics, value, XDR) |
| GET | `/v1/contracts` | Paginated active contracts |
| GET | `/v1/contracts/{id}` | Contract detail + event type breakdown |
| GET | `/v1/contracts/{id}/events` | Paginated events for one contract |

**Events / contract events pagination:** `page` (default 1), `page_size` (default 20, max 100), `search`, `event_type`, `decode_status` (`decoded` \| `raw`).

**Contracts pagination:** `page`, `page_size`, `search`, `schema_status` (`decoded` \| `raw_only`).

Response shape: `{ "items": [...], "total": N, "page": 1, "page_size": 20 }`.

### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/v1/auth/register` | — | Create account + send verification email |
| POST | `/v1/auth/login` | — | Login (verified email required) → JWT |
| POST | `/v1/auth/verify-email` | — | Consume verification token → JWT |
| POST | `/v1/auth/resend-verification` | — | Resend verification link |
| GET | `/v1/auth/me` | Bearer JWT | Current user profile |

Used by [naralabs-web](https://github.com/naralabsdev/naralabs-web) login/register flows (proxied via Next.js `/api/auth/*`).

### Schema registry (SEP-0048)

Stores versioned event schema definitions in Postgres (`event_schemas`). Contract authors publish schemas so the explorer can show **decoded** vs **raw** status once semantic decode is wired.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/v1/schemas` | — | Publish new schema version (auto-increment) |
| GET | `/v1/schemas` | — | List schemas (`network`, `search`, `limit`, `offset`) |
| GET | `/v1/schemas/{contract_id}` | — | Latest schema per event on a contract |
| GET | `/v1/schemas/{contract_id}/{event_name}` | — | Get schema (`?version=` optional) |

**Publish example:**

```bash
curl -X POST http://localhost:8080/v1/schemas \
  -H 'Content-Type: application/json' \
  -d '{
    "contractId": "C...",
    "network": "testnet",
    "eventName": "transfer",
    "schemaBody": {
      "name": "transfer",
      "args": [
        { "name": "from", "type": "address" },
        { "name": "to", "type": "address" },
        { "name": "amount", "type": "i128" }
      ]
    },
    "author": "Your Team"
  }'
```

**Schema body rules (SEP-0048):** JSON object with `name` (must match `eventName`) and non-empty `args` array of `{ "name", "type" }`.

**List example:**

```bash
curl 'http://localhost:8080/v1/schemas?network=testnet&search=CABC&limit=20&offset=0'
```

Registry pagination uses `limit` / `offset` (not `page` / `page_size`).

### OpenAPI & docs (dev)

| URL | Description |
|-----|-------------|
| http://localhost:8080/docs | Scalar interactive API docs (Bearer token persisted in browser) |
| http://localhost:8080/openapi.yaml | OpenAPI 3.1 spec |
| `./bin/atlas openapi > openapi.yaml` | Export spec to file |

```bash
make run-server
curl http://localhost:8080/v1/home
open http://localhost:8080/docs
```

## Production features

### M1.1 — ingest & explore

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

### v0.3 — auth, registry, extended explore

| Feature | Implementation |
|---------|----------------|
| Auth module | Register, login, email verification (Resend), JWT, `/me` |
| Schema registry | SEP-0048 publish/list/get in Postgres (`event_schemas`) |
| Event & contract detail | `/v1/events/{id}`, `/v1/contracts/{id}`, `/v1/contracts/{id}/events` |
| Decode filters | `decode_status` on events, `schema_status` on contracts |
| CORS | `CORS_ALLOWED_ORIGINS` for frontend origin |
| OpenAPI 0.3 + Scalar | Interactive docs with auth token persistence |

**Planned next:** wire registry schemas into ingest worker for level-3 `semantic_decoded` on `events`.

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
| CH-UI | http://localhost:3488 |
| Login | ClickHouse user / password from `.env` (default: `atlas` / `atlas`) |
| Database | `atlas` |

**ClickHouse tables:** `events`, `event_addresses`, `token_events`

**Postgres tables (auth + registry):** `users`, `email_verification_tokens`, `event_schemas`

Example queries:

```sql
-- ClickHouse
SELECT count() FROM events;
SELECT network, contract_id, ledger, id FROM events ORDER BY ingested_at DESC LIMIT 20;
SELECT action, count() FROM token_events GROUP BY action;
```

```sql
-- Postgres (via psql or any PG client)
SELECT contract_id, event_name, version, author FROM event_schemas ORDER BY created_at DESC LIMIT 20;
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

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `clickhouse is unhealthy` on first start | Wait ~30s; compose healthcheck uses `127.0.0.1:8123`. Run `docker compose up -d --force-recreate clickhouse`. |
| Port `5432` / `9000` / `8123` / `8080` already in use | Set custom **expose** ports in `.env` (`POSTGRES_EXPOSE_PORT`, `CLICKHOUSE_HTTP_EXPOSE_PORT`, `CLICKHOUSE_NATIVE_EXPOSE_PORT`, `HTTP_EXPOSE_PORT`, `CH_UI_EXPOSE_PORT`) and update `POSTGRES_URL` / `CLICKHOUSE_URL` / `PUBLISH_URL` port to match. Service ports inside containers stay fixed. Compose binds host ports on `127.0.0.1` only. |
| CH-UI login fails | Use ClickHouse credentials (`atlas` / `atlas` by default), not Postgres. |
| Worker restart loop / migration error | Check `docker logs naralabs-atlas-worker`. Migrations run automatically on startup. |
| No events ingested | Confirm `RPC_URL` is reachable and `WATCHED_CONTRACTS` is empty (index all contracts) or lists valid contract IDs. |
| Auth emails not sent | Set `RESEND_API_KEY`; without it, verification links are logged only. |
| CORS errors from frontend | Add frontend origin to `CORS_ALLOWED_ORIGINS`. |

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
| `HTTP_BIND` | Bind address for `./bin/atlas server` (`127.0.0.1` local, `0.0.0.0` in Docker) |
| `CORS_ALLOWED_ORIGINS` | Comma-separated frontend origins |
| `PUBLISH_URL` | Base URL in OpenAPI spec |
| `AUTH_JWT_SECRET` | JWT signing secret |
| `AUTH_JWT_EXPIRY` | JWT lifetime (default `168h`) |
| `WEB_APP_URL` | Base URL for email verification links |
| `RESEND_API_KEY` / `EMAIL_FROM` | Transactional email (Resend) |
| `*_EXPOSE_PORT` | Host ports for Docker Compose only (loopback-only publish) |

Registry uses `POSTGRES_URL` + `NETWORK` (no separate env block).

## Make targets (optional)

Makefile tersedia untuk CI/dev lokal, tapi **deploy VPS cukup `docker compose` saja**.

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/atlas` |
| `make test` | Run tests |
| `make run-server` | Start HTTP API locally |
| `make docker-up` | Alias `docker compose up --build -d` |

## Frontend integration

[naralabs-web](https://github.com/naralabsdev/naralabs-web) proxies Atlas via `/api/atlas/*` (server-only `ATLAS_API_URL`). Auth uses `/api/auth/*` route handlers. Explorer UI consumes explore endpoints; schema **status** (`decoded` / `raw`) is shown in contract lists — schema **registration** is via Atlas `POST /v1/schemas` (see OpenAPI docs).

## License

Apache-2.0
