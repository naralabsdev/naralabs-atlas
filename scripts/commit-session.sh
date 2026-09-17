#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

commit() {
  git add "$@"
  if git diff --cached --quiet; then
    echo "skip (empty): $MSG"
    return 0
  fi
  git commit -m "$MSG"
}

MSG='chore: update env example with auth reset and email settings' && commit .env.example
MSG='chore: wire auth env vars in docker compose' && commit docker-compose.yml
MSG='docs: update README for auth registry and decoder features' && commit README.md
MSG='chore: update go module dependencies' && commit go.mod go.sum

MSG='feat(db): add schema registry publish tokens migration' && commit db/migrations/postgres/005_schema_registry_tokens.up.sql db/migrations/postgres/005_schema_registry_tokens.down.sql
MSG='feat(db): add schema projects migration' && commit db/migrations/postgres/006_schema_projects.up.sql db/migrations/postgres/006_schema_projects.down.sql
MSG='feat(db): add schema project contracts migration' && commit db/migrations/postgres/007_schema_project_contracts.up.sql db/migrations/postgres/007_schema_project_contracts.down.sql
MSG='feat(db): add API keys migration' && commit db/migrations/postgres/008_api_keys.up.sql db/migrations/postgres/008_api_keys.down.sql
MSG='feat(db): add password reset tokens migration' && commit db/migrations/postgres/009_password_reset.up.sql db/migrations/postgres/009_password_reset.down.sql

MSG='feat(config): add auth reset TTL and resend cooldown settings' && commit config/config.go
MSG='feat(email): add password reset email template' && commit lib/email/resend.go

MSG='feat(auth): extend auth repository for password flows' && commit internal/module/auth/repository/auth_repository.go
MSG='feat(auth): add change forgot and reset password service methods' && commit internal/module/auth/service/auth_service.go internal/module/auth/service/errors.go
MSG='feat(auth): add password auth HTTP handlers and routes' && commit internal/module/auth/handler/auth_handler.go internal/module/auth/routes/routes.go

MSG='feat(auth): add API key model and repository' && commit internal/module/auth/model/api_key.go internal/module/auth/repository/api_key_repository.go
MSG='feat(auth): add publish token model and repository' && commit internal/module/auth/model/publish_token.go internal/module/auth/repository/publish_token_repository.go
MSG='feat(auth): add API key service and tests' && commit internal/module/auth/service/api_key_service.go internal/module/auth/service/api_key_service_test.go
MSG='feat(auth): add publish token service' && commit internal/module/auth/service/publish_token_service.go
MSG='feat(auth): add API key and publish token handlers' && commit internal/module/auth/handler/api_key_handler.go internal/module/auth/handler/publish_token_handler.go
MSG='feat(auth): register API key and publish token routes' && commit internal/module/auth/routes/api_key_routes.go internal/module/auth/routes/token_routes.go

MSG='feat(decoder): add semantic decode engine library' && commit lib/decoder/
MSG='feat(decoder): add decoder module service and handler' && commit internal/module/decoder/

MSG='feat(realtime): add redis pubsub realtime library' && commit lib/realtime/
MSG='feat(realtime): publish ingest events over redis' && commit internal/module/ingest/service/realtime_publish.go
MSG='feat(ingest): update ingest worker service and tests' && commit internal/module/ingest/service/ingest_worker_service.go internal/module/ingest/service/ingest_worker_service_test.go internal/module/ingest/service/reorg_test.go

MSG='feat(explore): add websocket home stats handler' && commit internal/module/explore/handler/ws_handler.go
MSG='feat(explore): add event mapper for explore reads' && commit internal/module/explore/repository/event_mapper.go internal/module/explore/repository/event_mapper_test.go
MSG='feat(explore): extend explore handler DTOs and routes' && commit internal/module/explore/handler/dto.go internal/module/explore/handler/explore_handler.go internal/module/explore/handler/explore_handler_test.go internal/module/explore/routes/routes.go

MSG='feat(registry): extend schema models for bundles and projects' && commit internal/module/registry/model/
MSG='feat(registry): add project and bundle repositories' && commit internal/module/registry/repository/project_repository.go internal/module/registry/repository/contract_authority_repository.go internal/module/registry/repository/verify_challenge_repository.go
MSG='feat(registry): extend schema repository and validation' && commit internal/module/registry/repository/schema_repository.go internal/module/registry/service/validate.go internal/module/registry/service/errors.go
MSG='feat(registry): add project bundle and verify services' && commit internal/module/registry/service/project_service.go internal/module/registry/service/schema_bundle_service.go internal/module/registry/service/schema_bundle_service_test.go internal/module/registry/service/bundle_group.go internal/module/registry/service/bundle_group_test.go internal/module/registry/service/contract_authority_service.go internal/module/registry/service/verify_service.go internal/module/registry/service/verify_service_test.go
MSG='feat(registry): update core schema service and tests' && commit internal/module/registry/service/schema_service.go internal/module/registry/service/schema_service_test.go
MSG='feat(registry): add bundle and project HTTP handlers' && commit internal/module/registry/handler/bundle_handler.go internal/module/registry/handler/project_handler.go internal/module/registry/handler/schema_handler.go
MSG='feat(registry): register expanded schema registry routes' && commit internal/module/registry/routes/routes.go

MSG='feat(server): wire decoder auth registry and realtime modules' && commit cmd/server/register.go cmd/server/server.go cmd/worker/worker.go
MSG='docs(openapi): document new auth registry and decode endpoints' && commit cmd/openapi/openapi.go

echo "Atlas commits in session done"
