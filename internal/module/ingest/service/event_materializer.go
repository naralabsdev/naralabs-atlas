package service

import (
	"github.com/naralabs/naralabs-atlas/internal/client/stellar"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
	"github.com/naralabs/naralabs-atlas/lib/scval"
)

// MaterializeEvents converts RPC events into persisted Atlas rows with
// level-1 raw XDR and level-2 tagged JSON.
func MaterializeEvents(events []stellar.ContractEvent, parser *scval.Parser) []model.ContractEvent {
	if parser == nil {
		parser = scval.DefaultParser
	}

	out := make([]model.ContractEvent, 0, len(events))
	for _, event := range events {
		out = append(out, materializeEvent(event, parser))
	}
	return out
}

func materializeEvent(event stellar.ContractEvent, parser *scval.Parser) model.ContractEvent {
	topicsJSON := parser.NormalizeTopics(event.TopicsXDR, event.TopicsJSON)
	valueJSON := parser.NormalizeValue(event.ValueXDR, event.ValueJSON)

	return model.ContractEvent{
		ID:              event.ID,
		Network:         event.Network,
		ContractID:      event.ContractID,
		Ledger:          event.Ledger,
		TxnHash:         event.TxnHash,
		EventType:       event.EventType,
		TopicsXDR:       append([]string(nil), event.TopicsXDR...),
		ValueXDR:        event.ValueXDR,
		TopicsJSON:      string(topicsJSON),
		ValueJSON:       string(valueJSON),
		SemanticDecoded: false,
		IngestedAt:      event.IngestedAt,
	}
}
