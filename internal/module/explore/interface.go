package explore

import (
	"context"

	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
)

type Service interface {
	GetStats(ctx context.Context, network string) (model.NetworkStats, error)
	ListRecentEvents(ctx context.Context, network string, limit int) ([]model.EventItem, error)
	ListActiveContracts(ctx context.Context, network string, limit int) ([]model.ContractItem, error)
	GetHome(ctx context.Context, network string, recentLimit, contractLimit int) (model.HomePayload, error)
	GetEvent(ctx context.Context, network, id string) (model.EventDetail, error)
	GetContract(ctx context.Context, network, contractID string) (model.ContractDetail, error)
	ListContractEvents(
		ctx context.Context,
		network, contractID string,
		page, pageSize int,
		search, eventType, decodeStatus string,
	) (model.PaginatedListResponse[model.EventItem], error)
}
