package model

import (
	"encoding/json"
	"time"
)

type ActivityBucket struct {
	Bucket time.Time `json:"bucket"`
	Count  uint64    `json:"count"`
}

type NetworkStats struct {
	Network            string           `json:"network"`
	TotalEvents        uint64           `json:"total_events"`
	ContractCount      uint64           `json:"contract_count"`
	LastIngestedLedger uint64           `json:"last_ingested_ledger"`
	ChainHeadLedger    *uint64          `json:"chain_head_ledger,omitempty"`
	IngestLagLedgers   *uint64          `json:"ingest_lag_ledgers,omitempty"`
	Events24h          uint64           `json:"events_24h"`
	LastIndexedAt      *time.Time       `json:"last_indexed_at,omitempty"`
	OldestStoredLedger *uint64          `json:"oldest_stored_ledger,omitempty"`
	Activity           []ActivityBucket `json:"activity"`
}

type EventItem struct {
	ID             string    `json:"id"`
	ContractID     string    `json:"contract_id"`
	Ledger         uint32    `json:"ledger"`
	TxnHash        string    `json:"txn_hash"`
	IngestedAt     time.Time `json:"ingested_at"`
	EventType      string    `json:"event_type"`
	SummaryPreview string    `json:"summary_preview"`
	DecodeStatus   string    `json:"decode_status"`
}

type ContractItem struct {
	ContractID   string    `json:"contract_id"`
	DisplayName  *string   `json:"display_name"`
	EventCount   uint64    `json:"event_count"`
	FirstLedger  uint32    `json:"first_ledger"`
	LastLedger   uint32    `json:"last_ledger"`
	LastSeen     time.Time `json:"last_seen"`
	SchemaStatus string    `json:"schema_status"`
}

type EventTypeCount struct {
	EventType string `json:"event_type"`
	Count     uint64 `json:"count"`
}

// ContractDetail is the explorer payload for a single Soroban contract.
type ContractDetail struct {
	ContractID       string           `json:"contract_id"`
	Network          string           `json:"network"`
	DisplayName      *string          `json:"display_name,omitempty"`
	EventCount       uint64           `json:"event_count"`
	TransactionCount uint64           `json:"transaction_count"`
	DecodedCount     uint64           `json:"decoded_count"`
	Events24h        uint64           `json:"events_24h"`
	FirstLedger      uint32           `json:"first_ledger"`
	LastLedger       uint32           `json:"last_ledger"`
	LastSeen         time.Time        `json:"last_seen"`
	SchemaStatus     string           `json:"schema_status"`
	TypeBreakdown []EventTypeCount `json:"type_breakdown"`
}

type PaginatedListResponse[T any] struct {
	Items    []T    `json:"items"`
	Total    uint64 `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type HomePayload struct {
	Stats           NetworkStats   `json:"stats"`
	RecentEvents    []EventItem    `json:"recent_events"`
	ActiveContracts []ContractItem `json:"active_contracts"`
}

type ListResponse[T any] struct {
	Items []T `json:"items"`
}

// EventDetail is the full explorer payload for a single Soroban event.
type EventDetail struct {
	ID             string          `json:"id"`
	Network        string          `json:"network"`
	ContractID     string          `json:"contract_id"`
	Ledger         uint32          `json:"ledger"`
	TxnHash        string          `json:"txn_hash"`
	EventType      string          `json:"event_type"`
	EventKind      string          `json:"event_kind"`
	EventTypeCode  uint32          `json:"event_type_code"`
	IngestedAt     time.Time       `json:"ingested_at"`
	SummaryPreview string          `json:"summary_preview"`
	DecodeStatus   string          `json:"decode_status"`
	Topics         json.RawMessage `json:"topics"`
	Value          json.RawMessage `json:"value"`
	TopicsXDR      []string        `json:"topics_xdr"`
	ValueXDR       string          `json:"value_xdr"`
}
