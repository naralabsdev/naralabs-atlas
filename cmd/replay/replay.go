package replay

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/repository"
	ingestsvc "github.com/naralabs/naralabs-atlas/internal/module/ingest/service"
	"github.com/naralabs/naralabs-atlas/lib/clickhouse"
	"github.com/naralabs/naralabs-atlas/lib/logger"
)

var (
	fromLedger uint32
	toLedger   uint32
)

var Cmd = &cobra.Command{
	Use:   "replay",
	Short: "Re-run level-2 decoder over stored events (idempotent upsert)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		log := logger.New(cfg.Log.Level, cfg.Log.JSON)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		chConn, err := clickhouse.Open(ctx, cfg)
		if err != nil {
			return err
		}
		defer chConn.Close()

		if err := clickhouse.Migrate(ctx, chConn, migrationPath("clickhouse")); err != nil {
			return err
		}

		svc := ingestsvc.NewReplayService(
			cfg,
			log,
			repository.NewEventRepository(chConn),
			repository.NewDerivedRepository(chConn),
		)
		return svc.Run(ctx, fromLedger, toLedger)
	},
}

func init() {
	Cmd.Flags().Uint32Var(&fromLedger, "from-ledger", 0, "First ledger to replay")
	Cmd.Flags().Uint32Var(&toLedger, "to-ledger", 0, "Last ledger to replay (0 = open-ended)")
}

func migrationPath(name string) string {
	if custom := os.Getenv("MIGRATIONS_DIR"); custom != "" {
		return filepath.Join(custom, name)
	}
	return filepath.Join("db", "migrations", name)
}
