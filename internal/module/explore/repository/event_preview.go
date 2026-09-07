package repository

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

func eventTypeFromTopics(topicsJSON string) string {
	topicsJSON = strings.TrimSpace(topicsJSON)
	if topicsJSON == "" || topicsJSON == "null" {
		return "event"
	}

	var topics []json.RawMessage
	if err := json.Unmarshal([]byte(topicsJSON), &topics); err != nil || len(topics) == 0 {
		return "event"
	}

	if sym := taggedString(topics[0], "symbol"); sym != "" {
		return sym
	}
	var plain string
	if err := json.Unmarshal(topics[0], &plain); err == nil && plain != "" {
		return plain
	}
	return "event"
}

func summaryPreview(eventType, topicsJSON, valueJSON string) string {
	eventType = strings.TrimSpace(eventType)
	valueJSON = strings.TrimSpace(valueJSON)

	switch strings.ToLower(eventType) {
	case "fee":
		if s := formatFeeSummary(valueJSON); s != "" {
			return s
		}
	case "transfer", "mint", "burn", "approve", "clawback":
		if s := formatSimpleAmountSummary(eventType, valueJSON); s != "" {
			return s
		}
	}

	if s := humanizeValueJSON(valueJSON); s != "" {
		if eventType != "" && !strings.EqualFold(eventType, "event") {
			return formatEventLabel(eventType) + " · " + s
		}
		return s
	}

	if eventType != "" {
		return formatEventLabel(eventType)
	}
	return "Soroban event"
}

func formatFeeSummary(valueJSON string) string {
	amount := taggedScalarString(valueJSON, "i128")
	if amount == "" {
		amount = taggedScalarString(valueJSON, "i64")
	}
	if amount == "" {
		return ""
	}

	formatted := formatStroopsAmount(amount)
	if strings.HasPrefix(amount, "-") {
		return "Network fee " + formatted
	}
	return "Fee credit " + formatted
}

func formatSimpleAmountSummary(eventType, valueJSON string) string {
	amount := firstNumericTaggedValue(valueJSON)
	if amount == "" {
		return formatEventLabel(eventType)
	}
	return fmt.Sprintf("%s %s", formatEventLabel(eventType), formatStroopsAmount(amount))
}

func humanizeValueJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "{}" {
		return ""
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return truncatePreview(raw, 72)
	}
	if len(doc) != 1 {
		return truncatePreview(raw, 72)
	}

	for kind, val := range doc {
		return humanizeTaggedValue(kind, val)
	}
	return ""
}

func humanizeTaggedValue(kind string, val json.RawMessage) string {
	switch kind {
	case "bool":
		var b bool
		if err := json.Unmarshal(val, &b); err != nil {
			return ""
		}
		if b {
			return "Confirmed"
		}
		return "Not confirmed"
	case "string", "symbol":
		return decodeJSONString(val)
	case "address":
		addr := decodeJSONString(val)
		if addr == "" {
			return ""
		}
		return truncateMiddle(addr, 6, 4)
	case "i128", "i64", "i32", "u128", "u64", "u32":
		if s := decodeJSONString(val); s != "" {
			if kind == "i128" || kind == "i64" || kind == "i32" {
				return formatStroopsAmount(s)
			}
			return s
		}
	case "vec":
		var items []json.RawMessage
		if err := json.Unmarshal(val, &items); err != nil {
			return ""
		}
		return fmt.Sprintf("%d values", len(items))
	case "map":
		return summarizeMapPreview(val)
	}
	return truncatePreview(string(val), 72)
}

func summarizeMapPreview(raw json.RawMessage) string {
	var payload struct {
		Entries []struct {
			K json.RawMessage `json:"k"`
			V json.RawMessage `json:"v"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload.Entries) == 0 {
		return "Structured payload"
	}

	parts := make([]string, 0, 3)
	for _, entry := range payload.Entries {
		key := taggedString(entry.K, "symbol")
		if key == "" {
			key = decodeJSONString(entry.K)
		}
		val := decodeJSONString(entry.V)
		if val == "" {
			val = truncatePreview(string(entry.V), 24)
		}
		if key == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", formatEventLabel(key), val))
		if len(parts) == 2 {
			break
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d fields", len(payload.Entries))
	}
	if len(payload.Entries) > len(parts) {
		return strings.Join(parts, ", ") + fmt.Sprintf(" +%d", len(payload.Entries)-len(parts))
	}
	return strings.Join(parts, ", ")
}

func formatStroopsAmount(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	sign := ""
	if strings.HasPrefix(raw, "-") {
		sign = "−"
		raw = strings.TrimPrefix(raw, "-")
	}

	n, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return raw
	}

	stroopsPerXLM := big.NewInt(10_000_000)
	whole := new(big.Int).Div(n, stroopsPerXLM)
	frac := new(big.Int).Mod(n, stroopsPerXLM)

	if frac.Sign() == 0 {
		return sign + whole.String() + " XLM"
	}

	fracStr := frac.String()
	for len(fracStr) < 7 {
		fracStr = "0" + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")
	return sign + whole.String() + "." + fracStr + " XLM"
}

func firstNumericTaggedValue(raw string) string {
	for _, kind := range []string{"i128", "i64", "u128", "u64", "i32", "u32"} {
		if v := taggedScalarString(raw, kind); v != "" {
			return v
		}
	}
	return ""
}

func taggedScalarString(raw, kind string) string {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &doc); err != nil {
		return ""
	}
	val, ok := doc[kind]
	if !ok {
		return ""
	}
	if s := decodeJSONString(val); s != "" {
		return s
	}
	var n json.Number
	if err := json.Unmarshal(val, &n); err == nil {
		return n.String()
	}
	return ""
}

func decodeJSONString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

func formatEventLabel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Event"
	}
	raw = insertSpacesInCamel(strings.ReplaceAll(raw, "_", " "))
	parts := strings.Fields(strings.ToLower(raw))
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func insertSpacesInCamel(raw string) string {
	var out []rune
	for i, r := range raw {
		if i > 0 && unicode.IsUpper(r) {
			prev := rune(raw[i-1])
			if prev != ' ' && !unicode.IsUpper(prev) {
				out = append(out, ' ')
			}
		}
		out = append(out, r)
	}
	return string(out)
}

func truncatePreview(raw string, max int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= max {
		return raw
	}
	return raw[:max-3] + "..."
}

func truncateMiddle(raw string, head, tail int) string {
	if len(raw) <= head+tail+1 {
		return raw
	}
	return raw[:head] + "…" + raw[len(raw)-tail:]
}

func decodeStatus(semanticDecoded uint8) string {
	if semanticDecoded > 0 {
		return "decoded"
	}
	return "raw"
}

func taggedString(raw json.RawMessage, key string) string {
	var tagged map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tagged); err != nil {
		return ""
	}
	val, ok := tagged[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(val, &s); err != nil {
		return ""
	}
	return s
}
