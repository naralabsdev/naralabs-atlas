package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/horizon"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/repository"
	"github.com/naralabs/naralabs-atlas/lib/scval"
)

type BackfillService struct {
	cfg          *config.Config
	log          *slog.Logger
	stellar      stellar.Client
	horizon      *horizon.Client
	eventRepo    repository.EventRepository
	derivedRepo  repository.DerivedRepository
	backfillRepo repository.BackfillRepository
}

func NewBackfillService(
	cfg *config.Config,
	log *slog.Logger,
	stellarClient stellar.Client,
	horizonClient *horizon.Client,
	eventRepo repository.EventRepository,
	derivedRepo repository.DerivedRepository,
	backfillRepo repository.BackfillRepository,
) *BackfillService {
	return &BackfillService{
		cfg:          cfg,
		log:          log,
		stellar:      stellarClient,
		horizon:      horizonClient,
		eventRepo:    eventRepo,
		derivedRepo:  derivedRepo,
		backfillRepo: backfillRepo,
	}
}

func (s *BackfillService) Run(ctx context.Context, fromLedger, toLedger uint32) error {
	if toLedger > 0 && fromLedger > toLedger {
		return fmt.Errorf("from_ledger must be <= to_ledger")
	}
	if toLedger == 0 {
		latest, err := s.stellar.LatestLedger(ctx)
		if err != nil {
			return err
		}
		toLedger = latest
	}

	state, err := s.backfillRepo.Start(ctx, s.cfg.Stellar.Network, fromLedger, toLedger)
	if err != nil {
		return err
	}

	cursor := state.NextLedger
	if cursor == 0 {
		cursor = fromLedger
	}

	interval := s.cfg.Backfill.Interval
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	batchSize := s.cfg.Backfill.BatchSize
	if batchSize <= 0 {
		batchSize = 200
	}

	for cursor <= toLedger {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := cursor + uint32(batchSize) - 1
		if end > toLedger {
			end = toLedger
		}

		if err := s.backfillLedgerWindow(ctx, cursor, end); err != nil {
			return err
		}
		if err := s.backfillRepo.UpdateProgress(ctx, s.cfg.Stellar.Network, end+1); err != nil {
			return err
		}
		cursor = end + 1

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	if err := s.backfillRepo.Complete(ctx, s.cfg.Stellar.Network); err != nil {
		return err
	}
	s.log.Info("backfill completed",
		"network", s.cfg.Stellar.Network,
		"from_ledger", fromLedger,
		"to_ledger", toLedger,
	)
	return nil
}

func (s *BackfillService) backfillLedgerWindow(ctx context.Context, fromLedger, toLedger uint32) error {
	batches := stellar.BuildFilterBatches(s.cfg.WatchedContractIDs(), 25)
	for _, batch := range batches {
		events, _, err := s.stellar.FetchEvents(ctx, s.cfg.Stellar.Network, stellar.FetchEventsInput{
			StartLedger: fromLedger,
			EndLedger:   toLedger,
			Limit:       uint32(s.cfg.Ingest.PageLimit),
			ContractIDs: batch,
		})
		if err != nil {
			return err
		}
		materialized := MaterializeEvents(events, scval.DefaultParser)
		if len(materialized) == 0 {
			continue
		}
		if err := s.eventRepo.UpsertBatch(ctx, materialized); err != nil {
			return err
		}
		addrs, tokens := ExtractDerived(materialized)
		if len(addrs) > 0 {
			if err := s.derivedRepo.UpsertAddresses(ctx, addrs); err != nil {
				return err
			}
		}
		if len(tokens) > 0 {
			if err := s.derivedRepo.UpsertTokenEvents(ctx, tokens); err != nil {
				return err
			}
		}
	}

	// Horizon pass helps discover transaction hashes for ledgers with sparse RPC coverage.
	if s.horizon != nil {
		txs, err := s.horizon.TransactionsByLedgerRange(ctx, int64(fromLedger), int64(toLedger), s.cfg.Backfill.BatchSize)
		if err != nil {
			s.log.Warn("horizon backfill lookup failed", "error", err)
			return nil
		}
		s.log.Debug("horizon transactions scanned", "count", len(txs), "from", fromLedger, "to", toLedger)
	}
	return nil
}
