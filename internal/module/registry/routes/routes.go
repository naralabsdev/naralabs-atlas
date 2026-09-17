package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/handler"
)

const schemaBase = "/v1/schemas"
const projectBase = "/v1/schema-projects"

// RegisterRegistryRoutes registers SEP-0048 schema registry endpoints.
func RegisterRegistryRoutes(api huma.API, h *handler.SchemaHandler, projects *handler.ProjectHandler, bundles *handler.BundleHandler) {
	if projects != nil {
		huma.Register(api, huma.Operation{
			OperationID: "list-schema-projects",
			Method:      http.MethodGet,
			Path:        projectBase,
			Summary:     "List schema projects",
			Description: "List schema registry projects for the authenticated user.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleList)

		huma.Register(api, huma.Operation{
			OperationID: "create-schema-project",
			Method:      http.MethodPost,
			Path:        projectBase,
			Summary:     "Create schema project",
			Description: "Create a draft schema project with an embedded publish token.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleCreate)

		huma.Register(api, huma.Operation{
			OperationID: "get-schema-project",
			Method:      http.MethodGet,
			Path:        projectBase + "/{id}",
			Summary:     "Get schema project",
			Description: "Return schema project details including the publish token for CLI auth.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleGet)

		huma.Register(api, huma.Operation{
			OperationID: "update-schema-project",
			Method:      http.MethodPatch,
			Path:        projectBase + "/{id}",
			Summary:     "Update schema project",
			Description: "Update schema name or description.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleUpdate)

		huma.Register(api, huma.Operation{
			OperationID: "list-schema-project-publications",
			Method:      http.MethodGet,
			Path:        projectBase + "/{id}/publications",
			Summary:     "List schema project publications",
			Description: "Return published event schemas grouped by contract with tier and version metadata.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleListPublications)

		huma.Register(api, huma.Operation{
			OperationID: "create-schema-verify-challenge",
			Method:      http.MethodPost,
			Path:        projectBase + "/{id}/verify/challenge",
			Summary:     "Create schema verify challenge",
			Description: "Create a wallet signing challenge for contract-level schema verification.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleVerifyChallenge)

		huma.Register(api, huma.Operation{
			OperationID: "verify-schema-contract",
			Method:      http.MethodPost,
			Path:        projectBase + "/{id}/verify/contract",
			Summary:     "Verify schema contract",
			Description: "Verify wallet signature and promote latest community schemas to verified for a contract.",
			Tags:        []string{"registry"},
			Security:    []map[string][]string{{"bearerAuth": {}}},
		}, projects.HandleVerifyContract)
	}
	if h == nil {
		return
	}

	if bundles != nil {
		huma.Register(api, huma.Operation{
			OperationID: "get-schema-registry-summary",
			Method:      http.MethodGet,
			Path:        schemaBase + "/summary",
			Summary:     "Get schema registry summary",
			Description: "Return aggregate counts for published schema contracts and trust tiers.",
			Tags:        []string{"registry"},
		}, bundles.HandleGetRegistrySummary)

		huma.Register(api, huma.Operation{
			OperationID: "list-schema-contracts",
			Method:      http.MethodGet,
			Path:        schemaBase + "/contracts",
			Summary:     "List schema contracts",
			Description: "List distinct Soroban contracts with published schema bundles and trust metadata.",
			Tags:        []string{"registry"},
		}, bundles.HandleListContracts)

		huma.Register(api, huma.Operation{
			OperationID: "get-schema-contract-profile",
			Method:      http.MethodGet,
			Path:        schemaBase + "/contracts/{contract_id}",
			Summary:     "Get schema contract profile",
			Description: "Return contract identity, schema bundles, trust status, and optional indexed activity.",
			Tags:        []string{"registry"},
		}, bundles.HandleGetContractProfile)

		huma.Register(api, huma.Operation{
			OperationID: "get-schema-bundle-detail",
			Method:      http.MethodGet,
			Path:        schemaBase + "/contracts/{contract_id}/bundles/{version}",
			Summary:     "Get schema bundle detail",
			Description: "Return a publish-batch schema bundle with all event definitions for the version.",
			Tags:        []string{"registry"},
		}, bundles.HandleGetBundleDetail)
	}

	huma.Register(api, huma.Operation{
		OperationID: "list-my-schemas",
		Method:      http.MethodGet,
		Path:        schemaBase + "/mine",
		Summary:     "List my schemas",
		Description: "List event schemas published by the authenticated user.",
		Tags:        []string{"registry"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleListMine)

	huma.Register(api, huma.Operation{
		OperationID: "publish-schema",
		Method:      http.MethodPost,
		Path:        schemaBase,
		Summary:     "Publish event schema",
		Description: "Publish a new event schema version using a publish token (nl_live_…).",
		Tags:        []string{"registry"},
		Security:    []map[string][]string{{"publishTokenAuth": {}}},
	}, h.HandlePublish)

	huma.Register(api, huma.Operation{
		OperationID: "verify-schema",
		Method:      http.MethodPost,
		Path:        schemaBase + "/{id}/verify",
		Summary:     "Verify schema",
		Description: "Mark a published schema as verified for the contract deployer wallet.",
		Tags:        []string{"registry"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleVerify)

	huma.Register(api, huma.Operation{
		OperationID: "get-schema-by-id",
		Method:      http.MethodGet,
		Path:        schemaBase + "/events/{id}",
		Summary:     "Get schema by ID",
		Description: "Retrieve a published event schema by its registry ID.",
		Tags:        []string{"registry"},
	}, h.HandleGetByID)

	huma.Register(api, huma.Operation{
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

	huma.Register(api, huma.Operation{
		OperationID: "list-event-schema-versions",
		Method:      http.MethodGet,
		Path:        schemaBase + "/{contract_id}/{event_name}/versions",
		Summary:     "List event schema versions",
		Description: "Return all published schema versions for a contract event, newest first.",
		Tags:        []string{"registry"},
	}, h.HandleListEventVersions)
}
