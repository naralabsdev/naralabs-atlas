package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/service"
)

type SchemaBundleService interface {
	ListContracts(ctx context.Context, network, search string, limit, offset int) (model.SchemaContractListResponse, error)
	GetRegistrySummary(ctx context.Context, network string) (model.SchemaRegistrySummary, error)
	GetContractProfile(ctx context.Context, network, contractID string) (model.SchemaContractProfile, error)
	GetBundleDetail(ctx context.Context, network, contractID string, version int) (model.SchemaBundleDetail, error)
}

type BundleHandler struct {
	svc SchemaBundleService
}

func NewBundleHandler(svc SchemaBundleService) *BundleHandler {
	return &BundleHandler{svc: svc}
}

func (h *BundleHandler) HandleListContracts(ctx context.Context, input *ListSchemaContractsInput) (*ListSchemaContractsOutput, error) {
	result, err := h.svc.ListContracts(ctx, input.Network, input.Search, input.Limit, input.Offset)
	if err != nil {
		return nil, mapBundleError(err)
	}
	return &ListSchemaContractsOutput{Body: result}, nil
}

func (h *BundleHandler) HandleGetRegistrySummary(ctx context.Context, input *SchemaRegistrySummaryInput) (*SchemaRegistrySummaryOutput, error) {
	result, err := h.svc.GetRegistrySummary(ctx, input.Network)
	if err != nil {
		return nil, mapBundleError(err)
	}
	return &SchemaRegistrySummaryOutput{Body: result}, nil
}

func (h *BundleHandler) HandleGetContractProfile(ctx context.Context, input *SchemaContractProfileInput) (*SchemaContractProfileOutput, error) {
	result, err := h.svc.GetContractProfile(ctx, input.Network, input.ContractID)
	if err != nil {
		return nil, mapBundleError(err)
	}
	return &SchemaContractProfileOutput{Body: result}, nil
}

func (h *BundleHandler) HandleGetBundleDetail(ctx context.Context, input *SchemaBundleDetailInput) (*SchemaBundleDetailOutput, error) {
	result, err := h.svc.GetBundleDetail(ctx, input.Network, input.ContractID, input.Version)
	if err != nil {
		return nil, mapBundleError(err)
	}
	return &SchemaBundleDetailOutput{Body: result}, nil
}

func mapBundleError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidContractID):
		return schemaError(http.StatusBadRequest, "INVALID_CONTRACT_ID", "contractId must be a valid Soroban contract address")
	case errors.Is(err, service.ErrInvalidNetwork):
		return schemaError(http.StatusBadRequest, "INVALID_NETWORK", "network must be testnet, mainnet, or futurenet")
	case errors.Is(err, service.ErrSchemaNotFound):
		return huma.Error404NotFound("schema not found")
	default:
		return huma.Error500InternalServerError("schema bundle request failed", err)
	}
}

type ListSchemaContractsInput struct {
	Network string `query:"network" doc:"Filter by network" enum:"testnet,mainnet,futurenet" example:"testnet"`
	Search  string `query:"search" doc:"Filter by contract ID substring"`
	Limit   int    `query:"limit" minimum:"1" maximum:"100" default:"20"`
	Offset  int    `query:"offset" minimum:"0" default:"0"`
}

type ListSchemaContractsOutput struct {
	Body model.SchemaContractListResponse
}

type SchemaRegistrySummaryInput struct {
	Network string `query:"network" doc:"Filter by network" enum:"testnet,mainnet,futurenet" example:"testnet"`
}

type SchemaRegistrySummaryOutput struct {
	Body model.SchemaRegistrySummary
}

type SchemaContractProfileInput struct {
	ContractID string `path:"contract_id" doc:"Soroban contract address"`
	Network    string `query:"network" doc:"Stellar network" enum:"testnet,mainnet,futurenet" example:"testnet"`
}

type SchemaContractProfileOutput struct {
	Body model.SchemaContractProfile
}

type SchemaBundleDetailInput struct {
	ContractID string `path:"contract_id" doc:"Soroban contract address"`
	Version    int    `path:"version" doc:"Schema bundle version" minimum:"1"`
	Network    string `query:"network" doc:"Stellar network" enum:"testnet,mainnet,futurenet" example:"testnet"`
}

type SchemaBundleDetailOutput struct {
	Body model.SchemaBundleDetail
}
