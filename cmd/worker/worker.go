// Package worker wires dependencies and runs the Atlas ingest worker.
package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	ingestrepo "github.com/naralabs/naralabs-atlas/internal/module/ingest/repository"
	ingestsvc "github.com/naralabs/naralabs-atlas/internal/module/ingest/service"
	"github.com/naralabs/naralabs-atlas/lib/clickhouse"
	"github.com/naralabs/naralabs-atlas/lib/db"
	"github.com/naralabs/naralabs-atlas/lib/logger"
)

// Run starts the ingest worker with all dependencies wired.
func Run(ctx context.Context) error {
	cfg := config.Get()
	log := logger.New(cfg.Log.Level, cfg.Log.JSON)
	log.Info("starting naralabs atlas", "env", cfg.Env, "network", cfg.Stellar.Network)

	pgPool, err := db.Open(cfg)
	if err != nil {
		log.Error("postgres connection failed", "error", err)
		return err
	}
	defer pgPool.Close()

	if err := db.Migrate(ctx, pgPool, migrationPath("postgres")); err != nil {
		log.Error("postgres migration failed", "error", err)
		return err
	}

	chConn, err := clickhouse.Open(ctx, cfg)
	if err != nil {
		log.Error("clickhouse connection failed", "error", err)
		return err
	}
	defer chConn.Close()

	if err := clickhouse.Migrate(ctx, chConn, migrationPath("clickhouse")); err != nil {
		log.Error("clickhouse migration failed", "error", err)
		return err
	}

	stellarClient, err := stellar.NewResilientClient(cfg, log)
	if err != nil {
		log.Error("stellar client init failed", "error", err)
		return err
	}

	worker := ingestsvc.NewIngestWorkerService(
		cfg,
		log,
		stellarClient,
		ingestrepo.NewCursorRepository(pgPool),
		ingestrepo.NewEventRepository(chConn),
		ingestrepo.NewDerivedRepository(chConn),
	)

	go watchConfigReload(ctx, log, worker)

	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("worker stopped with error", "error", err)
		return err
	}

	return nil
}

func watchConfigReload(ctx context.Context, log *slog.Logger, worker *ingestsvc.IngestWorkerService) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			if err := config.Reload(); err != nil {
				log.Error("config reload via SIGHUP rejected", "error", err)
				continue
			}
			cfg := config.Get()
			worker.SetConfig(cfg)
			log.Info("config reloaded via SIGHUP",
				"poll_interval", cfg.Ingest.PollInterval.String(),
				"watched_contracts", cfg.Ingest.WatchedContracts,
			)
		}
	}
}

func migrationPath(name string) string {
	if custom := os.Getenv("MIGRATIONS_DIR"); custom != "" {
		return filepath.Join(custom, name)
	}

	return filepath.Join("db", "migrations", name)
}
