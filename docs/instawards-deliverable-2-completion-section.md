# Instawards completion report — Deliverable 2 (Google Doc reference)

Use **in-place replacements** in the existing Deliverable 2 block (same headings as D1-style evidence). Primary SOW proof = **k6 / naralabs-perf** on `https://atlas.naralabs.io`; Go `TestDeliverable2` = CI engine only.

**Tracking:** naralabs-atlas `75d2487` · naralabs-perf `0d1b77e` · live API `https://atlas.naralabs.io`

**Overview (SOW paragraph):** k6 measured p95 ~73 ms; 9/9 HTTP fixture checks 100%; ten event types in manifest/fixture set.

**Fixtures block:** naralabs-perf fixtures + k6 scripts (not counter-only Atlas paths as primary).

**Reproduce:** `naralabs-perf` → `./scripts/run-decode-p95.sh`, `./scripts/run-decode-fixtures.sh`.

See live Google Doc Deliverable 2 section for full prose (updated via MCP).
