package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	exploremodel "github.com/naralabs/naralabs-atlas/internal/module/explore/model"
	explorerepo "github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

type BundleSchemaRepository interface {
	ListPublishedByNetwork(ctx context.Context, network, search string) ([]model.EventSchema, error)
	ListPublishedForContract(ctx context.Context, network, contractID string) ([]model.EventSchema, error)
}

type ContractActivityReader interface {
	GetContractByID(ctx context.Context, network, contractID string) (exploremodel.ContractDetail, error)
}

type SchemaBundleService struct {
	schemas        BundleSchemaRepository
	activity       ContractActivityReader
	defaultNetwork string
}

func NewSchemaBundleService(
	schemas BundleSchemaRepository,
	activity ContractActivityReader,
	defaultNetwork string,
) *SchemaBundleService {
	return &SchemaBundleService{
		schemas:        schemas,
		activity:       activity,
		defaultNetwork: strings.ToLower(strings.TrimSpace(defaultNetwork)),
	}
}

func (s *SchemaBundleService) ListContracts(
	ctx context.Context,
	network, search string,
	limit, offset int,
) (model.SchemaContractListResponse, error) {
	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return model.SchemaContractListResponse{}, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	schemas, err := s.schemas.ListPublishedByNetwork(ctx, resolved, search)
	if err != nil {
		return model.SchemaContractListResponse{}, err
	}

	summaries := buildContractSummaries(schemas)
	total := len(summaries)
	if offset >= total {
		return model.SchemaContractListResponse{Items: []model.SchemaContractListItem{}, Total: total}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return model.SchemaContractListResponse{
		Items: summaries[offset:end],
		Total: total,
	}, nil
}

func (s *SchemaBundleService) GetRegistrySummary(
	ctx context.Context,
	network string,
) (model.SchemaRegistrySummary, error) {
	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return model.SchemaRegistrySummary{}, err
	}

	schemas, err := s.schemas.ListPublishedByNetwork(ctx, resolved, "")
	if err != nil {
		return model.SchemaRegistrySummary{}, err
	}

	summaries := buildContractSummaries(schemas)

	var eventSchemas int
	var verifiedContracts int
	for _, item := range summaries {
		eventSchemas += item.EventCount
		if item.IsVerified {
			verifiedContracts++
		}
	}

	return model.SchemaRegistrySummary{
		Network:            resolved,
		PublishedContracts: len(summaries),
		EventSchemas:       eventSchemas,
		VerifiedContracts:  verifiedContracts,
		CommunityContracts: len(summaries) - verifiedContracts,
	}, nil
}

func (s *SchemaBundleService) GetContractProfile(
	ctx context.Context,
	network, contractID string,
) (model.SchemaContractProfile, error) {
	id, err := normalizeContractID(contractID)
	if err != nil {
		return model.SchemaContractProfile{}, err
	}

	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return model.SchemaContractProfile{}, err
	}

	schemas, err := s.schemas.ListPublishedForContract(ctx, resolved, id)
	if err != nil {
		return model.SchemaContractProfile{}, err
	}

	bundles := groupIntoBundles(schemas)
	isVerified, verifiedAt := contractVerification(schemas)

	profile := model.SchemaContractProfile{
		ContractID: id,
		Network:    resolved,
		Indexed:    false,
		IsVerified: isVerified,
		VerifiedAt: verifiedAt,
		Bundles:    toBundleSummaries(bundles),
		Activity:   nil,
	}

	if s.activity != nil {
		activity, indexed, activityErr := s.loadActivity(ctx, resolved, id)
		if activityErr != nil {
			return model.SchemaContractProfile{}, activityErr
		}
		profile.Indexed = indexed
		profile.Activity = activity
	}

	return profile, nil
}

func (s *SchemaBundleService) GetBundleDetail(
	ctx context.Context,
	network, contractID string,
	version int,
) (model.SchemaBundleDetail, error) {
	id, err := normalizeContractID(contractID)
	if err != nil {
		return model.SchemaBundleDetail{}, err
	}

	resolved, err := s.resolveNetwork(network)
	if err != nil {
		return model.SchemaBundleDetail{}, err
	}
	if version <= 0 {
		return model.SchemaBundleDetail{}, ErrSchemaNotFound
	}

	schemas, err := s.schemas.ListPublishedForContract(ctx, resolved, id)
	if err != nil {
		return model.SchemaBundleDetail{}, err
	}

	bundle, ok := findBundleByVersion(groupIntoBundles(schemas), version)
	if !ok {
		return model.SchemaBundleDetail{}, ErrSchemaNotFound
	}

	events := make([]model.SchemaBundleEvent, 0, len(bundle.Events))
	for _, event := range bundle.Events {
		public := event.Public()
		events = append(events, model.SchemaBundleEvent{
			ID:         public.ID,
			EventName:  public.EventName,
			Version:    public.Version,
			TrustTier:  public.TrustTier,
			SchemaBody: append([]byte(nil), public.SchemaBody...),
			Author:     public.Author,
			CreatedAt:  public.CreatedAt,
			UpdatedAt:  public.UpdatedAt,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].EventName < events[j].EventName
	})

	return model.SchemaBundleDetail{
		ContractID: id,
		Network:    resolved,
		Version:    bundle.Version,
		TrustTier:  bundle.TrustTier,
		Author:     bundle.Author,
		CreatedAt:  bundle.CreatedAt,
		UpdatedAt:  bundle.UpdatedAt,
		VerifiedAt: bundle.VerifiedAt,
		Events:     events,
	}, nil
}

func (s *SchemaBundleService) loadActivity(
	ctx context.Context,
	network, contractID string,
) (*model.SchemaContractActivity, bool, error) {
	detail, err := s.activity.GetContractByID(ctx, network, contractID)
	if err != nil {
		if errors.Is(err, explorerepo.ErrContractNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &model.SchemaContractActivity{
		EventCount:       detail.EventCount,
		TransactionCount: detail.TransactionCount,
		DecodedCount:     detail.DecodedCount,
		Events24h:        detail.Events24h,
		FirstLedger:      detail.FirstLedger,
		LastLedger:       detail.LastLedger,
		LastSeen:         detail.LastSeen,
		SchemaStatus:     detail.SchemaStatus,
	}, true, nil
}

func buildContractSummaries(schemas []model.EventSchema) []model.SchemaContractListItem {
	type contractKey struct {
		contractID string
		network    string
	}

	grouped := make(map[contractKey][]model.EventSchema)
	order := make([]contractKey, 0)

	for _, schema := range schemas {
		if schema.Status != "published" {
			continue
		}
		key := contractKey{contractID: schema.ContractID, network: schema.Network}
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], schema)
	}

	items := make([]model.SchemaContractListItem, 0, len(order))
	for _, key := range order {
		contractSchemas := grouped[key]
		bundles := groupIntoBundles(contractSchemas)
		isVerified, verifiedAt := contractVerification(contractSchemas)

		updatedAt := time.Time{}
		for _, schema := range contractSchemas {
			if schema.UpdatedAt.After(updatedAt) {
				updatedAt = schema.UpdatedAt
			}
		}

		items = append(items, model.SchemaContractListItem{
			ContractID:  key.contractID,
			Network:     key.network,
			SchemaCount: len(bundles),
			EventCount:  len(contractSchemas),
			IsVerified:  isVerified,
			VerifiedAt:  verifiedAt,
			UpdatedAt:   updatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	return items
}

func (s *SchemaBundleService) resolveNetwork(network string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(network))
	if n == "" {
		if s.defaultNetwork == "" {
			return "", ErrInvalidNetwork
		}
		return s.defaultNetwork, nil
	}
	return normalizeNetwork(n)
}
