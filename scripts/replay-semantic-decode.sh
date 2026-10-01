#!/usr/bin/env bash
# Re-apply level-3 semantic decode to events already stored in ClickHouse.
# Requires Postgres (schema registry) + ClickHouse, same env as atlas worker.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

FROM="${1:-0}"
TO="${2:-0}"

echo "ClickHouse migration + semantic replay from ledger ${FROM} to ${TO:-latest/open}"
exec go run . replay --from-ledger "$FROM" --to-ledger "$TO"
