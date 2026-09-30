#!/usr/bin/env bash
# Demonstrate Deliverable 2 (Event Decoder API) against a running Atlas server.
# Usage: BASE_URL=http://localhost:8080 API_KEY=nl_dev_… ./scripts/decoder-demo-operations.sh

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
NETWORK="${NETWORK:-testnet}"
CONTRACT="${DECODE_CONTRACT:-CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4}"

if [[ -z "${API_KEY:-}" ]]; then
  echo "Set API_KEY (developer key from dashboard) to run decode demos."
  exit 1
fi

TOPICS="$(cat testdata/decoder/counter_incremented.topics.json)"
VALUE="$(cat testdata/decoder/counter_incremented.value.json)"

echo "== POST /v1/decode (single event, registry-backed) =="
curl -sS -X POST "${BASE_URL}/v1/decode" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg net "${NETWORK}" \
    --arg cid "${CONTRACT}" \
    --arg ev "counter_incremented" \
    --argjson topics "${TOPICS}" \
    --argjson value "${VALUE}" \
    '{network:$net,contractId:$cid,eventName:$ev,topics:$topics,value:$value}')" | head -c 2500
echo -e "\n"

echo "== POST /v1/decode/batch (2 identical events) =="
curl -sS -X POST "${BASE_URL}/v1/decode/batch" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg net "${NETWORK}" \
    --arg cid "${CONTRACT}" \
    --arg ev "counter_incremented" \
    --argjson topics "${TOPICS}" \
    --argjson value "${VALUE}" \
    '{network:$net,items:[{contractId:$cid,eventName:$ev,topics:$topics,value:$value},{contractId:$cid,eventName:$ev,topics:$topics,value:$value}]}')" | head -c 2500
echo -e "\n"

echo "== POST /v1/decode (unknown contract → raw fallback) =="
curl -sS -X POST "${BASE_URL}/v1/decode" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n \
    --arg net "${NETWORK}" \
    --argjson topics "${TOPICS}" \
    --argjson value "${VALUE}" \
    '{network:$net,contractId:"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",topics:$topics,value:$value}')" | head -c 1500
echo -e "\n"

echo "== OpenAPI / Scalar (decoder tag) =="
echo "${BASE_URL}/docs"
echo "(Operations: decode-event, decode-events-batch; auth: API key)"

echo "== Local SOW acceptance (no server required) =="
echo "go test ./lib/decoder/ -run TestDeliverable2_DecoderSampleFixtures -v"
