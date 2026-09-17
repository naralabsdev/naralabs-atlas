package model

import "time"

type PublishToken struct {
	ID          string
	UserID      string
	TokenPrefix string
	Label       string
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

type PublishTokenPublic struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Prefix     string     `json:"prefix"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type PublishTokenCreateResult struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Token     string    `json:"token"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
}

type PublishTokenListResponse struct {
	Items []PublishTokenPublic `json:"items"`
	Total int                  `json:"total"`
}

func (t PublishToken) Public() PublishTokenPublic {
	return PublishTokenPublic{
		ID:         t.ID,
		Label:      t.Label,
		Prefix:     t.TokenPrefix,
		LastUsedAt: t.LastUsedAt,
		CreatedAt:  t.CreatedAt,
	}
}
