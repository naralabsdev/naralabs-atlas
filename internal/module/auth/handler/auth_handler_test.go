package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/naralabs/naralabs-atlas/internal/module/auth/service"
)

func TestMapAuthErrorStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"email exists", service.ErrEmailExists, http.StatusConflict},
		{"not verified", service.ErrEmailNotVerified, http.StatusForbidden},
		{"invalid credentials", service.ErrInvalidCredentials, http.StatusUnauthorized},
		{"invalid token", service.ErrInvalidToken, http.StatusBadRequest},
		{"expired token", service.ErrTokenExpired, http.StatusBadRequest},
		{"resend cooldown", service.ErrResendCooldown, http.StatusTooManyRequests},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := mapAuthError(tc.err)
			var statusErr huma.StatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("expected StatusError, got %T", err)
			}
			if statusErr.GetStatus() != tc.wantStatus {
				t.Fatalf("status=%d want %d", statusErr.GetStatus(), tc.wantStatus)
			}
		})
	}
}
