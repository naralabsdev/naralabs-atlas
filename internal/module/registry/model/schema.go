package model

import (
	"encoding/json"
	"time"
)

type EventSchema struct {
	ID         string
	ContractID string
	Network    string
	EventName  string
	Version    int
	SchemaBody json.RawMessage
	Author     *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EventSchemaPublic struct {
	ID         string          `json:"id"`
	ContractID string          `json:"contractId"`
	Network    string          `json:"network"`
	EventName  string          `json:"eventName"`
	Version    int             `json:"version"`
	SchemaBody json.RawMessage `json:"schemaBody"`
	Author     string          `json:"author,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

func (s EventSchema) Public() EventSchemaPublic {
	out := EventSchemaPublic{
		ID:         s.ID,
		ContractID: s.ContractID,
		Network:    s.Network,
		EventName:  s.EventName,
		Version:    s.Version,
		SchemaBody: s.SchemaBody,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
	if s.Author != nil {
		out.Author = *s.Author
	}
	return out
}

type PublishInput struct {
	ContractID string
	Network    string
	EventName  string
	SchemaBody json.RawMessage
	Author     string
}

type ListResponse struct {
	Items []EventSchemaPublic `json:"items"`
	Total int                 `json:"total"`
}
