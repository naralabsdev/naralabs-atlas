package db

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naralabs/naralabs-atlas/config"
)

type factory struct {
	prefixes []string
	opener   func(*config.Config) (*pgxpool.Pool, error)
}

var factories []*factory

func allPrefixes() string {
	prefixes := make([]string, 0, len(factories)*2)
	for _, f := range factories {
		prefixes = append(prefixes, f.prefixes...)
	}
	return strings.Join(prefixes, "|")
}

// Open establishes a Postgres connection pool based on the configured DSN.
func Open(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := cfg.DB.URI
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_URL is required")
	}

	for _, f := range factories {
		for _, prefix := range f.prefixes {
			if strings.HasPrefix(dsn, prefix) {
				return f.opener(cfg)
			}
		}
	}

	return nil, fmt.Errorf(
		"invalid database connection string %s, only (%s) is supported",
		dsn,
		allPrefixes(),
	)
}
