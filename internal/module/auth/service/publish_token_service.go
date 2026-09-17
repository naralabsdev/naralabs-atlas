package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/naralabs/naralabs-atlas/internal/module/auth/model"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/repository"
)

const (
	PublishTokenPrefix  = "nl_live_"
	maxPublishTokens    = 5
	publishTokenEntropy = 24
)

var ErrPublishTokenNotFound = errors.New("PUBLISH_TOKEN_NOT_FOUND")
var ErrPublishTokenLimit = errors.New("PUBLISH_TOKEN_LIMIT")

type PublishTokenRepository interface {
	CountActiveByUser(ctx context.Context, userID string) (int, error)
	Insert(ctx context.Context, userID, tokenHash, tokenPrefix, label string) (model.PublishToken, error)
	ListByUser(ctx context.Context, userID string) ([]model.PublishToken, error)
	Revoke(ctx context.Context, userID, tokenID string, revokedAt time.Time) error
	ResolveActive(ctx context.Context, tokenHash string) (model.PublishToken, error)
	TouchLastUsed(ctx context.Context, tokenID string, usedAt time.Time) error
}

type PublishTokenService struct {
	repo PublishTokenRepository
	auth *AuthService
	now  func() time.Time
}

func NewPublishTokenService(repo PublishTokenRepository, auth *AuthService) *PublishTokenService {
	return &PublishTokenService{
		repo: repo,
		auth: auth,
		now:  time.Now,
	}
}

func (s *PublishTokenService) Create(
	ctx context.Context,
	authorization, label string,
) (model.PublishTokenCreateResult, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.PublishTokenCreateResult{}, ErrUnauthorized
	}

	label = repository.NormalizePublishTokenLabel(label)
	if label == "" {
		label = "CLI"
	}

	count, err := s.repo.CountActiveByUser(ctx, userID)
	if err != nil {
		return model.PublishTokenCreateResult{}, err
	}
	if count >= maxPublishTokens {
		return model.PublishTokenCreateResult{}, ErrPublishTokenLimit
	}

	raw, hash, prefix, err := s.newPublishToken()
	if err != nil {
		return model.PublishTokenCreateResult{}, err
	}

	token, err := s.repo.Insert(ctx, userID, hash, prefix, label)
	if err != nil {
		return model.PublishTokenCreateResult{}, err
	}

	return model.PublishTokenCreateResult{
		ID:        token.ID,
		Label:     token.Label,
		Token:     raw,
		Prefix:    token.TokenPrefix,
		CreatedAt: token.CreatedAt,
	}, nil
}

func (s *PublishTokenService) List(
	ctx context.Context,
	authorization string,
) (model.PublishTokenListResponse, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.PublishTokenListResponse{}, ErrUnauthorized
	}

	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return model.PublishTokenListResponse{}, err
	}

	out := model.PublishTokenListResponse{
		Total: len(items),
		Items: make([]model.PublishTokenPublic, 0, len(items)),
	}
	for _, item := range items {
		out.Items = append(out.Items, item.Public())
	}
	return out, nil
}

func (s *PublishTokenService) Revoke(
	ctx context.Context,
	authorization, tokenID string,
) error {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return ErrUnauthorized
	}

	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return ErrPublishTokenNotFound
	}

	if err := s.repo.Revoke(ctx, userID, tokenID, s.now()); err != nil {
		if errors.Is(err, repository.ErrPublishTokenNotFound) {
			return ErrPublishTokenNotFound
		}
		return err
	}
	return nil
}

func (s *PublishTokenService) ResolvePublishAuth(
	ctx context.Context,
	authorization string,
) (userID, tokenID string, err error) {
	raw := normalizeBearer(authorization)
	if raw == "" {
		return "", "", ErrUnauthorized
	}
	if !strings.HasPrefix(raw, PublishTokenPrefix) {
		return "", "", ErrUnauthorized
	}

	hash, err := hashPublishToken(raw)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	token, err := s.repo.ResolveActive(ctx, hash)
	if errors.Is(err, repository.ErrPublishTokenNotFound) {
		return "", "", ErrUnauthorized
	}
	if err != nil {
		return "", "", err
	}

	_ = s.repo.TouchLastUsed(ctx, token.ID, s.now())
	return token.UserID, token.ID, nil
}

func (s *PublishTokenService) ResolvePublishUserID(
	ctx context.Context,
	authorization string,
) (string, error) {
	userID, _, err := s.ResolvePublishAuth(ctx, authorization)
	return userID, err
}

func (s *PublishTokenService) ResolvePublishTokenID(
	ctx context.Context,
	authorization string,
) (string, error) {
	_, tokenID, err := s.ResolvePublishAuth(ctx, authorization)
	return tokenID, err
}

func (s *PublishTokenService) newPublishToken() (raw, hash, prefix string, err error) {
	buf := make([]byte, publishTokenEntropy)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", fmt.Errorf("generate publish token: %w", err)
	}

	raw = PublishTokenPrefix + hex.EncodeToString(buf)
	hash, err = hashPublishToken(raw)
	if err != nil {
		return "", "", "", err
	}
	prefix = raw[:len(PublishTokenPrefix)+8] + "…"
	return raw, hash, prefix, nil
}

func hashPublishToken(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty publish token")
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), nil
}

func normalizeBearer(raw string) string {
	raw = strings.TrimSpace(raw)
	return strings.TrimPrefix(raw, "Bearer ")
}
