# Instawards completion report — Deliverable 2 section (copy into Google Doc)

Paste under **Scope of Work → Deliverable 2**, after decoder evidence links and **before** Deliverable 3.

---

### SOW completion targets (Deliverable 2)

| SOW target | How it is met |
| --- | --- |
| ≥ 10 event types decoded on fixtures | 10 distinct `eventName` values in `testdata/decoder/manifest.json` (`TestDeliverable2_DecoderSampleFixtures`) |
| ≥ 90% decode success on prepared fixtures | **11/11** expected outcomes (10 decoded + 1 intentional raw fallback) — in-process Go test |
| p95 single decode &lt; 500 ms (demo env) | **Local engine:** Pass (200 iterations, `TestDeliverable2`). **Live HTTP:** p95 **~73 ms** on `POST /v1/decode` (k6, 939 req, 5 VUs, 30s, `https://atlas.naralabs.io`, `transfer` on published testnet registry) |
| Single + batch HTTP API | `POST /v1/decode`, `POST /v1/decode/batch` with Bearer `nl_api_…` |

### Live HTTP benchmark (demo environment)

| Artifact | URL |
| --- | --- |
| k6 repo (scripts + reports) | https://github.com/naralabsdev/naralabs-perf |
| Written benchmark summary (§1–2) | https://github.com/naralabsdev/naralabs-atlas/blob/main/docs/deliverable3-benchmark-report.md |

**How to reproduce:** `go test ./lib/decoder/ -run TestDeliverable2_DecoderSampleFixtures -v` (no server). Live: `naralabs-perf` — `API_KEY`, `BASE_URL=https://atlas.naralabs.io`, `./scripts/run-decode-p95.sh`.

**Tracking (Deliverable 2):** naralabs-atlas commit https://github.com/naralabsdev/naralabs-atlas/commit/75d2487 (fixtures + test); k6 commit https://github.com/naralabsdev/naralabs-perf/commit/0d1b77e; benchmark write-up https://github.com/naralabsdev/naralabs-atlas/commit/da839fa
