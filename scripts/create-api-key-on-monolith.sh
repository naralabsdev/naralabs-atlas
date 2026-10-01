#!/usr/bin/env bash
# Create an Atlas developer API key (nl_api_…) directly in Postgres on the monolith.
# Matches internal/module/auth/service/api_key_service.go (24-byte entropy, SHA-256 hash).
#
# Run ON the VPS (or via SSH):
#   ./scripts/create-api-key-on-monolith.sh
#
# Optional env:
#   USER_EMAIL=you@naralabs.io   — attach key to this user (default: first user with <3 active keys)
#   API_KEY_LABEL="k6 benchmark" — label column (default: k6-benchmark)
#   PSQL_CMD='docker exec -i naralabs-postgres psql -U atlas -d atlas -v ON_ERROR_STOP=1'
#   WRITE_ENV_FILE=/path/.env  — append API_KEY=… (mode 600); omit to print once to stdout only
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PSQL="${PSQL_CMD:-docker exec -i naralabs-postgres psql -U atlas -d atlas -v ON_ERROR_STOP=1}"
LABEL="${API_KEY_LABEL:-k6-benchmark}"
USER_EMAIL="${USER_EMAIL:-}"

RAW="nl_api_$(openssl rand -hex 24)"
HASH="$(printf '%s' "$RAW" | openssl dgst -sha256 | awk '{print $NF}')"
PREFIX="${RAW:0:15}…"

run_psql() {
  if [[ -n "${POSTGRES_URL:-}" ]]; then
    psql "$POSTGRES_URL" -v ON_ERROR_STOP=1 "$@"
  else
    $PSQL "$@"
  fi
}

USER_ID=""
if [[ -n "$USER_EMAIL" ]]; then
  USER_ID="$(run_psql -t -A -c "SELECT id::text FROM users WHERE lower(email) = lower('${USER_EMAIL//\'/''}') LIMIT 1;")"
  if [[ -z "$USER_ID" ]]; then
    echo "No user found for USER_EMAIL=$USER_EMAIL" >&2
    exit 1
  fi
else
  USER_ID="$(run_psql -t -A -c "
    SELECT u.id::text
    FROM users u
    LEFT JOIN (
      SELECT user_id, COUNT(*) AS n
      FROM api_keys
      WHERE revoked_at IS NULL
      GROUP BY user_id
    ) k ON k.user_id = u.id
    WHERE COALESCE(k.n, 0) < 3
    ORDER BY u.created_at ASC
    LIMIT 1;
  ")"
  if [[ -z "$USER_ID" ]]; then
    echo "No user with fewer than 3 active API keys. Set USER_EMAIL or revoke a key." >&2
    exit 1
  fi
fi

ACTIVE="$(run_psql -t -A -c "SELECT COUNT(*) FROM api_keys WHERE user_id = '$USER_ID'::uuid AND revoked_at IS NULL;")"
if [[ "$ACTIVE" -ge 3 ]]; then
  echo "User $USER_ID already has 3 active API keys (Atlas limit)." >&2
  exit 1
fi

# Escape single quotes for SQL
LABEL_SQL="${LABEL//\'/''}"
HASH_SQL="${HASH//\'/''}"
PREFIX_SQL="${PREFIX//\'/''}"

run_psql -c "
  INSERT INTO api_keys (user_id, token_hash, token_prefix, label)
  VALUES ('$USER_ID'::uuid, '$HASH_SQL', '$PREFIX_SQL', '$LABEL_SQL');
"

echo "Created API key label=\"$LABEL\" for user_id=$USER_ID prefix=$PREFIX" >&2

if [[ -n "${WRITE_ENV_FILE:-}" ]]; then
  umask 077
  if [[ -f "$WRITE_ENV_FILE" ]] && grep -q '^API_KEY=' "$WRITE_ENV_FILE" 2>/dev/null; then
    sed -i.bak "s|^API_KEY=.*|API_KEY=$RAW|" "$WRITE_ENV_FILE"
    rm -f "${WRITE_ENV_FILE}.bak"
  else
    echo "API_KEY=$RAW" >> "$WRITE_ENV_FILE"
  fi
  echo "Wrote API_KEY to $WRITE_ENV_FILE (permissions 600). Full secret not printed." >&2
else
  echo ""
  echo "=== COPY ONCE — store in naralabs-perf .env and GitHub secret NARALABS_API_KEY ==="
  echo "API_KEY=$RAW"
  echo "================================================================================"
fi
