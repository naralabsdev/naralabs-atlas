#!/usr/bin/env bash
# Apply promote-community-schemas-to-verified.sql to Postgres (production monolith by default).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PSQL="${PSQL_CMD:-docker exec -i naralabs-postgres psql -U atlas -d atlas -v ON_ERROR_STOP=1}"

echo "== Before (API spot-check recommended: GET /v1/schemas/summary?network=testnet) =="
curl -sS "${ATLAS_PUBLIC_URL:-https://naralabs.io/api/atlas}/v1/schemas/summary?network=testnet" 2>/dev/null || true
echo ""

echo "== Running SQL on Postgres =="
if [[ -n "${POSTGRES_URL:-}" ]]; then
  psql "$POSTGRES_URL" < "$ROOT/scripts/promote-community-schemas-to-verified.sql"
else
  $PSQL < "$ROOT/scripts/promote-community-schemas-to-verified.sql"
fi

echo ""
echo "== After =="
curl -sS "${ATLAS_PUBLIC_URL:-https://naralabs.io/api/atlas}/v1/schemas/summary?network=testnet" 2>/dev/null || true
echo ""
