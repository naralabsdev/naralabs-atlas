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
	APIKeyPrefix  = "nl_api_"
	maxAPIKeys    = 3
	apiKeyEntropy = 24
)

var ErrAPIKeyNotFound = errors.New("API_KEY_NOT_FOUND")
var ErrAPIKeyLimit = errors.New("API_KEY_LIMIT")

type APIKeyRepository interface {
	CountActiveByUser(ctx context.Context, userID string) (int, error)
	Insert(ctx context.Context, userID, tokenHash, tokenPrefix, label string) (model.APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]model.APIKey, error)
	Revoke(ctx context.Context, userID, keyID string, revokedAt time.Time) error
	ResolveActive(ctx context.Context, tokenHash string) (model.APIKey, error)
	TouchLastUsed(ctx context.Context, keyID string, usedAt time.Time) error
}

type APIKeyService struct {
	repo APIKeyRepository
	auth *AuthService
	now  func() time.Time
}

func NewAPIKeyService(repo APIKeyRepository, auth *AuthService) *APIKeyService {
	return &APIKeyService{
		repo: repo,
		auth: auth,
		now:  time.Now,
	}
}

func (s *APIKeyService) Create(
	ctx context.Context,
	authorization, label string,
) (model.APIKeyCreateResult, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.APIKeyCreateResult{}, ErrUnauthorized
	}

	label = repository.NormalizeAPIKeyLabel(label)
	if label == "" {
		label = "Default"
	}

	count, err := s.repo.CountActiveByUser(ctx, userID)
	if err != nil {
		return model.APIKeyCreateResult{}, err
	}
	if count >= maxAPIKeys {
		return model.APIKeyCreateResult{}, ErrAPIKeyLimit
	}

	raw, hash, prefix, err := s.newAPIKey()
	if err != nil {
		return model.APIKeyCreateResult{}, err
	}

	key, err := s.repo.Insert(ctx, userID, hash, prefix, label)
	if err != nil {
		return model.APIKeyCreateResult{}, err
	}

	return model.APIKeyCreateResult{
		ID:        key.ID,
		Label:     key.Label,
		Key:       raw,
		Prefix:    key.TokenPrefix,
		CreatedAt: key.CreatedAt,
	}, nil
}

func (s *APIKeyService) List(
	ctx context.Context,
	authorization string,
) (model.APIKeyListResponse, error) {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return model.APIKeyListResponse{}, ErrUnauthorized
	}

	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return model.APIKeyListResponse{}, err
	}

	out := model.APIKeyListResponse{
		Total: len(items),
		Items: make([]model.APIKeyPublic, 0, len(items)),
	}
	for _, item := range items {
		out.Items = append(out.Items, item.Public())
	}
	return out, nil
}

func (s *APIKeyService) Revoke(
	ctx context.Context,
	authorization, keyID string,
) error {
	userID, err := s.auth.ParseUserID(authorization)
	if err != nil {
		return ErrUnauthorized
	}

	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return ErrAPIKeyNotFound
	}

	if err := s.repo.Revoke(ctx, userID, keyID, s.now()); err != nil {
		if errors.Is(err, repository.ErrAPIKeyNotFound) {
			return ErrAPIKeyNotFound
		}
		return err
	}
	return nil
}

func (s *APIKeyService) ResolveAPIKeyAuth(
	ctx context.Context,
	authorization string,
) (userID, keyID string, err error) {
	raw := normalizeBearer(authorization)
	if raw == "" {
		return "", "", ErrUnauthorized
	}
	if !strings.HasPrefix(raw, APIKeyPrefix) {
		return "", "", ErrUnauthorized
	}

	hash, err := hashAPIKey(raw)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	key, err := s.repo.ResolveActive(ctx, hash)
	if errors.Is(err, repository.ErrAPIKeyNotFound) {
		return "", "", ErrUnauthorized
	}
	if err != nil {
		return "", "", err
	}

	_ = s.repo.TouchLastUsed(ctx, key.ID, s.now())
	return key.UserID, key.ID, nil
}

func (s *APIKeyService) ValidateOptionalAPIKey(
	ctx context.Context,
	authorization string,
) error {
	raw := normalizeBearer(authorization)
	if raw == "" {
		return nil
	}
	if !strings.HasPrefix(raw, APIKeyPrefix) {
		return ErrUnauthorized
	}
	_, _, err := s.ResolveAPIKeyAuth(ctx, authorization)
	return err
}

func (s *APIKeyService) RequireAPIKey(
	ctx context.Context,
	authorization string,
) error {
	raw := normalizeBearer(authorization)
	if raw == "" {
		return ErrUnauthorized
	}
	_, _, err := s.ResolveAPIKeyAuth(ctx, authorization)
	return err
}

func (s *APIKeyService) newAPIKey() (raw, hash, prefix string, err error) {
	buf := make([]byte, apiKeyEntropy)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", fmt.Errorf("generate api key: %w", err)
	}

	raw = APIKeyPrefix + hex.EncodeToString(buf)
	hash, err = hashAPIKey(raw)
	if err != nil {
		return "", "", "", err
	}
	prefix = raw[:len(APIKeyPrefix)+8] + "…"
	return raw, hash, prefix, nil
}

func hashAPIKey(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty api key")
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), nil
}
