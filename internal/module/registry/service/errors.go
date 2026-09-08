package service

import "errors"

var (
	ErrInvalidContractID = errors.New("invalid contract id")
	ErrInvalidNetwork    = errors.New("invalid network")
	ErrInvalidEventName  = errors.New("invalid event name")
	ErrInvalidSchemaBody = errors.New("invalid schema body")
	ErrSchemaNotFound    = errors.New("schema not found")
)
