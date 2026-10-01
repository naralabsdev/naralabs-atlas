#!/usr/bin/env bash
# Enable POST /v1/playground/decode on production Atlas (monolith).
# Sets PLAYGROUND_BFF_TOKEN in .env, ensures docker-compose passes it, recreates server.
#
# Usage (from laptop):
#   ssh -i ~/.ssh/server_monolith ubuntu@43.156.234.180 'bash -s' < scripts/set-playground-bff-on-monolith.sh
#
# Then set the SAME value on naralabs-web (Vercel): PLAYGROUND_BFF_TOKEN
set -euo pipefail

ATLAS_DIR="${ATLAS_DIR:-$HOME/naralabs-atlas}"
cd "$ATLAS_DIR"

COMPOSE="$ATLAS_DIR/docker-compose.yml"
ENV_FILE="$ATLAS_DIR/.env"

if [[ ! -f "$COMPOSE" ]]; then
  echo "Missing $COMPOSE" >&2
  exit 1
fi

if ! grep -q 'PLAYGROUND_BFF_TOKEN: \${PLAYGROUND_BFF_TOKEN' "$COMPOSE" 2>/dev/null; then
  if grep -q 'EMAIL_FROM:' "$COMPOSE"; then
    sed -i '/EMAIL_FROM:/a\      PLAYGROUND_BFF_TOKEN: ${PLAYGROUND_BFF_TOKEN:-}' "$COMPOSE"
    echo "Patched docker-compose.yml (PLAYGROUND_BFF_TOKEN passthrough)"
  else
    echo "Could not patch docker-compose.yml — add PLAYGROUND_BFF_TOKEN to server service manually" >&2
    exit 1
  fi
fi

touch "$ENV_FILE"
if grep -q '^PLAYGROUND_BFF_TOKEN=' "$ENV_FILE"; then
  echo "PLAYGROUND_BFF_TOKEN already in .env (unchanged)"
else
  token="$(openssl rand -hex 32)"
  echo "PLAYGROUND_BFF_TOKEN=${token}" >> "$ENV_FILE"
  echo "Generated new PLAYGROUND_BFF_TOKEN in .env"
fi

PLAYGROUND_BFF_TOKEN="$(grep '^PLAYGROUND_BFF_TOKEN=' "$ENV_FILE" | cut -d= -f2- | tr -d '\r')"
if [[ -z "$PLAYGROUND_BFF_TOKEN" ]]; then
  echo "PLAYGROUND_BFF_TOKEN empty in .env" >&2
  exit 1
fi

export PLAYGROUND_BFF_TOKEN

docker compose up -d server --force-recreate

echo "Waiting for health..."
for _ in $(seq 1 30); do
  if docker exec naralabs-atlas-server wget -q -O- http://127.0.0.1:8080/health >/dev/null 2>&1; then
    echo "Atlas server healthy"
    break
  fi
  sleep 2
done

echo ""
echo "=== Next step (required for naralabs.io playground) ==="
echo "Add to Vercel → naralabs-web → Environment Variables:"
echo "  PLAYGROUND_BFF_TOKEN=<same as Atlas .env>"
echo ""
echo "To view Atlas token on this host:"
echo "  grep '^PLAYGROUND_BFF_TOKEN=' $ENV_FILE"
