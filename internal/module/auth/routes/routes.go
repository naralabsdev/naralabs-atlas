package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/naralabs/naralabs-atlas/internal/module/auth/handler"
)

const authBase = "/v1/auth"

// RegisterAuthRoutes registers authentication endpoints on the Huma API.
func RegisterAuthRoutes(api huma.API, h *handler.AuthHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-register",
		Method:      http.MethodPost,
		Path:        authBase + "/register",
		Summary:     "Register account",
		Description: "Create a new account and send an email verification link.",
		Tags:        []string{"auth"},
	}, h.HandleRegister)

	huma.Register(api, huma.Operation{
		OperationID: "auth-login",
		Method:      http.MethodPost,
		Path:        authBase + "/login",
		Summary:     "Login",
		Description: "Authenticate with email and password. Requires a verified email.",
		Tags:        []string{"auth"},
	}, h.HandleLogin)

	huma.Register(api, huma.Operation{
		OperationID: "auth-verify-email",
		Method:      http.MethodPost,
		Path:        authBase + "/verify-email",
		Summary:     "Verify email",
		Description: "Consume an email verification token and return a JWT session.",
		Tags:        []string{"auth"},
	}, h.HandleVerifyEmail)

	huma.Register(api, huma.Operation{
		OperationID: "auth-resend-verification",
		Method:      http.MethodPost,
		Path:        authBase + "/resend-verification",
		Summary:     "Resend verification email",
		Description: "Send a new verification link when the account is not yet verified.",
		Tags:        []string{"auth"},
	}, h.HandleResendVerification)

	huma.Register(api, huma.Operation{
		OperationID: "auth-me",
		Method:      http.MethodGet,
		Path:        authBase + "/me",
		Summary:     "Current user",
		Description: "Return the authenticated user profile.",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, h.HandleMe)
}
