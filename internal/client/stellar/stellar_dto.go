package stellar

import (
	"encoding/json"
	"time"
)

type FetchEventsInput struct {
	Network     string
	StartLedger uint32
	EndLedger   uint32
	Limit       uint32
	Cursor      string
	ContractIDs []string
}

// ContractEvent is the RPC payload before Atlas level-1/2 normalization.
type ContractEvent struct {
	ID         string
	Network    string
	ContractID string
	Ledger     uint32
	TxnHash    string
	EventType  uint32
	TopicsXDR  []string
	ValueXDR   string
	TopicsJSON []json.RawMessage
	ValueJSON  json.RawMessage
	IngestedAt time.Time
}
