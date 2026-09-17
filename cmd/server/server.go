// Package server wires dependencies and runs the Atlas HTTP API.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	authhandler "github.com/naralabs/naralabs-atlas/internal/module/auth/handler"
	authrepo "github.com/naralabs/naralabs-atlas/internal/module/auth/repository"
	authservice "github.com/naralabs/naralabs-atlas/internal/module/auth/service"
	decoderhandler "github.com/naralabs/naralabs-atlas/internal/module/decoder/handler"
	decoderservice "github.com/naralabs/naralabs-atlas/internal/module/decoder/service"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/routes"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/service"
	registryhandler "github.com/naralabs/naralabs-atlas/internal/module/registry/handler"
	registryrepo "github.com/naralabs/naralabs-atlas/internal/module/registry/repository"
	registryservice "github.com/naralabs/naralabs-atlas/internal/module/registry/service"
	"github.com/naralabs/naralabs-atlas/lib/clickhouse"
	"github.com/naralabs/naralabs-atlas/lib/db"
	mailemail "github.com/naralabs/naralabs-atlas/lib/email"
	"github.com/naralabs/naralabs-atlas/lib/logger"
	"github.com/naralabs/naralabs-atlas/lib/realtime"
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

	if err := db.Migrate(ctx, pg, migrationPath("postgres")); err != nil {
		return fmt.Errorf("postgres migration: %w", err)
	}

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

	authRepo := authrepo.NewAuthRepository(pg)
	tokenRepo := authrepo.NewPublishTokenRepository(pg)
	apiKeyRepo := authrepo.NewAPIKeyRepository(pg)
	authMailer := newAuthMailer(cfg)
	authSvc := authservice.NewAuthService(cfg, authRepo, authMailer)
	tokenSvc := authservice.NewPublishTokenService(tokenRepo, authSvc)
	apiKeySvc := authservice.NewAPIKeyService(apiKeyRepo, authSvc)
	authHandler := authhandler.NewAuthHandler(authSvc)
	tokenHandler := authhandler.NewPublishTokenHandler(tokenSvc)
	apiKeyHandler := authhandler.NewAPIKeyHandler(apiKeySvc)

	exploreHandler := handler.NewExploreHandler(svc, cfg.Stellar.Network, apiKeySvc)

	schemaRepo := registryrepo.NewSchemaRepository(pg)
	projectRepo := registryrepo.NewProjectRepository(pg)
	challengeRepo := registryrepo.NewVerifyChallengeRepository(pg)
	authorityRepo := registryrepo.NewContractAuthorityRepository(pg)
	schemaSvc := registryservice.NewSchemaService(schemaRepo, cfg.Stellar.Network)
	projectSvc := registryservice.NewProjectService(
		projectRepo,
		registryservice.NewTokenCreatorAdapter(tokenSvc),
		authSvc,
	)
	authoritySvc := registryservice.NewContractAuthorityService(authorityRepo)
	verifySvc := registryservice.NewVerifyService(
		projectRepo,
		schemaRepo,
		challengeRepo,
		authoritySvc,
		authSvc,
		cfg.Stellar.Network,
	)
	registryHandler := registryhandler.NewSchemaHandler(schemaSvc, tokenSvc, authSvc, projectSvc)
	projectHandler := registryhandler.NewProjectHandler(projectSvc, verifySvc)
	bundleSvc := registryservice.NewSchemaBundleService(schemaRepo, repo, cfg.Stellar.Network)
	bundleHandler := registryhandler.NewBundleHandler(bundleSvc)

	decodeSchemaLookup := decoderservice.NewSchemaRepositoryAdapter(schemaRepo)
	decodeSvc := decoderservice.NewDecodeService(decodeSchemaLookup)
	decodeHandler := decoderhandler.NewDecodeHandler(decodeSvc, apiKeySvc)

	hub := realtime.NewHub(cfg.Realtime.MaxClients)
	var subscriber *realtime.RedisSubscriber
	var wsHandler *handler.HomeWebSocketHandler
	if cfg.Realtime.Enabled {
		subscriber, err = realtime.NewSubscriber(cfg, log)
		if err != nil {
			return err
		}
		wsHandler = handler.NewHomeWebSocketHandler(cfg, svc, hub, log)
	}

	router := routes.NewRouter(
		cfg,
		exploreHandler,
		wsHandler,
		authHandler,
		tokenHandler,
		apiKeyHandler,
		decodeHandler,
		registryHandler,
		projectHandler,
		bundleHandler,
	)

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	if subscriber != nil && wsHandler != nil {
		go func() {
			if err := subscriber.Run(runCtx, wsHandler.HandleIngest); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn("realtime subscriber stopped", "error", err)
			}
		}()
		defer func() {
			runCancel()
			if err := subscriber.Close(); err != nil {
				log.Warn("realtime subscriber close failed", "error", err)
			}
			hub.Close()
		}()
	}

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

func migrationPath(name string) string {
	if custom := os.Getenv("MIGRATIONS_DIR"); custom != "" {
		return filepath.Join(custom, name)
	}
	return filepath.Join("db", "migrations", name)
}

func newAuthMailer(cfg *config.Config) mailemail.Sender {
	if cfg.Auth.ResendAPIKey != "" {
		return mailemail.NewResendSender(cfg.Auth.ResendAPIKey, cfg.Auth.EmailFrom)
	}
	return mailemail.LogSender{}
}
