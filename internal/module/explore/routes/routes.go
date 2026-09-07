package routes

import (
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/docs"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
)

const openAPIVersion = "0.3.0"

// NewAPI builds the chi router and Huma API instance (without registering feature routes).
func NewAPI(cfg *config.Config) (huma.API, chi.Router) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	if origins := cfg.CORSOriginList(); len(origins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   origins,
			AllowedMethods:   []string{http.MethodGet, http.MethodOptions},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
	}

	api := humachi.New(r, buildHumaConfig(cfg))
	return api, r
}

// RegisterExploreRoutes registers explorer read endpoints on the Huma API.
func RegisterExploreRoutes(api huma.API, h *handler.ExploreHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Description: "Returns service liveness status.",
		Tags:        []string{"health"},
	}, h.HandleHealth)

	base := "/v1"
	huma.Register(api, huma.Operation{
		OperationID: "get-home",
		Method:      http.MethodGet,
		Path:        base + "/home",
		Summary:     "Home dashboard payload",
		Description: "Aggregated stats, recent events, and active contracts for the explorer home page.",
		Tags:        []string{"explore"},
	}, h.HandleHome)

	huma.Register(api, huma.Operation{
		OperationID: "get-stats",
		Method:      http.MethodGet,
		Path:        base + "/stats",
		Summary:     "Network statistics",
		Description: "Network overview metrics and recent activity buckets.",
		Tags:        []string{"explore"},
	}, h.HandleStats)

	huma.Register(api, huma.Operation{
		OperationID: "list-events",
		Method:      http.MethodGet,
		Path:        base + "/events",
		Summary:     "Recent events",
		Description: "Paginated list of recently ingested Soroban events.",
		Tags:        []string{"explore"},
	}, h.HandleRecentEvents)

	huma.Register(api, huma.Operation{
		OperationID: "get-event",
		Method:      http.MethodGet,
		Path:        base + "/events/{id}",
		Summary:     "Event detail",
		Description: "Full Soroban event payload including decoded topics, value, and XDR.",
		Tags:        []string{"explore"},
	}, h.HandleGetEvent)

	huma.Register(api, huma.Operation{
		OperationID: "list-contracts",
		Method:      http.MethodGet,
		Path:        base + "/contracts",
		Summary:     "Active contracts",
		Description: "Contracts ranked by recent event activity.",
		Tags:        []string{"explore"},
	}, h.HandleActiveContracts)

	huma.Register(api, huma.Operation{
		OperationID: "get-contract",
		Method:      http.MethodGet,
		Path:        base + "/contracts/{id}",
		Summary:     "Contract detail",
		Description: "Contract activity summary and event type breakdown.",
		Tags:        []string{"explore"},
	}, h.HandleGetContract)

	huma.Register(api, huma.Operation{
		OperationID: "list-contract-events",
		Method:      http.MethodGet,
		Path:        base + "/contracts/{id}/events",
		Summary:     "Contract events",
		Description: "Paginated, filterable list of events for a Soroban contract.",
		Tags:        []string{"explore"},
	}, h.HandleListContractEvents)
}

// NewRouter wires chi middleware, Huma docs, and explore routes into an http.Handler.
func NewRouter(cfg *config.Config, exploreHandler *handler.ExploreHandler) http.Handler {
	api, router := NewAPI(cfg)
	RegisterExploreRoutes(api, exploreHandler)
	return router
}

func buildHumaConfig(cfg *config.Config) huma.Config {
	humaCfg := huma.DefaultConfig(cfg.ServiceName, openAPIVersion)
	humaCfg.Info.Description = docs.APIDescription
	humaCfg.Info.Contact = &huma.Contact{
		Name:  "NaraLabs",
		URL:   "https://github.com/naralabsdev/naralabs-atlas",
		Email: "dev@naralabs.dev",
	}
	humaCfg.Servers = []*huma.Server{{URL: publishURL(cfg)}}
	humaCfg.Extensions = map[string]any{
		"tags": []map[string]any{
			{"name": "health", "description": "Service health endpoints"},
			{"name": "explore", "description": "Soroban explorer read API"},
		},
	}
	humaCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Optional Bearer token for authenticated endpoints (future M2 routes). Stored in browser localStorage by Scalar docs.",
		},
	}

	if cfg.IsProduction() {
		humaCfg.DocsPath = ""
	} else {
		humaCfg.DocsRenderer = huma.DocsRendererScalar
		humaCfg.DocsRendererConfig = map[string]any{
			"theme": "purple",
		}
	}

	return humaCfg
}

func publishURL(cfg *config.Config) string {
	if v := strings.TrimSpace(cfg.PublishURL); v != "" {
		return v
	}
	addr := strings.TrimSpace(cfg.HTTP.Addr)
	if addr == "" {
		return "http://localhost:8080"
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "http://" + addr
}
