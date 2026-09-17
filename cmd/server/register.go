package server

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/naralabs/naralabs-atlas/config"
	authhandler "github.com/naralabs/naralabs-atlas/internal/module/auth/handler"
	authroutes "github.com/naralabs/naralabs-atlas/internal/module/auth/routes"
	decoderhandler "github.com/naralabs/naralabs-atlas/internal/module/decoder/handler"
	decoderroutes "github.com/naralabs/naralabs-atlas/internal/module/decoder/routes"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/routes"
	registryhandler "github.com/naralabs/naralabs-atlas/internal/module/registry/handler"
	registryroutes "github.com/naralabs/naralabs-atlas/internal/module/registry/routes"
)

// RegisterRoutes wires the Huma API and feature routes. Returns the API for OpenAPI export.
func RegisterRoutes(
	cfg *config.Config,
	exploreHandler *handler.ExploreHandler,
	authHandler *authhandler.AuthHandler,
	tokenHandler *authhandler.PublishTokenHandler,
	apiKeyHandler *authhandler.APIKeyHandler,
	decodeHandler *decoderhandler.DecodeHandler,
	registryHandler *registryhandler.SchemaHandler,
	projectHandler *registryhandler.ProjectHandler,
	bundleHandler *registryhandler.BundleHandler,
) huma.API {
	api, _ := routes.NewAPI(cfg)
	routes.RegisterExploreRoutes(api, exploreHandler)
	if authHandler != nil {
		authroutes.RegisterAuthRoutes(api, authHandler)
	}
	if tokenHandler != nil {
		authroutes.RegisterPublishTokenRoutes(api, tokenHandler)
	}
	if apiKeyHandler != nil {
		authroutes.RegisterAPIKeyRoutes(api, apiKeyHandler)
	}
	if decodeHandler != nil {
		decoderroutes.RegisterDecoderRoutes(api, decodeHandler)
	}
	if registryHandler != nil || projectHandler != nil || bundleHandler != nil {
		registryroutes.RegisterRegistryRoutes(api, registryHandler, projectHandler, bundleHandler)
	}
	return api
}
