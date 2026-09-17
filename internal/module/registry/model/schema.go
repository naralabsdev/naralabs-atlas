package model

import (
	"encoding/json"
	"time"
)

type EventSchema struct {
	ID              string
	ContractID      string
	Network         string
	EventName       string
	Version         int
	SchemaBody      json.RawMessage
	Author          *string
	PublisherUserID *string
	TrustTier       string
	Status          string
	VerifiedWallet  *string
	VerifiedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type EventSchemaPublic struct {
	ID             string          `json:"id"`
	ContractID     string          `json:"contractId"`
	Network        string          `json:"network"`
	EventName      string          `json:"eventName"`
	Version        int             `json:"version"`
	SchemaBody     json.RawMessage `json:"schemaBody"`
	Author         string          `json:"author,omitempty"`
	TrustTier      string          `json:"trustTier"`
	Status         string          `json:"status"`
	VerifiedWallet string          `json:"verifiedWallet,omitempty"`
	VerifiedAt     *time.Time      `json:"verifiedAt,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type EventSchemaVersionPublic struct {
	ID             string     `json:"id"`
	Version        int        `json:"version"`
	TrustTier      string     `json:"trustTier"`
	Status         string     `json:"status"`
	Author         string     `json:"author,omitempty"`
	VerifiedWallet string     `json:"verifiedWallet,omitempty"`
	VerifiedAt     *time.Time `json:"verifiedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type EventVersionsResponse struct {
	ContractID string                     `json:"contractId"`
	Network    string                     `json:"network"`
	EventName  string                     `json:"eventName"`
	Items      []EventSchemaVersionPublic `json:"items"`
	Total      int                        `json:"total"`
}

func (s EventSchema) Public() EventSchemaPublic {
	out := EventSchemaPublic{
		ID:         s.ID,
		ContractID: s.ContractID,
		Network:    s.Network,
		EventName:  s.EventName,
		Version:    s.Version,
		SchemaBody: s.SchemaBody,
		TrustTier:  s.TrustTier,
		Status:     s.Status,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
	if s.Author != nil {
		out.Author = *s.Author
	}
	if s.VerifiedWallet != nil {
		out.VerifiedWallet = *s.VerifiedWallet
	}
	if s.VerifiedAt != nil {
		out.VerifiedAt = s.VerifiedAt
	}
	if out.TrustTier == "" {
		out.TrustTier = "community"
	}
	if out.Status == "" {
		out.Status = "published"
	}
	return out
}

func (s EventSchema) VersionPublic() EventSchemaVersionPublic {
	out := EventSchemaVersionPublic{
		ID:        s.ID,
		Version:   s.Version,
		TrustTier: s.TrustTier,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.Author != nil {
		out.Author = *s.Author
	}
	if s.VerifiedWallet != nil {
		out.VerifiedWallet = *s.VerifiedWallet
	}
	if s.VerifiedAt != nil {
		out.VerifiedAt = s.VerifiedAt
	}
	if out.TrustTier == "" {
		out.TrustTier = "community"
	}
	if out.Status == "" {
		out.Status = "published"
	}
	return out
}

type PublishInput struct {
	PublisherUserID string
	ContractID      string
	Network         string
	EventName       string
	SchemaBody      json.RawMessage
	Author          string
}

type ListResponse struct {
	Items []EventSchemaPublic `json:"items"`
	Total int                 `json:"total"`
}
