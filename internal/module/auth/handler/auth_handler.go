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

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) HandleRegister(ctx context.Context, input *RegisterInput) (*RegisterOutput, error) {
	result, err := h.svc.Register(ctx, input.Body.Email, input.Body.Password, input.Body.CallbackURL)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &RegisterOutput{}
	out.Body = result
	return out, nil
}

func (h *AuthHandler) HandleLogin(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
	session, err := h.svc.Login(ctx, input.Body.Email, input.Body.Password)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &LoginOutput{}
	out.Body = session
	return out, nil
}

func (h *AuthHandler) HandleVerifyEmail(ctx context.Context, input *VerifyEmailInput) (*VerifyEmailOutput, error) {
	session, err := h.svc.VerifyEmail(ctx, input.Body.Token)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &VerifyEmailOutput{}
	out.Body = session
	return out, nil
}

func (h *AuthHandler) HandleResendVerification(ctx context.Context, input *ResendVerificationInput) (*ResendVerificationOutput, error) {
	alreadyVerified, err := h.svc.ResendVerification(ctx, input.Body.Email, input.Body.CallbackURL)
	if errors.Is(err, service.ErrResendCooldown) {
		return nil, mapAuthError(err)
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to resend verification email", err)
	}

	out := &ResendVerificationOutput{}
	out.Body.Sent = !alreadyVerified
	out.Body.AlreadyVerified = alreadyVerified
	return out, nil
}

func (h *AuthHandler) HandleMe(ctx context.Context, input *MeInput) (*MeOutput, error) {
	token := strings.TrimSpace(input.Authorization)
	if token == "" {
		return nil, mapAuthError(service.ErrUnauthorized)
	}

	user, err := h.svc.Me(ctx, token)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &MeOutput{}
	out.Body = user
	return out, nil
}

func (h *AuthHandler) HandleChangePassword(ctx context.Context, input *ChangePasswordInput) (*ChangePasswordOutput, error) {
	token := strings.TrimSpace(input.Authorization)
	if token == "" {
		return nil, mapAuthError(service.ErrUnauthorized)
	}

	err := h.svc.ChangePassword(ctx, token, input.Body.CurrentPassword, input.Body.NewPassword)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &ChangePasswordOutput{}
	out.Body.Changed = true
	return out, nil
}

func (h *AuthHandler) HandleForgotPassword(ctx context.Context, input *ForgotPasswordInput) (*ForgotPasswordOutput, error) {
	err := h.svc.RequestPasswordReset(ctx, input.Body.Email)
	if errors.Is(err, service.ErrResetCooldown) {
		return nil, mapAuthError(err)
	}
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &ForgotPasswordOutput{}
	out.Body.Sent = true
	return out, nil
}

func (h *AuthHandler) HandleResetPassword(ctx context.Context, input *ResetPasswordInput) (*ResetPasswordOutput, error) {
	session, err := h.svc.ResetPassword(ctx, input.Body.Token, input.Body.Password)
	if err != nil {
		return nil, mapAuthError(err)
	}

	out := &ResetPasswordOutput{}
	out.Body = session
	return out, nil
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, service.ErrEmailExists):
		return authError(http.StatusConflict, "EMAIL_EXISTS", "An account with this email already exists")
	case errors.Is(err, service.ErrEmailNotVerified):
		return authError(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Please verify your email before signing in")
	case errors.Is(err, service.ErrInvalidCredentials):
		return authError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	case errors.Is(err, service.ErrInvalidToken):
		return authError(http.StatusBadRequest, "INVALID_TOKEN", "Invalid verification link")
	case errors.Is(err, service.ErrTokenExpired):
		return authError(http.StatusBadRequest, "TOKEN_EXPIRED", "Verification link has expired")
	case errors.Is(err, service.ErrResendCooldown):
		return authError(http.StatusTooManyRequests, "RESEND_COOLDOWN", "Please wait before requesting another verification email")
	case errors.Is(err, service.ErrResetCooldown):
		return authError(http.StatusTooManyRequests, "RESET_COOLDOWN", "Please wait before requesting another password reset email")
	case errors.Is(err, service.ErrSamePassword):
		return authError(http.StatusBadRequest, "SAME_PASSWORD", "New password must be different from your current password")
	case errors.Is(err, service.ErrUnauthorized):
		return authError(http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	default:
		if msg := err.Error(); strings.Contains(msg, "invalid email") || strings.Contains(msg, "password must") || strings.Contains(msg, "email is required") {
			return huma.Error400BadRequest(msg)
		}
		return huma.Error500InternalServerError("auth request failed", err)
	}
}

type authErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e authErrorBody) Error() string {
	return e.Message
}

func (e authErrorBody) ErrorDetail() *huma.ErrorDetail {
	return &huma.ErrorDetail{
		Message:  e.Message,
		Location: "code",
		Value:    e.Code,
	}
}

func authError(status int, code, message string) error {
	return huma.NewError(status, message, authErrorBody{Code: code, Message: message})
}

type RegisterInput struct {
	Body struct {
		Email        string `json:"email" doc:"Account email" example:"you@example.com"`
		Password     string `json:"password" doc:"Account password" minLength:"8"`
		CallbackURL  string `json:"callbackUrl,omitempty" doc:"Relative redirect path after verification" example:"/"`
	}
}

type RegisterOutput struct {
	Body model.RegisterResult
}

type LoginInput struct {
	Body struct {
		Email    string `json:"email" doc:"Account email" example:"you@example.com"`
		Password string `json:"password" doc:"Account password" minLength:"8"`
	}
}

type LoginOutput struct {
	Body model.AuthSession
}

type VerifyEmailInput struct {
	Body struct {
		Token string `json:"token" doc:"Email verification token from the link"`
	}
}

type VerifyEmailOutput struct {
	Body model.AuthSession
}

type ResendVerificationInput struct {
	Body struct {
		Email       string `json:"email" doc:"Account email" example:"you@example.com"`
		CallbackURL string `json:"callbackUrl,omitempty" doc:"Relative redirect path after verification" example:"/"`
	}
}

type ResendVerificationOutput struct {
	Body struct {
		Sent            bool `json:"sent"`
		AlreadyVerified bool `json:"alreadyVerified"`
	}
}

type MeInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
}

type MeOutput struct {
	Body model.UserPublic
}

type ChangePasswordInput struct {
	Authorization string `header:"Authorization" doc:"Bearer JWT access token"`
	Body          struct {
		CurrentPassword string `json:"currentPassword" doc:"Current account password" minLength:"8"`
		NewPassword     string `json:"newPassword" doc:"New account password" minLength:"8"`
	}
}

type ChangePasswordOutput struct {
	Body struct {
		Changed bool `json:"changed"`
	}
}

type ForgotPasswordInput struct {
	Body struct {
		Email string `json:"email" doc:"Account email" example:"you@example.com"`
	}
}

type ForgotPasswordOutput struct {
	Body struct {
		Sent bool `json:"sent"`
	}
}

type ResetPasswordInput struct {
	Body struct {
		Token    string `json:"token" doc:"Password reset token from the email link"`
		Password string `json:"password" doc:"New account password" minLength:"8"`
	}
}

type ResetPasswordOutput struct {
	Body model.AuthSession
}
