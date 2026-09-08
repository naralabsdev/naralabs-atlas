package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/service"
)

type SchemaHandler struct {
	svc *service.SchemaService
}

func NewSchemaHandler(svc *service.SchemaService) *SchemaHandler {
	return &SchemaHandler{svc: svc}
}

func (h *SchemaHandler) HandlePublish(ctx context.Context, input *PublishInput) (*PublishOutput, error) {
	schema, err := h.svc.Publish(ctx, model.PublishInput{
		ContractID: input.Body.ContractID,
		Network:    input.Body.Network,
		EventName:  input.Body.EventName,
		SchemaBody: json.RawMessage(input.Body.SchemaBody),
		Author:     input.Body.Author,
	})
	if err != nil {
		return nil, mapSchemaError(err)
	}

	out := &PublishOutput{}
	out.Body = schema
	return out, nil
}

func (h *SchemaHandler) HandleList(ctx context.Context, input *ListInput) (*ListOutput, error) {
	result, err := h.svc.List(ctx, input.Network, input.Search, input.Limit, input.Offset)
	if err != nil {
		return nil, mapSchemaError(err)
	}
	return &ListOutput{Body: result}, nil
}

func (h *SchemaHandler) HandleListByContract(ctx context.Context, input *ContractSchemasInput) (*ContractSchemasOutput, error) {
	items, err := h.svc.ListByContract(ctx, input.Network, input.ContractID)
	if err != nil {
		return nil, mapSchemaError(err)
	}
	return &ContractSchemasOutput{
		Body: model.ListResponse{
			Items: items,
			Total: len(items),
		},
	}, nil
}

func (h *SchemaHandler) HandleGetEventSchema(ctx context.Context, input *EventSchemaInput) (*EventSchemaOutput, error) {
	schema, err := h.svc.GetEventSchema(ctx, input.Network, input.ContractID, input.EventName, input.Version)
	if err != nil {
		return nil, mapSchemaError(err)
	}
	return &EventSchemaOutput{Body: schema}, nil
}

func mapSchemaError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidContractID):
		return schemaError(http.StatusBadRequest, "INVALID_CONTRACT_ID", "contractId must be a valid Soroban contract address")
	case errors.Is(err, service.ErrInvalidNetwork):
		return schemaError(http.StatusBadRequest, "INVALID_NETWORK", "network must be testnet, mainnet, or futurenet")
	case errors.Is(err, service.ErrInvalidEventName):
		return schemaError(http.StatusBadRequest, "INVALID_EVENT_NAME", "eventName is required")
	case errors.Is(err, service.ErrInvalidSchemaBody):
		return schemaError(http.StatusBadRequest, "INVALID_SCHEMA_BODY", "schemaBody must be a SEP-0048 event definition with matching name and non-empty args")
	case errors.Is(err, service.ErrSchemaNotFound):
		return huma.Error404NotFound("schema not found")
	default:
		if msg := err.Error(); strings.Contains(msg, "invalid") {
			return huma.Error400BadRequest(msg)
		}
		return huma.Error500InternalServerError("schema request failed", err)
	}
}

type schemaErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e schemaErrorBody) Error() string {
	return e.Message
}

func (e schemaErrorBody) ErrorDetail() *huma.ErrorDetail {
	return &huma.ErrorDetail{
		Message:  e.Message,
		Location: "code",
		Value:    e.Code,
	}
}

func schemaError(status int, code, message string) error {
	return huma.NewError(status, message, schemaErrorBody{Code: code, Message: message})
}

type PublishInput struct {
	Body struct {
		ContractID string          `json:"contractId" doc:"Soroban contract address" example:"CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"`
		Network    string          `json:"network,omitempty" doc:"Stellar network" enum:"testnet,mainnet,futurenet" example:"testnet"`
		EventName  string          `json:"eventName" doc:"Event name from the contract spec" example:"transfer"`
		SchemaBody json.RawMessage `json:"schemaBody" doc:"SEP-0048 event definition JSON"`
		Author     string          `json:"author,omitempty" doc:"Optional publisher label" example:"Blend Team"`
	}
}

type PublishOutput struct {
	Body model.EventSchemaPublic
}

type ListInput struct {
	Network string `query:"network" doc:"Filter by network" enum:"testnet,mainnet,futurenet" example:"testnet"`
	Search  string `query:"search" doc:"Filter by contract ID substring"`
	Limit   int    `query:"limit" minimum:"1" maximum:"100" default:"20"`
	Offset  int    `query:"offset" minimum:"0" default:"0"`
}

type ListOutput struct {
	Body model.ListResponse
}

type ContractSchemasInput struct {
	ContractID string `path:"contract_id" doc:"Soroban contract address"`
	Network    string `query:"network" doc:"Stellar network" enum:"testnet,mainnet,futurenet" example:"testnet"`
}

type ContractSchemasOutput struct {
	Body model.ListResponse
}

type EventSchemaInput struct {
	ContractID string `path:"contract_id" doc:"Soroban contract address"`
	EventName  string `path:"event_name" doc:"Event name"`
	Network    string `query:"network" doc:"Stellar network" enum:"testnet,mainnet,futurenet" example:"testnet"`
	Version    int    `query:"version" doc:"Specific schema version; latest when omitted" minimum:"0"`
}

type EventSchemaOutput struct {
	Body model.EventSchemaPublic
}
