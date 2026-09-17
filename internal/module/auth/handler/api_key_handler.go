package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/auth/model"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/service"
)

type APIKeyHandler struct {
	svc *service.APIKeyService
}

func NewAPIKeyHandler(svc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

func (h *APIKeyHandler) HandleCreate(ctx context.Context, input *CreateAPIKeyInput) (*CreateAPIKeyOutput, error) {
	result, err := h.svc.Create(ctx, input.Authorization, input.Body.Label)
	if err != nil {
		return nil, mapAPIKeyError(err)
	}

	out := &CreateAPIKeyOutput{}
	out.Body = result
	return out, nil
}

func (h *APIKeyHandler) HandleList(ctx context.Context, input *ListAPIKeysInput) (*ListAPIKeysOutput, error) {
	result, err := h.svc.List(ctx, input.Authorization)
	if err != nil {
		return nil, mapAPIKeyError(err)
	}

	out := &ListAPIKeysOutput{}
	out.Body = result
	return out, nil
}

func (h *APIKeyHandler) HandleRevoke(ctx context.Context, input *RevokeAPIKeyInput) (*RevokeAPIKeyOutput, error) {
	if err := h.svc.Revoke(ctx, input.Authorization, input.ID); err != nil {
		return nil, mapAPIKeyError(err)
	}

	out := &RevokeAPIKeyOutput{}
	out.Body.Revoked = true
	return out, nil
}

func mapAPIKeyError(err error) error {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		return authError(http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, service.ErrAPIKeyNotFound):
		return authError(http.StatusNotFound, "API_KEY_NOT_FOUND", "API key not found")
	case errors.Is(err, service.ErrAPIKeyLimit):
		return authError(http.StatusConflict, "API_KEY_LIMIT", "Maximum number of API keys reached")
	default:
		if msg := err.Error(); strings.Contains(msg, "invalid") {
			return huma.Error400BadRequest(msg)
		}
		return huma.Error500InternalServerError("api key request failed", err)
	}
}

type CreateAPIKeyInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	Body          struct {
		Label string `json:"label,omitempty" doc:"Optional label for this API key" example:"Production app"`
	}
}

type CreateAPIKeyOutput struct {
	Body model.APIKeyCreateResult
}

type ListAPIKeysInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
}

type ListAPIKeysOutput struct {
	Body model.APIKeyListResponse
}

type RevokeAPIKeyInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"API key ID"`
}

type RevokeAPIKeyOutput struct {
	Body struct {
		Revoked bool `json:"revoked"`
	}
}
