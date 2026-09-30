#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/testdata/registry/samples"
SCHEMAS="$OUT/schemas"
MANIFEST="$OUT/manifest.json"

PSQL="${PSQL_CMD:-docker exec -i naralabs-postgres psql -U atlas -d atlas -t -A}"

mkdir -p "$SCHEMAS"
rm -f "$SCHEMAS"/*.json

TMP="$(mktemp)"
$PSQL -c "
SELECT json_build_object(
  'contractId', contract_id,
  'eventName', event_name,
  'schemaBody', schema_body,
  'author', author
)::text
FROM event_schemas
WHERE status = 'published' AND network = 'testnet'
ORDER BY contract_id, event_name;
" > "$TMP"

python3 - <<PY
import json, pathlib, collections

root = pathlib.Path("$OUT")
schemas_dir = root / "schemas"
rows = []
for line in open("$TMP"):
    line = line.strip()
    if not line:
        continue
    rows.append(json.loads(line))

by_contract = collections.OrderedDict()
for row in rows:
    cid = row["contractId"]
    ename = row["eventName"]
    author = row.get("author") or ""
    fname = f"{ename}__{cid[:8]}.json"
    (schemas_dir / fname).write_text(json.dumps(row["schemaBody"], indent=2) + "\n")
    entry = by_contract.setdefault(cid, {"contractId": cid, "label": author, "events": []})
    entry["label"] = author
    entry["events"].append({"eventName": ename, "schemaFile": fname})

contracts = list(by_contract.values())
event_count = sum(len(c["events"]) for c in contracts)
manifest = {
    "description": "Exported from production Postgres (published community catalog). Re-run scripts/export-registry-manifest.sh on monolith.",
    "network": "testnet",
    "source": "production_postgres_export",
    "sow_minimum_contracts": 5,
    "sow_minimum_event_types": 10,
    "exported_contracts": len(contracts),
    "exported_event_types": event_count,
    "contracts": contracts,
}
(root / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
print(f"Wrote {len(contracts)} contracts, {event_count} events")
PY

rm -f "$TMP"
