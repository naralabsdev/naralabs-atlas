package server

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/routes"
)

// RegisterRoutes wires the Huma API and explore routes. Returns the API for OpenAPI export.
func RegisterRoutes(cfg *config.Config, exploreHandler *handler.ExploreHandler) huma.API {
	api, _ := routes.NewAPI(cfg)
	routes.RegisterExploreRoutes(api, exploreHandler)
	return api
}
