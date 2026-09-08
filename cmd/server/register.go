package server

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/naralabs/naralabs-atlas/config"
	authhandler "github.com/naralabs/naralabs-atlas/internal/module/auth/handler"
	authroutes "github.com/naralabs/naralabs-atlas/internal/module/auth/routes"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
	registryhandler "github.com/naralabs/naralabs-atlas/internal/module/registry/handler"
	registryroutes "github.com/naralabs/naralabs-atlas/internal/module/registry/routes"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/routes"
)

// RegisterRoutes wires the Huma API and feature routes. Returns the API for OpenAPI export.
func RegisterRoutes(
	cfg *config.Config,
	exploreHandler *handler.ExploreHandler,
	authHandler *authhandler.AuthHandler,
	registryHandler *registryhandler.SchemaHandler,
) huma.API {
	api, _ := routes.NewAPI(cfg)
	routes.RegisterExploreRoutes(api, exploreHandler)
	if authHandler != nil {
		authroutes.RegisterAuthRoutes(api, authHandler)
	}
	if registryHandler != nil {
		registryroutes.RegisterRegistryRoutes(api, registryHandler)
	}
	return api
}
