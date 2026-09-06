// Package ingest defines the Soroban event ingestion bounded context.
package ingest

import "context"

// Worker defines the background ingest loop contract.
type Worker interface {
	Run(ctx context.Context) error
}
