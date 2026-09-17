package service

import (
	"context"
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/model"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/repository"
)

type mockAPIKeyRepo struct {
	items map[string]model.APIKey
	hashes map[string]model.APIKey
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{
		items:  map[string]model.APIKey{},
		hashes: map[string]model.APIKey{},
	}
}

func (m *mockAPIKeyRepo) CountActiveByUser(_ context.Context, userID string) (int, error) {
	count := 0
	for _, item := range m.items {
		if item.UserID == userID && item.RevokedAt == nil {
			count++
		}
	}
	return count, nil
}

func (m *mockAPIKeyRepo) Insert(_ context.Context, userID, tokenHash, tokenPrefix, label string) (model.APIKey, error) {
	key := model.APIKey{
		ID:          "key-" + tokenPrefix,
		UserID:      userID,
		TokenPrefix: tokenPrefix,
		Label:       label,
		CreatedAt:   time.Now(),
	}
	m.items[key.ID] = key
	m.hashes[tokenHash] = key
	return key, nil
}

func (m *mockAPIKeyRepo) ListByUser(_ context.Context, userID string) ([]model.APIKey, error) {
	var out []model.APIKey
	for _, item := range m.items {
		if item.UserID == userID && item.RevokedAt == nil {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *mockAPIKeyRepo) Revoke(_ context.Context, userID, keyID string, revokedAt time.Time) error {
	item, ok := m.items[keyID]
	if !ok || item.UserID != userID || item.RevokedAt != nil {
		return repository.ErrAPIKeyNotFound
	}
	item.RevokedAt = &revokedAt
	m.items[keyID] = item
	return nil
}

func (m *mockAPIKeyRepo) ResolveActive(_ context.Context, tokenHash string) (model.APIKey, error) {
	item, ok := m.hashes[tokenHash]
	if !ok || item.RevokedAt != nil {
		return model.APIKey{}, repository.ErrAPIKeyNotFound
	}
	return item, nil
}

func (m *mockAPIKeyRepo) TouchLastUsed(_ context.Context, keyID string, usedAt time.Time) error {
	item, ok := m.items[keyID]
	if !ok {
		return nil
	}
	item.LastUsedAt = &usedAt
	m.items[keyID] = item
	return nil
}

func testAuthService(t *testing.T) *AuthService {
	t.Helper()
	return &AuthService{
		cfg: &config.Config{
			Auth: config.AuthConfig{
				JWTSecret: "test-secret",
				JWTExpiry: time.Hour,
			},
		},
		now: time.Now,
	}
}

func TestCreateAPIKeyReturnsRawKeyOnce(t *testing.T) {
	auth := testAuthService(t)
	session, err := auth.issueSession(model.User{ID: "user-1", Email: "you@example.com"})
	if err != nil {
		t.Fatal(err)
	}

	svc := NewAPIKeyService(newMockAPIKeyRepo(), auth)
	result, err := svc.Create(context.Background(), "Bearer "+session.Token, "Production")
	if err != nil {
		t.Fatal(err)
	}
	if result.Key == "" || result.Prefix == "" {
		t.Fatalf("expected key and prefix, got %+v", result)
	}
	if result.Key[:len(APIKeyPrefix)] != APIKeyPrefix {
		t.Fatalf("expected nl_api_ prefix, got %q", result.Key)
	}
}

func TestRequireAPIKeyRejectsMissingHeader(t *testing.T) {
	svc := NewAPIKeyService(newMockAPIKeyRepo(), testAuthService(t))
	if err := svc.RequireAPIKey(context.Background(), ""); err == nil {
		t.Fatal("expected unauthorized")
	}
}
