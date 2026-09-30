# Deliverable 3 — Benchmark Report (Instawards SOW)

**Project:** NaraLabs Atlas & Explorer  
**Network:** Stellar Soroban testnet (primary)  
**Report date:** 2026-03-30  
**Scope:** SOW completion targets for decode success rate and response time, plus dashboard/API documentation coverage referenced in Deliverable 3 evidence.

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

## 2. Single-event decode latency (local engine)

| Metric | SOW target | Measured | Method |
| --- | --- | --- | --- |
| p95 single decode | < 500 ms (demo env) | **Pass** (local smoke) | 200 iterations on `counter_incremented` fixture in `TestDeliverable2_DecoderSampleFixtures` |

This measures the in-process decode engine (schema match + field extraction) without HTTP or Postgres round-trips. Production `POST /v1/decode` adds network and registry lookup; integrators should expect higher p95 under load but the SOW fixture test validates core decoder performance.

**Reproduce:** same command as §1 (test fails if p95 ≥ 500ms).

---

## 3. Dashboard & documentation (Deliverable 3 functional targets)

| SOW target | Evidence |
| --- | --- |
| Contract search | [naralabs.io](https://naralabs.io) hero search (events, contracts, docs); list filters e.g. `/events?search=` |
| Schema detail view | `/schemas`, contract pages with published event schemas |
| Decode playground | [/playground](https://naralabs.io/playground) — registry-backed contract picker, JSON/XDR input, server-side decode via BFF |
| API docs (≥ 3 core endpoints) | [Atlas API overview](https://naralabs.io/docs/api/overview) documents decode (`POST /v1/decode`), explore (`GET /v1/events`, `GET /v1/contracts`), and registry (`GET /v1/schemas`, publish) |

---

## 4. Production surfaces (reviewer spot-check)

| Surface | URL |
| --- | --- |
| Events explorer | https://naralabs.io/events |
| Contracts explorer | https://naralabs.io/contracts |
| Schema registry UI | https://naralabs.io/schemas |
| Decode playground | https://naralabs.io/playground |
| Developer docs | https://naralabs.io/docs |
| Proxied Atlas health | https://naralabs.io/api/atlas/health |

---

## 5. Demo video (SOW evidence type)

SOW Deliverable 3 lists **Screenshots, Documentation, Demo Video** as evidence. This markdown report covers **benchmark** and **documentation** pointers; the **recorded demo walkthrough** is linked from the Instawards completion report (Google Doc) once the team publishes a public URL (YouTube, Loom, or similar).

---

*Prepared for Instawards / Ambassador review. Decoder numbers are tied to committed fixtures and CI; dashboard URLs reflect production deployment at report time.*
