package realtime

import (
	"encoding/json"
	"time"
)

const (
	MsgHomeSnapshot        = "home.snapshot"
	MsgHomeStatsUpdated    = "home.stats_updated"
	MsgHomeEventsIngested  = "home.events_ingested"
	MsgHomeContractsUpdated = "home.contracts_updated"
	MsgPing                = "ping"
	MsgPong                = "pong"
)

// IngestMessage is published by the ingest worker after events are persisted.
type IngestMessage struct {
	Network            string         `json:"network"`
	LastIngestedLedger uint64         `json:"last_ingested_ledger"`
	Events             []IngestEvent  `json:"events"`
}

type IngestEvent struct {
	ID              string    `json:"id"`
	ContractID      string    `json:"contract_id"`
	Ledger          uint32    `json:"ledger"`
	TxnHash         string    `json:"txn_hash"`
	TopicsJSON      string    `json:"topics_json"`
	ValueJSON       string    `json:"value_json"`
	SemanticDecoded bool      `json:"semantic_decoded"`
	IngestedAt      time.Time `json:"ingested_at"`
}

// ClientMessage is sent from the Atlas server to WebSocket clients.
type ClientMessage struct {
	Type    string          `json:"type"`
	Network string          `json:"network,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func EncodeClientMessage(msg ClientMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func EncodeIngestMessage(msg IngestMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func DecodeIngestMessage(data []byte) (IngestMessage, error) {
	var msg IngestMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return IngestMessage{}, err
	}
	return msg, nil
}
