package service

import (
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/model"
)

func TestSanitizeCallbackURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/", "/"},
		{"/contracts", "/contracts"},
		{"https://evil.test", ""},
		{"//evil.test", ""},
		{"", ""},
	}

	for _, tc := range tests {
		if got := sanitizeCallbackURL(tc.in, "http://localhost:3000"); got != tc.want {
			t.Fatalf("sanitizeCallbackURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := validatePassword("short"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := validatePassword("longenough"); err == nil {
		t.Fatal("expected missing complexity error")
	}
	if err := validatePassword("Longenough1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueAndParseToken(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret",
			JWTExpiry: time.Hour,
		},
	}
	svc := &AuthService{
		cfg: cfg,
		now: time.Now,
	}

	session, err := svc.issueSession(model.User{ID: "user-1", Email: "you@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if session.Token == "" {
		t.Fatal("expected token")
	}

	userID, err := svc.parseToken("Bearer " + session.Token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "user-1" {
		t.Fatalf("userID=%q", userID)
	}
}

func TestHashToken(t *testing.T) {
	hash, err := hashToken("abc")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == "abc" {
		t.Fatalf("unexpected hash %q", hash)
	}
}
