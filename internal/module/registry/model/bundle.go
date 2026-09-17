package model

import "time"

type SchemaContractActivity struct {
	EventCount       uint64    `json:"eventCount"`
	TransactionCount uint64    `json:"transactionCount"`
	DecodedCount     uint64    `json:"decodedCount"`
	Events24h        uint64    `json:"events24h"`
	FirstLedger      uint32    `json:"firstLedger"`
	LastLedger       uint32    `json:"lastLedger"`
	LastSeen         time.Time `json:"lastSeen"`
	SchemaStatus     string    `json:"schemaStatus"`
}

type SchemaContractListItem struct {
	ContractID  string     `json:"contractId"`
	Network     string     `json:"network"`
	SchemaCount int        `json:"schemaCount"`
	EventCount  int        `json:"eventCount"`
	IsVerified  bool       `json:"isVerified"`
	VerifiedAt  *time.Time `json:"verifiedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type SchemaContractListResponse struct {
	Items []SchemaContractListItem `json:"items"`
	Total int                      `json:"total"`
}

type SchemaRegistrySummary struct {
	Network             string `json:"network"`
	PublishedContracts  int    `json:"publishedContracts"`
	EventSchemas        int    `json:"eventSchemas"`
	VerifiedContracts   int    `json:"verifiedContracts"`
	CommunityContracts  int    `json:"communityContracts"`
}

type SchemaBundleSummary struct {
	Version    int        `json:"version"`
	TrustTier  string     `json:"trustTier"`
	EventCount int        `json:"eventCount"`
	Author     string     `json:"author,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

type SchemaContractProfile struct {
	ContractID string                 `json:"contractId"`
	Network    string                 `json:"network"`
	Indexed    bool                   `json:"indexed"`
	IsVerified bool                   `json:"isVerified"`
	VerifiedAt *time.Time             `json:"verifiedAt,omitempty"`
	Bundles    []SchemaBundleSummary  `json:"bundles"`
	Activity   *SchemaContractActivity `json:"activity"`
}

type SchemaBundleEvent struct {
	ID         string `json:"id"`
	EventName  string `json:"eventName"`
	Version    int    `json:"version"`
	TrustTier  string `json:"trustTier"`
	SchemaBody []byte `json:"schemaBody"`
	Author     string `json:"author,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type SchemaBundleDetail struct {
	ContractID string              `json:"contractId"`
	Network    string              `json:"network"`
	Version    int                 `json:"version"`
	TrustTier  string              `json:"trustTier"`
	Author     string              `json:"author,omitempty"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
	VerifiedAt *time.Time          `json:"verifiedAt,omitempty"`
	Events     []SchemaBundleEvent `json:"events"`
}
