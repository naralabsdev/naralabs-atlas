package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/decoder/handler"
)

const decodeBase = "/v1/decode"

// RegisterDecoderRoutes registers event decoder endpoints.
func RegisterDecoderRoutes(api huma.API, h *handler.DecodeHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "decode-event",
		Method:      http.MethodPost,
		Path:        decodeBase,
		Summary:     "Decode Soroban event",
		Description: "Decode one raw Soroban event into semantic JSON using registered SEP-0048 schemas.",
		Tags:        []string{"decoder"},
		Security:    []map[string][]string{{"apiKeyAuth": {}}},
	}, h.HandleDecode)

	huma.Register(api, huma.Operation{
		OperationID: "decode-events-batch",
		Method:      http.MethodPost,
		Path:        decodeBase + "/batch",
		Summary:     "Batch decode Soroban events",
		Description: "Decode up to 50 raw Soroban events in one request.",
		Tags:        []string{"decoder"},
		Security:    []map[string][]string{{"apiKeyAuth": {}}},
	}, h.HandleDecodeBatch)

	if h.PlaygroundEnabled() {
		huma.Register(api, huma.Operation{
			OperationID: "playground-decode-event",
			Method:      http.MethodPost,
			Path:        "/v1/playground/decode",
			Summary:     "Decode event (Naralabs playground BFF)",
			Description: "Server-to-server decode for the Naralabs web Decode Playground. Requires PLAYGROUND_BFF_TOKEN.",
			Tags:        []string{"decoder", "playground"},
		}, h.HandlePlaygroundDecode)
	}
}
