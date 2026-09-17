package service

import "errors"

var (
	ErrInvalidContractID          = errors.New("invalid contract id")
	ErrInvalidNetwork             = errors.New("invalid network")
	ErrInvalidEventName           = errors.New("invalid event name")
	ErrInvalidSchemaBody          = errors.New("invalid schema body")
	ErrSchemaNotFound             = errors.New("schema not found")
	ErrUnauthorized               = errors.New("unauthorized")
	ErrForbidden                  = errors.New("forbidden")
	ErrInvalidSignature           = errors.New("invalid signature")
	ErrChallengeNotFound          = errors.New("challenge not found")
	ErrChallengeExpired           = errors.New("challenge expired")
	ErrChallengeAlreadyUsed       = errors.New("challenge already used")
	ErrContractAuthorityDenied    = errors.New("contract authority denied")
	ErrContractAuthorityUnavailable = errors.New("contract authority unavailable")
	ErrNothingToVerify            = errors.New("nothing to verify")
)
