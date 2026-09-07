// Package server wires dependencies and runs the Atlas HTTP API.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/routes"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/service"
	"github.com/naralabs/naralabs-atlas/lib/clickhouse"
	"github.com/naralabs/naralabs-atlas/lib/db"
	"github.com/naralabs/naralabs-atlas/lib/logger"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Run the public HTTP API (read path)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return Run(cmd.Context())
		},
	}
}

func Run(ctx context.Context) error {
	cfg := config.Get()
	log := logger.New(cfg.Log.Level, cfg.Log.JSON)
	log.Info("starting atlas api server", "env", cfg.Env, "network", cfg.Stellar.Network)

	pg, err := db.Open(cfg)
	if err != nil {
		return err
	}
	defer pg.Close()

	chConn, err := clickhouse.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer chConn.Close()

	stellarClient, err := stellar.NewResilientClient(cfg, log)
	if err != nil {
		return err
	}

	repo := repository.NewExploreRepository(chConn, pg)
	svc := service.NewExploreService(repo, stellarClient)
	exploreHandler := handler.NewExploreHandler(svc, cfg.Stellar.Network)
	router := routes.NewRouter(cfg, exploreHandler)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-stop:
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	log.Info("atlas api server stopped")
	return nil
}
