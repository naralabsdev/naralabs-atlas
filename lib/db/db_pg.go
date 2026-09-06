package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/config"
)

func init() {
	factories = append(factories, &factory{
		prefixes: []string{"postgres://", "postgresql://"},
		opener: func(cfg *config.Config) (*pgxpool.Pool, error) {
			pool, err := pgxpool.New(context.Background(), cfg.DB.URI)
			if err != nil {
				return nil, fmt.Errorf("connect postgres: %w", err)
			}

			if err := pool.Ping(context.Background()); err != nil {
				pool.Close()
				return nil, fmt.Errorf("ping postgres: %w", err)
			}

			slog.Info("postgres connected successfully")
			return pool, nil
		},
	})
}
