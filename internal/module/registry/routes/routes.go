package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/handler"
)

const schemaBase = "/v1/schemas"

// RegisterRegistryRoutes registers SEP-0048 schema registry endpoints.
func RegisterRegistryRoutes(api huma.API, h *handler.SchemaHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "publish-schema",
		Method:      http.MethodPost,
		Path:        schemaBase,
		Summary:     "Publish event schema",
		Description: "Publish a new SEP-0048-aligned event schema version for a contract and network.",
		Tags:        []string{"registry"},
	}, h.HandlePublish)

	huma.Register(api, huma.Operation{
		OperationID: "list-schemas",
		Method:      http.MethodGet,
		Path:        schemaBase,
		Summary:     "List event schemas",
		Description: "Search and list published event schemas, optionally filtered by network and contract ID substring.",
		Tags:        []string{"registry"},
	}, h.HandleList)

	huma.Register(api, huma.Operation{
		OperationID: "list-contract-schemas",
		Method:      http.MethodGet,
		Path:        schemaBase + "/{contract_id}",
		Summary:     "List schemas for contract",
		Description: "Return the latest published schema for each event on a contract.",
		Tags:        []string{"registry"},
	}, h.HandleListByContract)

	huma.Register(api, huma.Operation{
		OperationID: "get-event-schema",
		Method:      http.MethodGet,
		Path:        schemaBase + "/{contract_id}/{event_name}",
		Summary:     "Get event schema",
		Description: "Retrieve a specific event schema by contract, event name, and optional version.",
		Tags:        []string{"registry"},
	}, h.HandleGetEventSchema)
}
