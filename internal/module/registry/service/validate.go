package service

import (
	"encoding/json"
	"strings"
)

var allowedNetworks = map[string]struct{}{
	"testnet":   {},
	"mainnet":   {},
	"futurenet": {},
}

func normalizeNetwork(network string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(network))
	if n == "" {
		return "", ErrInvalidNetwork
	}
	if _, ok := allowedNetworks[n]; !ok {
		return "", ErrInvalidNetwork
	}
	return n, nil
}

func normalizeContractID(contractID string) (string, error) {
	id := strings.TrimSpace(contractID)
	if id == "" || !strings.HasPrefix(id, "C") || len(id) < 10 {
		return "", ErrInvalidContractID
	}
	return id, nil
}

func normalizeEventName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", ErrInvalidEventName
	}
	return n, nil
}

// validateSchemaBody checks SEP-0048-aligned event definition shape:
// { "name": "...", "args": [ { "name": "...", "type": "..." }, ... ] }
func validateSchemaBody(eventName string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return ErrInvalidSchemaBody
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return ErrInvalidSchemaBody
	}

	nameRaw, ok := body["name"]
	if !ok {
		return ErrInvalidSchemaBody
	}
	var name string
	if err := json.Unmarshal(nameRaw, &name); err != nil || strings.TrimSpace(name) == "" {
		return ErrInvalidSchemaBody
	}
	if !strings.EqualFold(strings.TrimSpace(name), eventName) {
		return ErrInvalidSchemaBody
	}

	argsRaw, ok := body["args"]
	if !ok {
		return ErrInvalidSchemaBody
	}
	var args []map[string]json.RawMessage
	if err := json.Unmarshal(argsRaw, &args); err != nil || len(args) == 0 {
		return ErrInvalidSchemaBody
	}

	for _, arg := range args {
		var argName string
		if err := json.Unmarshal(arg["name"], &argName); err != nil || strings.TrimSpace(argName) == "" {
			return ErrInvalidSchemaBody
		}
		var argType string
		if err := json.Unmarshal(arg["type"], &argType); err != nil || strings.TrimSpace(argType) == "" {
			return ErrInvalidSchemaBody
		}
	}

	return nil
}
