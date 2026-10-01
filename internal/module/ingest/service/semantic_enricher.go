package service

import (
	"context"
	"encoding/json"
	"time"

	decoderservice "github.com/naralabs/naralabs-atlas/internal/module/decoder/service"
	"github.com/naralabs/naralabs-atlas/internal/module/ingest/model"
)

// SemanticEnricher applies registry-backed level-3 decode to materialized events.
type SemanticEnricher struct {
	decode *decoderservice.DecodeService
}

func NewSemanticEnricher(decode *decoderservice.DecodeService) *SemanticEnricher {
	if decode == nil {
		return nil
	}
	return &SemanticEnricher{decode: decode}
}

// ApplyBatch mutates events in place, setting semantic columns when a schema matches.
func (e *SemanticEnricher) ApplyBatch(ctx context.Context, events []model.ContractEvent, bumpIngestedAt bool) {
	if e == nil || len(events) == 0 {
		return
	}
	now := time.Now().UTC()
	for i := range events {
		e.applyOne(ctx, &events[i])
		if bumpIngestedAt {
			events[i].IngestedAt = now
		}
	}
}

func (e *SemanticEnricher) applyOne(ctx context.Context, event *model.ContractEvent) {
	if event == nil || e.decode == nil {
		return
	}
	var topics []json.RawMessage
	if err := json.Unmarshal([]byte(event.TopicsJSON), &topics); err != nil {
		return
	}
	value := json.RawMessage(event.ValueJSON)

	result, err := e.decode.Decode(ctx, decoderservice.DecodeEventInput{
		Network:    event.Network,
		ContractID: event.ContractID,
		TopicsXDR:  event.TopicsXDR,
		TopicsJSON: topics,
		ValueXDR:   event.ValueXDR,
		ValueJSON:  value,
	})
	if err != nil {
		return
	}

	event.SemanticDecoded = result.DecodeStatus == "decoded"
	event.DecodedEventName = result.EventName
	if result.SchemaVersion > 0 {
		event.SchemaVersion = uint16(result.SchemaVersion)
	}
	event.DecodeSummary = result.Summary
	if len(result.Fields) > 0 {
		raw, err := json.Marshal(result.Fields)
		if err == nil {
			event.DecodedFieldsJSON = string(raw)
		}
	}
}
