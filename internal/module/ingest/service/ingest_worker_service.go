package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/repository"
	"github.com/naralabs/naralabs-atlas/lib/rpcchain"
	"github.com/naralabs/naralabs-atlas/lib/scval"
)

type IngestWorkerService struct {
	cfg atomic.Pointer[config.Config]
	log *slog.Logger

	stellar     stellar.Client
	cursorRepo  repository.CursorRepository
	eventRepo   repository.EventRepository
	derivedRepo repository.DerivedRepository

	pollInterval atomic.Int64
	lastReorg    atomic.Int64
}

func NewIngestWorkerService(
	cfg *config.Config,
	log *slog.Logger,
	stellarClient stellar.Client,
	cursorRepo repository.CursorRepository,
	eventRepo repository.EventRepository,
	derivedRepo repository.DerivedRepository,
) *IngestWorkerService {
	w := &IngestWorkerService{
		log:         log,
		stellar:     stellarClient,
		cursorRepo:  cursorRepo,
		eventRepo:   eventRepo,
		derivedRepo: derivedRepo,
	}
	w.SetConfig(cfg)
	return w
}

func (w *IngestWorkerService) SetConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}
	w.cfg.Store(cfg)
	w.pollInterval.Store(cfg.Ingest.PollInterval.Nanoseconds())
}

func (w *IngestWorkerService) cfgSnapshot() *config.Config {
	return w.cfg.Load()
}

func (w *IngestWorkerService) Run(ctx context.Context) error {
	cfg := w.cfgSnapshot()
	w.log.Info("atlas ingest worker started",
		"service", cfg.ServiceName,
		"network", cfg.Stellar.Network,
		"poll_interval", cfg.Ingest.PollInterval.String(),
	)

	if _, err := w.ingestOnce(ctx); err != nil {
		w.log.Error("initial ingest cycle failed", "error", err)
	}

	for {
		interval := time.Duration(w.pollInterval.Load())
		if interval <= 0 {
			interval = w.cfgSnapshot().Ingest.PollInterval
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			w.log.Info("atlas ingest worker stopped")
			return ctx.Err()
		case <-timer.C:
			caughtUp, err := w.ingestOnce(ctx)
			if err != nil {
				w.log.Error("ingest cycle failed", "error", err)
			}
			cfg = w.cfgSnapshot()
			next := NextAdaptivePoll(interval, cfg.Ingest.PollIntervalMin, cfg.Ingest.PollIntervalMax, caughtUp)
			w.pollInterval.Store(next.Nanoseconds())
		}
	}
}

func (w *IngestWorkerService) ingestOnce(ctx context.Context) (bool, error) {
	cfg := w.cfgSnapshot()
	state, err := w.cursorRepo.Get(ctx, cfg.Stellar.Network)
	if err != nil {
		return false, err
	}

	latestLedger, err := w.stellar.LatestLedger(ctx)
	if err != nil {
		return false, err
	}

	startLedger, err := ResolveColdStart(
		state.LastLedger,
		latestLedger,
		cfg.Ingest.RetentionLedgers,
		cfg.Ingest.StartLedgerRaw,
	)
	if err != nil {
		return false, err
	}

	if startLedger >= latestLedger {
		w.log.Debug("already caught up",
			"last_ledger", state.LastLedger,
			"latest_ledger", latestLedger,
		)
		return true, nil
	}

	if err := w.maybeRescanReorg(ctx, cfg, state, startLedger); err != nil {
		return false, err
	}

	batches := stellar.BuildFilterBatches(cfg.WatchedContractIDs(), 25)
	totalEvents := 0
	cursorLedger := startLedger

	for _, batch := range batches {
		events, nextLedger, fetchErr := w.stellar.FetchEvents(ctx, cfg.Stellar.Network, stellar.FetchEventsInput{
			StartLedger: startLedger,
			EndLedger:   latestLedger,
			Limit:       uint32(cfg.Ingest.PageLimit),
			ContractIDs: batch,
		})
		if rpcchain.IsReanchorCursor(fetchErr) {
			w.log.Warn("rpc endpoint switched during pagination, re-anchoring",
				"from_ledger", startLedger,
				"batch", stellar.FilterBatchLabel(batch),
			)
			return false, nil
		}
		if fetchErr != nil {
			return false, fetchErr
		}

		materialized := MaterializeEvents(events, scval.DefaultParser)
		if err := w.persist(ctx, materialized); err != nil {
			return false, err
		}
		totalEvents += len(materialized)
		if nextLedger > cursorLedger {
			cursorLedger = nextLedger
		}
	}

	nextLedger := cursorLedger
	if nextLedger <= state.LastLedger {
		nextLedger = latestLedger
	}

	if err := w.cursorRepo.Upsert(ctx, cfg.Stellar.Network, nextLedger, ""); err != nil {
		return false, err
	}

	w.log.Info("ingest cycle completed",
		"events", totalEvents,
		"from_ledger", startLedger,
		"to_ledger", nextLedger,
		"latest_ledger", latestLedger,
		"scval_failures", scval.FailureCount(),
	)

	return nextLedger >= latestLedger, nil
}

func (w *IngestWorkerService) persist(ctx context.Context, events []model.ContractEvent) error {
	if len(events) == 0 {
		return nil
	}
	cfg := w.cfgSnapshot()
	chunkSize := cfg.Ingest.BatchSize
	if chunkSize <= 0 {
		chunkSize = len(events)
	}
	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize
		if end > len(events) {
			end = len(events)
		}
		chunk := events[i:end]
		if err := w.eventRepo.UpsertBatch(ctx, chunk); err != nil {
			return err
		}
		addrs, tokens := ExtractDerived(chunk)
		if len(addrs) > 0 {
			if err := w.derivedRepo.UpsertAddresses(ctx, addrs); err != nil {
				return err
			}
		}
		if len(tokens) > 0 {
			if err := w.derivedRepo.UpsertTokenEvents(ctx, tokens); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *IngestWorkerService) maybeRescanReorg(
	ctx context.Context,
	cfg *config.Config,
	state model.IngestState,
	startLedger uint32,
) error {
	if cfg.Ingest.ReorgWindow == 0 {
		return nil
	}
	now := time.Now().UnixNano()
	last := w.lastReorg.Load()
	if last > 0 && time.Duration(now-last) < cfg.Ingest.ReorgInterval {
		return nil
	}
	reorgFrom := ReorgFromLedger(state.LastLedger, cfg.Ingest.ReorgWindow)
	if reorgFrom == 0 || reorgFrom >= startLedger {
		return nil
	}

	w.log.Info("reorg rescan starting", "from_ledger", reorgFrom, "to_ledger", startLedger)
	batches := stellar.BuildFilterBatches(cfg.WatchedContractIDs(), 25)
	for _, batch := range batches {
		events, _, err := w.stellar.FetchEvents(ctx, cfg.Stellar.Network, stellar.FetchEventsInput{
			StartLedger: reorgFrom,
			EndLedger:   startLedger,
			Limit:       uint32(cfg.Ingest.PageLimit),
			ContractIDs: batch,
		})
		if err != nil {
			return err
		}
		if err := w.persist(ctx, MaterializeEvents(events, scval.DefaultParser)); err != nil {
			return err
		}
	}
	w.lastReorg.Store(now)
	return nil
}
