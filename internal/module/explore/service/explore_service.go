package service

import (
	"context"
	"fmt"

	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
)

type ExploreService struct {
	repo    repository.ExploreRepository
	stellar stellar.Client
}

func NewExploreService(repo repository.ExploreRepository, stellarClient stellar.Client) *ExploreService {
	return &ExploreService{repo: repo, stellar: stellarClient}
}

func (s *ExploreService) GetStats(ctx context.Context, network string) (model.NetworkStats, error) {
	stats, err := s.repo.GetStats(ctx, network)
	if err != nil {
		return model.NetworkStats{}, err
	}
	if head, err := s.stellar.LatestLedger(ctx); err == nil && head > 0 {
		h := uint64(head)
		stats.ChainHeadLedger = &h
		if stats.LastIngestedLedger <= h {
			lag := h - stats.LastIngestedLedger
			stats.IngestLagLedgers = &lag
		}
	}
	return stats, nil
}

func (s *ExploreService) ListRecentEvents(ctx context.Context, network string, limit int) ([]model.EventItem, error) {
	return s.repo.ListRecentEvents(ctx, network, limit)
}

func (s *ExploreService) ListActiveContracts(ctx context.Context, network string, limit int) ([]model.ContractItem, error) {
	return s.repo.ListActiveContracts(ctx, network, limit)
}

func (s *ExploreService) GetHome(ctx context.Context, network string, recentLimit, contractLimit int) (model.HomePayload, error) {
	stats, err := s.GetStats(ctx, network)
	if err != nil {
		return model.HomePayload{}, fmt.Errorf("home stats: %w", err)
	}
	events, err := s.repo.ListRecentEvents(ctx, network, recentLimit)
	if err != nil {
		return model.HomePayload{}, fmt.Errorf("home events: %w", err)
	}
	contracts, err := s.repo.ListActiveContracts(ctx, network, contractLimit)
	if err != nil {
		return model.HomePayload{}, fmt.Errorf("home contracts: %w", err)
	}
	return model.HomePayload{
		Stats:           stats,
		RecentEvents:    events,
		ActiveContracts: contracts,
	}, nil
}

func (s *ExploreService) GetEvent(ctx context.Context, network, id string) (model.EventDetail, error) {
	return s.repo.GetEventByID(ctx, network, id)
}

func (s *ExploreService) GetContract(
	ctx context.Context,
	network, contractID string,
) (model.ContractDetail, error) {
	return s.repo.GetContractByID(ctx, network, contractID)
}

func (s *ExploreService) ListContractEvents(
	ctx context.Context,
	network, contractID string,
	page, pageSize int,
	search, eventType, decodeStatus string,
) (model.PaginatedListResponse[model.EventItem], error) {
	if _, err := s.repo.GetContractByID(ctx, network, contractID); err != nil {
		return model.PaginatedListResponse[model.EventItem]{}, err
	}
	return s.repo.ListContractEvents(ctx, network, contractID, page, pageSize, search, eventType, decodeStatus)
}
