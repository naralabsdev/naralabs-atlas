package service

import (
	"context"

	ingestmodel "github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/lib/realtime"
)

func mapIngestMessage(network string, events []ingestmodel.ContractEvent) realtime.IngestMessage {
	msg := realtime.IngestMessage{
		Network: network,
		Events:  make([]realtime.IngestEvent, 0, len(events)),
	}
	for _, event := range events {
		if event.Ledger > uint32(msg.LastIngestedLedger) {
			msg.LastIngestedLedger = uint64(event.Ledger)
		}
		msg.Events = append(msg.Events, realtime.IngestEvent{
			ID:              event.ID,
			ContractID:      event.ContractID,
			Ledger:          event.Ledger,
			TxnHash:         event.TxnHash,
			TopicsJSON:      event.TopicsJSON,
			ValueJSON:       event.ValueJSON,
			SemanticDecoded: event.SemanticDecoded,
			IngestedAt:      event.IngestedAt,
		})
	}
	return msg
}

func (w *IngestWorkerService) publishIngest(ctx context.Context, events []ingestmodel.ContractEvent) {
	if w.publisher == nil || len(events) == 0 {
		return
	}
	cfg := w.cfgSnapshot()
	msg := mapIngestMessage(cfg.Stellar.Network, events)
	if err := w.publisher.PublishIngest(ctx, msg); err != nil {
		w.log.Warn("realtime publish failed", "error", err, "events", len(events))
	}
}
