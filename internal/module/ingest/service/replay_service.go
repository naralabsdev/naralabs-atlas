package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/repository"
	"github.com/naralabs/naralabs-atlas/lib/scval"
)

type ReplayService struct {
	cfg       *config.Config
	log       *slog.Logger
	eventRepo repository.EventRepository
	derived   repository.DerivedRepository
}

func NewReplayService(
	cfg *config.Config,
	log *slog.Logger,
	eventRepo repository.EventRepository,
	derived repository.DerivedRepository,
) *ReplayService {
	return &ReplayService{
		cfg:       cfg,
		log:       log,
		eventRepo: eventRepo,
		derived:   derived,
	}
}

func (s *ReplayService) Run(ctx context.Context, fromLedger, toLedger uint32) error {
	if toLedger > 0 && fromLedger > toLedger {
		return fmt.Errorf("from_ledger must be <= to_ledger")
	}

	limit := s.cfg.Replay.BatchSize
	if limit <= 0 {
		limit = 500
	}

	cursor := fromLedger
	total := 0
	for {
		if toLedger > 0 && cursor > toLedger {
			break
		}
		upper := cursor + uint32(limit) - 1
		if toLedger > 0 && upper > toLedger {
			upper = toLedger
		}

		events, err := s.eventRepo.ListByLedgerRange(ctx, s.cfg.Stellar.Network, cursor, upper, limit)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			if toLedger > 0 && cursor >= toLedger {
				break
			}
			if toLedger == 0 {
				break
			}
			cursor = upper + 1
			continue
		}

		rematerialized := rematerialize(events)
		if err := s.eventRepo.UpsertBatch(ctx, rematerialized); err != nil {
			return err
		}
		addrs, tokens := ExtractDerived(rematerialized)
		if len(addrs) > 0 {
			if err := s.derived.UpsertAddresses(ctx, addrs); err != nil {
				return err
			}
		}
		if len(tokens) > 0 {
			if err := s.derived.UpsertTokenEvents(ctx, tokens); err != nil {
				return err
			}
		}

		total += len(rematerialized)
		last := rematerialized[len(rematerialized)-1].Ledger
		cursor = last
		if last >= upper {
			cursor = upper + 1
		}
	}

	s.log.Info("decoder replay completed",
		"network", s.cfg.Stellar.Network,
		"from_ledger", fromLedger,
		"to_ledger", toLedger,
		"events", total,
		"scval_failures", scval.FailureCount(),
	)
	return nil
}

func rematerialize(events []model.ContractEvent) []model.ContractEvent {
	parser := scval.DefaultParser
	out := make([]model.ContractEvent, 0, len(events))
	for _, event := range events {
		topicsJSON := parser.NormalizeTopics(event.TopicsXDR, nil)
		valueJSON := parser.NormalizeValue(event.ValueXDR, nil)
		out = append(out, model.ContractEvent{
			ID:              event.ID,
			Network:         event.Network,
			ContractID:      event.ContractID,
			Ledger:          event.Ledger,
			TxnHash:         event.TxnHash,
			EventType:       event.EventType,
			TopicsXDR:       append([]string(nil), event.TopicsXDR...),
			ValueXDR:        event.ValueXDR,
			TopicsJSON:      string(topicsJSON),
			ValueJSON:       string(valueJSON),
			SemanticDecoded: event.SemanticDecoded,
			IngestedAt:      event.IngestedAt,
		})
	}
	return out
}
