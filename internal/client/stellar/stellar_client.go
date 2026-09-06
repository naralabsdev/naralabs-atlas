// Package stellar wraps the Stellar Soroban RPC client for Atlas ingestion.
package stellar

import (
	"context"
)

// Client defines outbound RPC operations used by the ingest worker.
type Client interface {
	LatestLedger(ctx context.Context) (uint32, error)
	FetchEvents(ctx context.Context, network string, input FetchEventsInput) ([]ContractEvent, uint32, error)
}
