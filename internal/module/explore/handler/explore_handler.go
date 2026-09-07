package handler

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	explore "github.com/naralabs/naralabs-atlas/internal/module/explore"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
)

type ExploreHandler struct {
	svc     explore.Service
	network string
}

func NewExploreHandler(svc explore.Service, defaultNetwork string) *ExploreHandler {
	return &ExploreHandler{svc: svc, network: defaultNetwork}
}

func (h *ExploreHandler) HandleHealth(_ context.Context, _ *struct{}) (*HealthOutput, error) {
	out := &HealthOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func (h *ExploreHandler) HandleStats(ctx context.Context, input *StatsInput) (*StatsOutput, error) {
	stats, err := h.svc.GetStats(ctx, h.resolveNetwork(input.Network))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load stats", err)
	}
	return &StatsOutput{Body: stats}, nil
}

func (h *ExploreHandler) HandleRecentEvents(ctx context.Context, input *EventsInput) (*EventsOutput, error) {
	items, err := h.svc.ListRecentEvents(ctx, h.resolveNetwork(input.Network), clampLimit(input.Limit))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load events", err)
	}
	return &EventsOutput{Body: model.ListResponse[model.EventItem]{Items: items}}, nil
}

func (h *ExploreHandler) HandleActiveContracts(ctx context.Context, input *ContractsInput) (*ContractsOutput, error) {
	items, err := h.svc.ListActiveContracts(ctx, h.resolveNetwork(input.Network), clampLimit(input.Limit))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load contracts", err)
	}
	return &ContractsOutput{Body: model.ListResponse[model.ContractItem]{Items: items}}, nil
}

func (h *ExploreHandler) HandleHome(ctx context.Context, input *HomeInput) (*HomeOutput, error) {
	network := h.resolveNetwork(input.Network)
	payload, err := h.svc.GetHome(
		ctx,
		network,
		clampLimit(input.RecentLimit),
		clampLimit(input.ContractLimit),
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load home payload", err)
	}
	return &HomeOutput{Body: payload}, nil
}

func (h *ExploreHandler) HandleGetEvent(ctx context.Context, input *EventDetailInput) (*EventDetailOutput, error) {
	detail, err := h.svc.GetEvent(ctx, h.resolveNetwork(input.Network), input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, huma.Error404NotFound("event not found")
		}
		return nil, huma.Error500InternalServerError("failed to load event", err)
	}
	return &EventDetailOutput{Body: detail}, nil
}

func (h *ExploreHandler) HandleGetContract(ctx context.Context, input *ContractDetailInput) (*ContractDetailOutput, error) {
	detail, err := h.svc.GetContract(ctx, h.resolveNetwork(input.Network), input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrContractNotFound) {
			return nil, huma.Error404NotFound("contract not found")
		}
		return nil, huma.Error500InternalServerError("failed to load contract", err)
	}
	return &ContractDetailOutput{Body: detail}, nil
}

func (h *ExploreHandler) HandleListContractEvents(ctx context.Context, input *ContractEventsInput) (*ContractEventsOutput, error) {
	payload, err := h.svc.ListContractEvents(
		ctx,
		h.resolveNetwork(input.Network),
		input.ID,
		clampPage(input.Page),
		clampPageSize(input.PageSize),
		input.Search,
		input.EventType,
		input.DecodeStatus,
	)
	if err != nil {
		if errors.Is(err, repository.ErrContractNotFound) {
			return nil, huma.Error404NotFound("contract not found")
		}
		return nil, huma.Error500InternalServerError("failed to load contract events", err)
	}
	return &ContractEventsOutput{Body: payload}, nil
}

func clampPage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func clampPageSize(size int) int {
	const min, max, fallback = 1, 100, 20
	if size == 0 {
		return fallback
	}
	if size < min {
		return min
	}
	if size > max {
		return max
	}
	return size
}

func (h *ExploreHandler) resolveNetwork(network string) string {
	if network != "" {
		return network
	}
	return h.network
}

func clampLimit(limit int) int {
	const min, max, fallback = 1, 100, 8
	if limit == 0 {
		return fallback
	}
	if limit < min {
		return min
	}
	if limit > max {
		return max
	}
	return limit
}
