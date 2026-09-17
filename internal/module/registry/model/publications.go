package model

import "time"

type SchemaVersionSummary struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	TrustTier string    `json:"trustTier"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type VerifiedSummary struct {
	ID             string    `json:"id"`
	Version        int       `json:"version"`
	VerifiedWallet string    `json:"verifiedWallet"`
	VerifiedAt     time.Time `json:"verifiedAt"`
}

type EventPublication struct {
	EventName       string                `json:"eventName"`
	LatestCommunity *SchemaVersionSummary `json:"latestCommunity,omitempty"`
	Verified        *VerifiedSummary      `json:"verified,omitempty"`
	Versions        []SchemaVersionSummary `json:"versions"`
}

type ContractPublication struct {
	ContractID string             `json:"contractId"`
	Network    string             `json:"network"`
	Events     []EventPublication `json:"events"`
	CanVerify  bool               `json:"canVerify"`
}

type ProjectPublicationsResponse struct {
	Contracts []ContractPublication `json:"contracts"`
}

type VerifyChallengeResponse struct {
	Nonce     string    `json:"nonce"`
	Message   string    `json:"message"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type VerifiedEventResult struct {
	EventName string `json:"eventName"`
	SchemaID  string `json:"schemaId"`
	Version   int    `json:"version"`
}

type VerifyContractResponse struct {
	ContractID string                `json:"contractId"`
	Network    string                `json:"network"`
	Wallet     string                `json:"wallet"`
	VerifiedAt time.Time             `json:"verifiedAt"`
	Events     []VerifiedEventResult `json:"events"`
}

type VerifyChallengeInput struct {
	ContractID string
	Network    string
}

type VerifyContractInput struct {
	ContractID string
	Network    string
	Wallet     string
	Signature  string
	Nonce      string
}
