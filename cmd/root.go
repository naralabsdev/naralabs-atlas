package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/naralabs/naralabs-atlas/cmd/backfill"
	"github.com/naralabs/naralabs-atlas/cmd/replay"
	"github.com/naralabs/naralabs-atlas/cmd/worker"
	"github.com/naralabs/naralabs-atlas/config"
	"github.com/spf13/cobra"
)

const version = "0.2.0"

var rootCmd = &cobra.Command{
	Use:   "atlas",
	Short: "NaraLabs Atlas — Soroban event indexer",
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run the Soroban event ingest worker",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Reload(); err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		return worker.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(replay.Cmd)
	rootCmd.AddCommand(backfill.Cmd)
	rootCmd.Version = version
}

// Execute runs the CLI entrypoint.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
