// Package scval converts Soroban ScVal payloads (base64 XDR or RPC JSON) into
// stable tagged JSON for Atlas level-2 storage.
package scval

import (
	"encoding/json"
	"sync/atomic"
)

// DefaultParser is the process-wide ScVal parser used during ingest.
var DefaultParser = &Parser{}

// Parser turns base64-encoded ScVal XDR into tagged JSON documents.
type Parser struct{}

// ParseBase64 decodes one ScVal XDR blob. On failure it returns a lossless
// opaque wrapper so ingest never stalls on a single bad value.
func (p *Parser) ParseBase64(base64XDR string) json.RawMessage {
	if base64XDR == "" {
		return json.RawMessage("null")
	}

	out, err := p.parseBase64Strict(base64XDR)
	if err != nil {
		failures.Add(1)
		return opaquePayload("xdr_parse_failed", base64XDR, err)
	}
	return out
}

// FailureCount returns ScVal parse failures since process start.
func FailureCount() uint64 {
	return failures.Load()
}

var failures atomic.Uint64
