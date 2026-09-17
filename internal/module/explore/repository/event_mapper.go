package repository

import (
	"time"

	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
)

// MapStoredEvent builds an explorer EventItem from persisted event columns.
func MapStoredEvent(
	id, contractID, txnHash string,
	ledger uint32,
	topicsJSON, valueJSON string,
	semanticDecoded bool,
	ingestedAt time.Time,
) model.EventItem {
	var decoded uint8
	if semanticDecoded {
		decoded = 1
	}
	eventType := eventTypeFromTopics(topicsJSON)
	return model.EventItem{
		ID:             id,
		ContractID:     contractID,
		Ledger:         ledger,
		TxnHash:        txnHash,
		IngestedAt:     ingestedAt,
		EventType:      eventType,
		SummaryPreview: summaryPreview(eventType, topicsJSON, valueJSON),
		DecodeStatus:   decodeStatus(decoded),
	}
}
