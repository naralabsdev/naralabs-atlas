package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/auth/handler"
)

const apiKeyBase = "/v1/api-keys"

// RegisterAPIKeyRoutes registers developer API key management endpoints.
func RegisterAPIKeyRoutes(api huma.API, h *handler.APIKeyHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "create-api-key",
		Method:      http.MethodPost,
		Path:        apiKeyBase,
		Summary:     "Create API key",
		Description: "Create a new nl_api key for programmatic access to decode and explore endpoints. The raw key is returned once.",
		Tags:        []string{"api-keys"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleCreate)

	huma.Register(api, huma.Operation{
		OperationID: "list-api-keys",
		Method:      http.MethodGet,
		Path:        apiKeyBase,
		Summary:     "List API keys",
		Description: "List active API keys for the authenticated user.",
		Tags:        []string{"api-keys"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleList)

	huma.Register(api, huma.Operation{
		OperationID: "revoke-api-key",
		Method:      http.MethodDelete,
		Path:        apiKeyBase + "/{id}",
		Summary:     "Revoke API key",
		Description: "Revoke an API key so it can no longer access protected endpoints.",
		Tags:        []string{"api-keys"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleRevoke)
}
