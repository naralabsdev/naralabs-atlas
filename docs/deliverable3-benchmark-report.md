# Deliverable 3 — Benchmark Report (Instawards SOW)

**Project:** NaraLabs Atlas & Explorer  
**Network:** Stellar Soroban testnet (primary)  
**Report date:** 2026-10-01  
**Scope:** SOW completion targets for decode success rate, response time, and Deliverable 3 dashboard/documentation evidence.

---

## 1. Decoder success rate (prepared fixtures)

| Metric | SOW target | Measured | Method |
| --- | --- | --- | --- |
| Fixture outcomes | ≥ 90% expected match | **100%** (11/11) | `TestDeliverable2_DecoderSampleFixtures` on `testdata/decoder/manifest.json` |
| Distinct decoded event types | ≥ 10 (Deliverable 2) | **10** | Same test; 10 `decoded` fixtures + 1 intentional `raw` fallback |

**Reproduce:**

```bash
cd naralabs-atlas
go test ./lib/decoder/ -run TestDeliverable2_DecoderSampleFixtures -v
```

---

## 2. Single-event decode latency

### 2a. Core engine (CI / local — not HTTP)

| Metric | SOW target | Measured | Method |
| --- | --- | --- | --- |
| p95 single decode | < 500 ms (demo env) | **Pass** (local smoke) | 200 iterations on `counter_incremented` in `TestDeliverable2_DecoderSampleFixtures` |

This measures **in-process** `DecodeEvent` only (no HTTP, no Postgres). Use this to validate the decoder library in CI.

**Reproduce:** `go test ./lib/decoder/ -run TestDeliverable2_DecoderSampleFixtures -v`

### 2b. Live API (k6 — recommended for SOW “demo environment”)

| Metric | SOW target | Measured | Method |
| --- | --- | --- | --- |
| p95 `POST /v1/decode` | < 500 ms | Run benchmark; attach `reports/k6-*.json` | [naralabs-perf](https://github.com/naralabsdev/naralabs-perf) — `k6/decode-single.js` |
| Fixture success on HTTP | ≥ 90% checks | Run `k6/decode-fixtures.js` | Same repo; 11 prepared bodies from `testdata/decoder` |

**Reproduce:**

```bash
git clone https://github.com/naralabsdev/naralabs-perf
cd naralabs-perf && cp .env.example .env   # set API_KEY
./scripts/run-decode-p95.sh
./scripts/run-decode-fixtures.sh
```

Requires a developer **API key** (`nl_api_…`). Default target: `https://naralabs.io/api/atlas`.

---

## 3. Dashboard and documentation (Deliverable 3 functional targets)

| SOW target | Evidence (clickable) |
| --- | --- |
| Contract search | [naralabs.io](https://naralabs.io) hero search; list filters e.g. [/events?search=](https://naralabs.io/events) |
| Schema detail view | [Schema registry UI](https://naralabs.io/schemas) and per-contract schema pages |
| Decode playground | [Decode Playground](https://naralabs.io/playground) — registry-backed contract picker, JSON/XDR input, server-side decode via BFF |
| API docs (≥ 3 core endpoint groups) | [Atlas API overview](https://naralabs.io/docs/api/overview) (decode, explore reads, schema registry); [Decode API](https://naralabs.io/docs/api/decode-api); [Playground guide](https://naralabs.io/docs/api/playground) |

---

## 4. Production surfaces (reviewer spot-check)

| Surface | URL |
| --- | --- |
| Events explorer | [naralabs.io/events](https://naralabs.io/events) |
| Contracts explorer | [naralabs.io/contracts](https://naralabs.io/contracts) |
| Schema registry UI | [naralabs.io/schemas](https://naralabs.io/schemas) |
| Decode playground | [naralabs.io/playground](https://naralabs.io/playground) |
| Developer docs | [naralabs.io/docs](https://naralabs.io/docs) |
| Registry summary (testnet) | [GET /v1/schemas/summary?network=testnet](https://naralabs.io/api/atlas/v1/schemas/summary?network=testnet) |
| Proxied Atlas health | [GET /health](https://naralabs.io/api/atlas/health) |

---

## 5. Evidence index (Deliverable 3)

Use these codes in the Instawards completion report (same style as Deliverables 1–2).

| Code | Artifact | URL |
| --- | --- | --- |
| D3-L01 | Web app (explorer + playground) | https://naralabs.io |
| D3-L02 | Frontend repository | https://github.com/naralabsdev/naralabs-web |
| D3-L03 | Events explorer | https://naralabs.io/events |
| D3-L04 | Contracts explorer | https://naralabs.io/contracts |
| D3-L05 | Schema registry UI | https://naralabs.io/schemas |
| D3-L06 | Decode Playground | https://naralabs.io/playground |
| D3-L07 | Product documentation | https://naralabs.io/docs |
| D3-L08 | API overview (≥3 endpoint groups) | https://naralabs.io/docs/api/overview |
| D3-L09 | Decode API reference | https://naralabs.io/docs/api/decode-api |
| D3-L10 | Playground integration doc | https://naralabs.io/docs/api/playground |
| D3-L11 | This benchmark report | https://github.com/naralabsdev/naralabs-atlas/blob/main/docs/deliverable3-benchmark-report.md |
| D3-L11b | k6 HTTP benchmarks (p95 + fixtures) | https://github.com/naralabsdev/naralabs-perf |
| D3-L12 | Demo video | *Add public YouTube/Loom URL in completion report before submission* |

---

## 6. Demo video (SOW evidence type)

SOW Deliverable 3 lists **Screenshots, Documentation, Demo Video** as evidence types. This report covers **benchmark** and **documentation** pointers. The **recorded demo walkthrough** must be linked from the Instawards completion report (Google Doc) as **D3-L12** once published.

---

*Prepared for Instawards / Ambassador review. Decoder numbers are tied to committed fixtures and CI; dashboard URLs reflect production deployment at report time.*
