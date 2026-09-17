package service

import "errors"

var (
	ErrEmailExists        = errors.New("EMAIL_EXISTS")
	ErrEmailNotVerified   = errors.New("EMAIL_NOT_VERIFIED")
	ErrInvalidCredentials = errors.New("INVALID_CREDENTIALS")
	ErrInvalidToken       = errors.New("INVALID_TOKEN")
	ErrTokenExpired       = errors.New("TOKEN_EXPIRED")
	ErrResendCooldown     = errors.New("RESEND_COOLDOWN")
	ErrResetCooldown      = errors.New("RESET_COOLDOWN")
	ErrSamePassword       = errors.New("SAME_PASSWORD")
	ErrUnauthorized       = errors.New("UNAUTHORIZED")
)
