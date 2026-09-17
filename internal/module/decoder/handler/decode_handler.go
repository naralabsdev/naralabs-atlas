package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	decoderservice "github.com/naralabs/naralabs-atlas/internal/module/decoder/service"
)

type APIKeyAuthenticator interface {
	RequireAPIKey(ctx context.Context, authorization string) error
}

type DecodeHandler struct {
	svc     *decoderservice.DecodeService
	apiKeys APIKeyAuthenticator
}

func NewDecodeHandler(svc *decoderservice.DecodeService, apiKeys APIKeyAuthenticator) *DecodeHandler {
	return &DecodeHandler{svc: svc, apiKeys: apiKeys}
}

func (h *DecodeHandler) HandleDecode(ctx context.Context, input *DecodeInput) (*DecodeOutput, error) {
	if err := h.apiKeys.RequireAPIKey(ctx, input.Authorization); err != nil {
		return nil, decodeError(http.StatusUnauthorized, "UNAUTHORIZED", "Valid API key required")
	}

	result, err := h.svc.Decode(ctx, toDecodeInput(input.Body))
	if err != nil {
		return nil, mapDecodeError(err)
	}

	out := &DecodeOutput{}
	out.Body = result
	return out, nil
}

func (h *DecodeHandler) HandleDecodeBatch(ctx context.Context, input *BatchDecodeInput) (*BatchDecodeOutput, error) {
	if err := h.apiKeys.RequireAPIKey(ctx, input.Authorization); err != nil {
		return nil, decodeError(http.StatusUnauthorized, "UNAUTHORIZED", "Valid API key required")
	}

	if len(input.Body.Events) == 0 {
		return nil, huma.Error400BadRequest("events array is required")
	}

	items := make([]decoderservice.DecodeEventInput, 0, len(input.Body.Events))
	for _, event := range input.Body.Events {
		items = append(items, toDecodeInput(event))
	}

	result, err := h.svc.DecodeBatch(ctx, items)
	if err != nil {
		return nil, mapDecodeError(err)
	}

	out := &BatchDecodeOutput{}
	out.Body = result
	return out, nil
}

func toDecodeInput(body DecodeEventBody) decoderservice.DecodeEventInput {
	return decoderservice.DecodeEventInput{
		Network:       body.Network,
		ContractID:    body.ContractID,
		EventName:     body.EventName,
		SchemaVersion: body.SchemaVersion,
		TopicsXDR:     body.TopicsXDR,
		TopicsJSON:    body.TopicsJSON,
		ValueXDR:      body.ValueXDR,
		ValueJSON:     body.ValueJSON,
	}
}

func mapDecodeError(err error) error {
	switch {
	case errors.Is(err, decoderservice.ErrInvalidDecodeInput):
		return huma.Error400BadRequest("invalid decode request")
	case errors.Is(err, decoderservice.ErrBatchTooLarge):
		return decodeError(http.StatusBadRequest, "BATCH_TOO_LARGE", "Batch exceeds maximum size")
	default:
		return huma.Error500InternalServerError("decode request failed", err)
	}
}

func decodeError(status int, code, message string) error {
	return huma.NewError(status, message, errors.New(code))
}

type DecodeEventBody struct {
	Network       string            `json:"network" doc:"Stellar network" example:"testnet"`
	ContractID    string            `json:"contractId" doc:"Emitting contract ID" example:"CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4"`
	EventName     string            `json:"eventName,omitempty" doc:"Optional event name hint"`
	SchemaVersion int               `json:"schemaVersion,omitempty" doc:"Optional schema version"`
	TopicsXDR     []string          `json:"topicsXdr,omitempty" doc:"Base64 XDR topic array"`
	TopicsJSON    []json.RawMessage `json:"topicsJson,omitempty" doc:"RPC-style topic JSON array"`
	ValueXDR      string            `json:"valueXdr,omitempty" doc:"Base64 XDR event value"`
	ValueJSON     json.RawMessage   `json:"valueJson,omitempty" doc:"RPC-style value JSON"`
}

type DecodeInput struct {
	Authorization string `header:"Authorization" doc:"Bearer nl_api_ API key"`
	Body          DecodeEventBody
}

type DecodeOutput struct {
	Body decoderservice.DecodeEventResult
}

type BatchDecodeInput struct {
	Authorization string `header:"Authorization" doc:"Bearer nl_api_ API key"`
	Body          struct {
		Events []DecodeEventBody `json:"events" doc:"Events to decode (max 50)"`
	}
}

type BatchDecodeOutput struct {
	Body decoderservice.BatchDecodeResult
}
