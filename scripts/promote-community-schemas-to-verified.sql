-- Promote all published community event_schemas to verified (testnet catalog).
-- Run on production Postgres via monolith, e.g.:
--   PSQL_CMD='docker exec naralabs-postgres psql -U atlas -d atlas' ./scripts/promote-community-schemas-to-verified.sh
--
-- Uses Stellar null-account placeholder for verified_wallet when unset (display-only bulk ops).
-- For wallet-backed verification, use POST /v1/schemas/verify instead.

BEGIN;

SELECT trust_tier, status, network, COUNT(*) AS n
FROM event_schemas
GROUP BY trust_tier, status, network
ORDER BY network, trust_tier, status;

UPDATE event_schemas
SET trust_tier = 'verified',
    verified_at = COALESCE(verified_at, NOW()),
    verified_wallet = COALESCE(
        NULLIF(BTRIM(verified_wallet), ''),
        'GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF'
    ),
    updated_at = NOW()
WHERE status = 'published'
  AND trust_tier = 'community'
  AND network = 'testnet';

SELECT trust_tier, status, network, COUNT(*) AS n
FROM event_schemas
GROUP BY trust_tier, status, network
ORDER BY network, trust_tier, status;

COMMIT;
