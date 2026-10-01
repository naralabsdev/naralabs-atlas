package wiring

import (
	decoderservice "github.com/naralabs/naralabs-atlas/internal/module/decoder/service"
	ingestsvc "github.com/naralabs/naralabs-atlas/internal/module/ingest/service"
	registryrepo "github.com/naralabs/naralabs-atlas/internal/module/registry/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewSemanticEnricher wires registry lookup into the ingest semantic decode path.
func NewSemanticEnricher(pg *pgxpool.Pool) *ingestsvc.SemanticEnricher {
	if pg == nil {
		return nil
	}
	schemaRepo := registryrepo.NewSchemaRepository(pg)
	lookup := decoderservice.NewSchemaRepositoryAdapter(schemaRepo)
	decode := decoderservice.NewDecodeService(lookup)
	return ingestsvc.NewSemanticEnricher(decode)
}
