# SEP-0048-aligned event schema format (NaraLabs Atlas registry)

This document describes the **event schema body** stored in the NaraLabs Atlas schema registry (`event_schemas.schema_body` in Postgres). It operationalizes [SEP-0048 Contract Interface Specification](https://github.com/stellar/stellar-protocol/blob/master/core/sep-0048.md) event definitions for Soroban contracts.

## Scope (Deliverable 1)

The registry supports **publish**, **list**, and **retrieve** by:

| Key | HTTP API |
| --- | --- |
| Network | Query `network` (default from server `NETWORK` env) |
| Contract ID | Path `/v1/schemas/{contract_id}` or publish body `contractId` |
| Event name | Path `/v1/schemas/{contract_id}/{event_name}` |
| Schema version | Query `?version=` on get; auto-increment on publish |

OpenAPI: run `./bin/atlas server` and open `/docs`, or export with `./bin/atlas openapi`.

## Schema body JSON shape

Each published row is one **event type** on one **contract** and **network**. The body is JSON with:

| Field | Required | Description |
| --- | --- | --- |
| `name` | Yes | Event name; must match `eventName` in `POST /v1/schemas` |
| `args` | One of `args` / `params` | SEP-0048-style positional arguments |
| `params` | One of `args` / `params` | NaraLabs extended form with `location` (`topic`, `data`, …) |
| `prefix_topics` | Recommended | Symbol topics that identify the event before decoding fields |
| `data_format` | Optional | e.g. `single_value` when payload is one ScVal in data |

### Minimal SEP-0048-style example (`args`)

```json
{
  "name": "transfer",
  "args": [
    { "name": "from", "type": "address" },
    { "name": "to", "type": "address" },
    { "name": "amount", "type": "i128" }
  ]
}
```

### Recommended Soroban example (`prefix_topics` + `params`)

Most contracts emit a **topic prefix** (symbols) plus typed data. NaraLabs accepts:

```json
{
  "name": "counter_incremented",
  "prefix_topics": ["cntr", "incr"],
  "data_format": "single_value",
  "params": [
    { "name": "count", "type": "u32", "location": "data" }
  ]
}
```

### Supported scalar types (non-exhaustive)

`u32`, `u64`, `i128`, `address`, `symbol`, `string`, `bool`, and other ScVal types used in Soroban events.

Validation is implemented in `internal/module/registry/service/validate.go`.

## Authoring YAML (CLI)

Developers typically author **`naralabs.schema.yaml`** and publish with `@naralabs/cli`:

```bash
naralabs init my-contract
# edit naralabs.schema.yaml
naralabs registry publish
```

See `testdata/registry/samples/sample-counter.naralabs.schema.yaml` and `naralabs-cli` README.

## Sample contract catalog (Instawards fixtures)

Committed samples for reviewers and tests:

| Path | Contents |
| --- | --- |
| `testdata/registry/samples/manifest.json` | **5 contracts**, **12 event types** (exceeds SOW minimum 5 / 10) |
| `testdata/registry/samples/schemas/*.json` | Per-event schema bodies |
| `testdata/registry/samples/sample-counter.naralabs.schema.yaml` | Multi-event YAML example |

Acceptance test: `TestDeliverable1_RegistrySampleCatalog` in `internal/module/registry/service/schema_registry_sow_test.go`.

## API operations (demonstration)

### Publish (publish token)

```http
POST /v1/schemas
Authorization: Bearer nl_live_…
Content-Type: application/json

{
  "contractId": "CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4",
  "network": "testnet",
  "eventName": "counter_incremented",
  "schemaBody": { "...": "see testdata/registry/samples/schemas/counter_incremented.json" },
  "author": "NaraLabs"
}
```

### List

```http
GET /v1/schemas?network=testnet&limit=50&offset=0
GET /v1/schemas/contracts?network=testnet
```

### Retrieve

```http
GET /v1/schemas/{contract_id}
GET /v1/schemas/{contract_id}/{event_name}
GET /v1/schemas/{contract_id}/{event_name}/versions
GET /v1/schemas/events/{id}
```

### Live completion check (deployed registry)

```http
GET /v1/schemas/summary?network=testnet
```

SOW targets: `publishedContracts >= 5`, `eventSchemas >= 10` on the demo environment.

## Related code

| Component | Location |
| --- | --- |
| Registry service | `internal/module/registry/service/schema_service.go` |
| HTTP handlers | `internal/module/registry/handler/schema_handler.go` |
| Routes | `internal/module/registry/routes/routes.go` |
| Postgres migration | `db/migrations/postgres/004_event_schemas.up.sql` |
