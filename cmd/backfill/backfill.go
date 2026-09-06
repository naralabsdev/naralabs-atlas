package backfill

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/horizon"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	ingestrepo "github.com/naralabs/naralabs-atlas/internal/module/ingest/repository"
	ingestsvc "github.com/naralabs/naralabs-atlas/internal/module/ingest/service"
	"github.com/naralabs/naralabs-atlas/lib/clickhouse"
	"github.com/naralabs/naralabs-atlas/lib/db"
	"github.com/naralabs/naralabs-atlas/lib/logger"
)

var (
	fromLedger uint32
	toLedger   uint32
)

var Cmd = &cobra.Command{
	Use:   "backfill",
	Short: "Backfill historical Soroban events via RPC (+ Horizon metadata)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		log := logger.New(cfg.Log.Level, cfg.Log.JSON)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		pgPool, err := db.Open(cfg)
		if err != nil {
			return err
		}
		defer pgPool.Close()
		if err := db.Migrate(ctx, pgPool, migrationPath("postgres")); err != nil {
			return err
		}

		chConn, err := clickhouse.Open(ctx, cfg)
		if err != nil {
			return err
		}
		defer chConn.Close()
		if err := clickhouse.Migrate(ctx, chConn, migrationPath("clickhouse")); err != nil {
			return err
		}

		stellarClient, err := stellar.NewResilientClient(cfg, log)
		if err != nil {
			return err
		}

		svc := ingestsvc.NewBackfillService(
			cfg,
			log,
			stellarClient,
			horizon.NewClient(cfg.Stellar.HorizonURL),
			ingestrepo.NewEventRepository(chConn),
			ingestrepo.NewDerivedRepository(chConn),
			ingestrepo.NewBackfillRepository(pgPool),
		)
		return svc.Run(ctx, fromLedger, toLedger)
	},
}

func init() {
	Cmd.Flags().Uint32Var(&fromLedger, "from-ledger", 0, "First ledger to backfill")
	Cmd.Flags().Uint32Var(&toLedger, "to-ledger", 0, "Last ledger to backfill (0 = latest)")
}

func migrationPath(name string) string {
	if custom := os.Getenv("MIGRATIONS_DIR"); custom != "" {
		return filepath.Join(custom, name)
	}
	return filepath.Join("db", "migrations", name)
}
