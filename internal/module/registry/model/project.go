package model

import "time"

type SchemaProject struct {
	ID             string
	UserID         string
	Name           string
	Slug           string
	Description    string
	Status         string
	PublishTokenID string
	PublishToken   string
	ContractID     *string
	Network        *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SchemaProjectPublic struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SchemaProjectDetail struct {
	SchemaProjectPublic
	PublishToken string `json:"publishToken"`
}

type SchemaProjectListResponse struct {
	Items []SchemaProjectPublic `json:"items"`
	Total int                   `json:"total"`
}

type CreateSchemaProjectInput struct {
	Name string
}

type UpdateSchemaProjectInput struct {
	Name        string
	Description string
}

func (p SchemaProject) Public() SchemaProjectPublic {
	return SchemaProjectPublic{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func (p SchemaProject) Detail() SchemaProjectDetail {
	return SchemaProjectDetail{
		SchemaProjectPublic: p.Public(),
		PublishToken:        p.PublishToken,
	}
}
