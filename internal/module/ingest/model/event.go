package model

import "time"

type IngestState struct {
	ID            int32
	Network       string
	LastLedger    uint32
	LastRPCCursor string
	UpdatedAt     time.Time
}

// ContractEvent is a Soroban contract event persisted by Atlas.
type ContractEvent struct {
	ID              string
	Network         string
	ContractID      string
	Ledger          uint32
	TxnHash         string
	EventType       uint32
	TopicsXDR       []string
	ValueXDR        string
	TopicsJSON      string
	ValueJSON       string
	SemanticDecoded bool
	IngestedAt      time.Time
}

type EventAddress struct {
	Network    string
	Address    string
	ContractID string
	EventID    string
	Ledger     uint32
	Role       string
	IngestedAt time.Time
}

type TokenEvent struct {
	Network     string
	ContractID  string
	EventID     string
	Ledger      uint32
	TokenSymbol string
	TokenAmount string
	FromAddress string
	ToAddress   string
	Action      string
	IngestedAt  time.Time
}

type BackfillState struct {
	ID          int32
	Network     string
	FromLedger  uint32
	ToLedger    uint32
	NextLedger  uint32
	Status      string
	StartedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}
