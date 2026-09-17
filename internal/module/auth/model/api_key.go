package model

import "time"

type APIKey struct {
	ID          string
	UserID      string
	TokenPrefix string
	Label       string
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

type APIKeyPublic struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Prefix     string     `json:"prefix"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type APIKeyCreateResult struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Key       string    `json:"key"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
}

type APIKeyListResponse struct {
	Items []APIKeyPublic `json:"items"`
	Total int            `json:"total"`
}

func (k APIKey) Public() APIKeyPublic {
	return APIKeyPublic{
		ID:         k.ID,
		Label:      k.Label,
		Prefix:     k.TokenPrefix,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
}
