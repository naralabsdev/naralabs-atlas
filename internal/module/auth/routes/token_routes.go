package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/auth/handler"
)

const tokenBase = "/v1/tokens"

// RegisterPublishTokenRoutes registers publish token management endpoints.
func RegisterPublishTokenRoutes(api huma.API, h *handler.PublishTokenHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "create-publish-token",
		Method:      http.MethodPost,
		Path:        tokenBase,
		Summary:     "Create publish token",
		Description: "Create a new nl_live publish token for CLI schema publishing. The raw token is returned once.",
		Tags:        []string{"tokens"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleCreate)

	huma.Register(api, huma.Operation{
		OperationID: "list-publish-tokens",
		Method:      http.MethodGet,
		Path:        tokenBase,
		Summary:     "List publish tokens",
		Description: "List active publish tokens for the authenticated user.",
		Tags:        []string{"tokens"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleList)

	huma.Register(api, huma.Operation{
		OperationID: "revoke-publish-token",
		Method:      http.MethodDelete,
		Path:        tokenBase + "/{id}",
		Summary:     "Revoke publish token",
		Description: "Revoke a publish token so it can no longer publish schemas.",
		Tags:        []string{"tokens"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleRevoke)
}
