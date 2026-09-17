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

type PublishTokenHandler struct {
	svc *service.PublishTokenService
}

func NewPublishTokenHandler(svc *service.PublishTokenService) *PublishTokenHandler {
	return &PublishTokenHandler{svc: svc}
}

func (h *PublishTokenHandler) HandleCreate(ctx context.Context, input *CreatePublishTokenInput) (*CreatePublishTokenOutput, error) {
	result, err := h.svc.Create(ctx, input.Authorization, input.Body.Label)
	if err != nil {
		return nil, mapPublishTokenError(err)
	}

	out := &CreatePublishTokenOutput{}
	out.Body = result
	return out, nil
}

func (h *PublishTokenHandler) HandleList(ctx context.Context, input *ListPublishTokensInput) (*ListPublishTokensOutput, error) {
	result, err := h.svc.List(ctx, input.Authorization)
	if err != nil {
		return nil, mapPublishTokenError(err)
	}

	out := &ListPublishTokensOutput{}
	out.Body = result
	return out, nil
}

func (h *PublishTokenHandler) HandleRevoke(ctx context.Context, input *RevokePublishTokenInput) (*RevokePublishTokenOutput, error) {
	if err := h.svc.Revoke(ctx, input.Authorization, input.ID); err != nil {
		return nil, mapPublishTokenError(err)
	}

	out := &RevokePublishTokenOutput{}
	out.Body.Revoked = true
	return out, nil
}

func mapPublishTokenError(err error) error {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		return authError(http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, service.ErrPublishTokenNotFound):
		return authError(http.StatusNotFound, "PUBLISH_TOKEN_NOT_FOUND", "Publish token not found")
	case errors.Is(err, service.ErrPublishTokenLimit):
		return authError(http.StatusConflict, "PUBLISH_TOKEN_LIMIT", "Maximum number of publish tokens reached")
	default:
		if msg := err.Error(); strings.Contains(msg, "invalid") {
			return huma.Error400BadRequest(msg)
		}
		return huma.Error500InternalServerError("publish token request failed", err)
	}
}

type CreatePublishTokenInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	Body          struct {
		Label string `json:"label,omitempty" doc:"Optional label for this token" example:"MacBook CLI"`
	}
}

type CreatePublishTokenOutput struct {
	Body model.PublishTokenCreateResult
}

type ListPublishTokensInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
}

type ListPublishTokensOutput struct {
	Body model.PublishTokenListResponse
}

type RevokePublishTokenInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	ID            string `path:"id" doc:"Publish token ID"`
}

type RevokePublishTokenOutput struct {
	Body struct {
		Revoked bool `json:"revoked"`
	}
}
