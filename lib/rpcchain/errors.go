// Package rpcchain provides resilient Stellar RPC call chains for Atlas.
package rpcchain

import "errors"

var (
	// ErrAllEndpointsDown is returned when every RPC endpoint is unavailable.
	ErrAllEndpointsDown = errors.New("rpcchain: all endpoints down")

	// ErrReanchorCursor signals that pagination must restart from ledger position.
	ErrReanchorCursor = errors.New("rpcchain: endpoint switched, discard RPC cursor")
)

func IsReanchorCursor(err error) bool {
	return errors.Is(err, ErrReanchorCursor)
}
