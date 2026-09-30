#!/usr/bin/env bash
# Demonstrate Deliverable 1 registry operations against a running Atlas server.
# Usage: BASE_URL=http://localhost:8080 ./scripts/registry-demo-operations.sh

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
NETWORK="${NETWORK:-testnet}"

echo "== List schemas (network=${NETWORK}) =="
curl -sS "${BASE_URL}/v1/schemas?network=${NETWORK}&limit=5" | head -c 2000
echo -e "\n"

echo "== Registry summary (SOW check: publishedContracts>=5, eventSchemas>=10) =="
curl -sS "${BASE_URL}/v1/schemas/summary?network=${NETWORK}"
echo -e "\n"

echo "== List schema contracts =="
curl -sS "${BASE_URL}/v1/schemas/contracts?network=${NETWORK}" | head -c 2000
echo -e "\n"

SAMPLE_CONTRACT="CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4"
echo "== Retrieve schemas for sample contract ${SAMPLE_CONTRACT} =="
curl -sS "${BASE_URL}/v1/schemas/${SAMPLE_CONTRACT}?network=${NETWORK}"
echo -e "\n"

echo "== Retrieve single event schema counter_incremented =="
curl -sS "${BASE_URL}/v1/schemas/${SAMPLE_CONTRACT}/counter_incremented?network=${NETWORK}"
echo -e "\n"

if [[ -n "${PUBLISH_TOKEN:-}" ]]; then
  echo "== Publish sample event (requires PUBLISH_TOKEN=nl_live_…) =="
  SCHEMA_BODY="$(cat testdata/registry/samples/schemas/counter_incremented.json)"
  curl -sS -X POST "${BASE_URL}/v1/schemas" \
    -H "Authorization: Bearer ${PUBLISH_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$(jq -n \
      --arg cid "${SAMPLE_CONTRACT}" \
      --arg net "${NETWORK}" \
      --argjson body "${SCHEMA_BODY}" \
      '{contractId:$cid,network:$net,eventName:"counter_incremented",schemaBody:$body,author:"NaraLabs demo"}')"
  echo -e "\n"
else
  echo "(Skip publish: set PUBLISH_TOKEN to run POST /v1/schemas demo)"
fi
