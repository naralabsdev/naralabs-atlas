# Instawards completion report — Deliverable 3 section (copy into Google Doc)

Use this block under **Scope of Work → Deliverable 3** in the Instawards completion Google Doc.

**Formatting (Google Docs):**

1. Paste with **Ctrl+Shift+V** (paste without formatting) so link colors inherit the doc theme.
2. Apply **Heading 2** to the line `Deliverable 3: Explorer Dashboard & Developer Documentation`.
3. Apply **Heading 3** to `Evidence provided` and `SOW completion targets`.
4. Select each `https://…` URL and use **Insert → Link** only if it did not auto-link.
5. Use **black / dark gray** body text (`#000000` / `#171717`); avoid copying blue link styles from Deliverable 1 tables.

---

## Deliverable 3: Explorer Dashboard & Developer Documentation

**Deliverable:** Lightweight dashboard (contract search, schema views, decode playground) and integration documentation for schema publishing and API usage.

**Evidence type:** Screenshots, documentation, demo video (SOW §6.1).

**Status:** Completed (live product, docs, benchmark report; demo video URL in D3-L12 when published).

**Tracking:** naralabs-web commit a6861f2 — https://github.com/naralabsdev/naralabs-web/commit/a6861f2  
Benchmark report (Atlas) commit da839fa — https://github.com/naralabsdev/naralabs-atlas/commit/da839fa  
k6 HTTP benchmarks — https://github.com/naralabsdev/naralabs-perf (commit 0d1b77e)

**Repository:** https://github.com/naralabsdev/naralabs-web

### Overview

The Next.js app at https://naralabs.io provides hero search, Events and Contracts explorers, the SEP-0048 schema registry UI, Decode Playground at /playground, Nextra developer docs (decode, explore, registry API groups), and developer API keys. The playground uses the same decode path as integrators via a server-side BFF to POST /v1/playground/decode (no end-user API key in the browser).

### Evidence provided

| SOW ask | Evidence |
| --- | --- |
| Contract search | https://naralabs.io — hero search; https://naralabs.io/events |
| Schema detail view | https://naralabs.io/schemas |
| Decode playground | https://naralabs.io/playground |
| API documentation (≥3 core endpoint groups) | https://naralabs.io/docs/api/overview — decode, explore reads, schema registry |
| Benchmark report (success rate + latency) | https://github.com/naralabsdev/naralabs-atlas/blob/main/docs/deliverable3-benchmark-report.md (k6 p95 ~73ms; 9/9 HTTP fixtures; Go 11/11) |
| Explorer source (sample) | https://github.com/naralabsdev/naralabs-web/tree/main/src/app/(playground)/playground |
| Demo video | D3-L12 — paste public video URL here before ambassador submission |

### SOW completion targets (Deliverable 3)

| SOW target | How it is met |
| --- | --- |
| Contract search | Hero search + explorer list filters |
| Schema detail view | /schemas UI and contract/event schema pages |
| Decode playground | /playground (testnet, registry-backed contracts) |
| API documentation | /docs/api/overview, /docs/api/decode-api, /docs/api/playground |
| Benchmark report | deliverable3-benchmark-report.md; ties to TestDeliverable2 locally |

### Evidence index (L-codes)

| Code | Artifact | URL |
| --- | --- | --- |
| D3-L01 | Web app | https://naralabs.io |
| D3-L02 | Frontend repo | https://github.com/naralabsdev/naralabs-web |
| D3-L06 | Playground | https://naralabs.io/playground |
| D3-L07 | Docs home | https://naralabs.io/docs |
| D3-L08 | API overview | https://naralabs.io/docs/api/overview |
| D3-L11 | Benchmark report | https://github.com/naralabsdev/naralabs-atlas/blob/main/docs/deliverable3-benchmark-report.md |
| D3-L12 | Demo video | *(pending public URL)* |

**How to reproduce:** Open https://naralabs.io (Events, Contracts, Schemas, Playground, /docs). Benchmark: `go test ./lib/decoder/ -run TestDeliverable2 -v` in naralabs-atlas.
