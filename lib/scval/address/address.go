// Package address extracts contract/account addresses from level-2 JSON.
package address

import (
	"encoding/json"
	"strconv"
	"strings"
)

type Row struct {
	Address string
	Role    string
}

// Extract scans topics and value JSON for Soroban address strings.
func Extract(topicsJSON, valueJSON string) []Row {
	seen := make(map[string]struct{})
	out := make([]Row, 0, 4)

	add := func(addr, role string) {
		addr = strings.TrimSpace(addr)
		if addr == "" || !looksLikeAddress(addr) {
			return
		}
		key := role + ":" + addr
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, Row{Address: addr, Role: role})
	}

	topics := parseJSONArray(topicsJSON)
	for i, topic := range topics {
		add(extractAddress(topic), topicRole(i))
	}
	walkValue(parseObject(valueJSON), "value", add)
	return out
}

func topicRole(index int) string {
	switch index {
	case 0:
		return "topic_event"
	case 1:
		return "topic_from"
	case 2:
		return "topic_to"
	default:
		return "topic_arg"
	}
}

func walkValue(v any, role string, add func(string, string)) {
	switch typed := v.(type) {
	case map[string]any:
		if addr, ok := typed["address"].(string); ok {
			add(addr, role)
		}
		for key, child := range typed {
			walkValue(child, role+"."+key, add)
		}
	case []any:
		for i, child := range typed {
			walkValue(child, role+"["+itoa(i)+"]", add)
		}
	case string:
		add(typed, role)
	}
}

func extractAddress(raw json.RawMessage) string {
	var addr string
	if err := json.Unmarshal(raw, &addr); err == nil {
		return addr
	}
	var tagged map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tagged); err == nil {
		if s, ok := tagged["address"]; ok {
			_ = json.Unmarshal(s, &addr)
			return addr
		}
	}
	return ""
}

func parseJSONArray(raw string) []json.RawMessage {
	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func parseObject(raw string) map[string]any {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return map[string]any{}
	}
	return obj
}

func looksLikeAddress(v string) bool {
	if len(v) < 32 {
		return false
	}
	if strings.HasPrefix(v, "G") || strings.HasPrefix(v, "C") || strings.HasPrefix(v, "M") {
		return true
	}
	return strings.Contains(v, "-")
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
