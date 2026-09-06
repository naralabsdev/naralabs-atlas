package clickhouse

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/naralabs/naralabs-atlas/config"
)

// Open creates a ClickHouse connection from the configured DSN.
func Open(ctx context.Context, cfg *config.Config) (clickhouse.Conn, error) {
	opts, err := clickhouse.ParseDSN(cfg.ClickHouse.URI)
	if err != nil {
		return nil, fmt.Errorf("parse clickhouse dsn: %w", err)
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	slog.Info("clickhouse connected successfully")
	return conn, nil
}
