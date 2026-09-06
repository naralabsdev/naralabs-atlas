// Package token extracts SEP-41 style transfer hints from level-2 JSON.
package token

import (
	"encoding/json"
	"strings"
)

type TransferHint struct {
	Action  string
	Symbol  string
	Amount  string
	From    string
	To      string
}

var transferTopicNames = map[string]struct{}{
	"transfer": {},
	"mint":     {},
	"burn":     {},
	"clawback": {},
}

// ExtractTransfer scans topics/value JSON for common token event shapes.
func ExtractTransfer(topicsJSON, valueJSON string) (TransferHint, bool) {
	topics := parseJSONArray(topicsJSON)
	if len(topics) == 0 {
		return TransferHint{}, false
	}

	action := symbolString(topics[0])
	if _, ok := transferTopicNames[strings.ToLower(action)]; !ok {
		return TransferHint{}, false
	}

	hint := TransferHint{Action: strings.ToLower(action)}
	if len(topics) > 1 {
		hint.From = addressString(topics[1])
	}
	if len(topics) > 2 {
		hint.To = addressString(topics[2])
	}

	value := parseObject(valueJSON)
	if sym, ok := value["symbol"].(string); ok {
		hint.Symbol = sym
	}
	if amt, ok := value["amount"].(string); ok {
		hint.Amount = amt
	} else if amt, ok := value["amount"].(json.Number); ok {
		hint.Amount = amt.String()
	} else if amt, ok := value["amount"].(float64); ok {
		hint.Amount = json.Number(strings.TrimSpace(formatFloat(amt))).String()
	}
	if hint.From == "" {
		if from, ok := value["from"].(string); ok {
			hint.From = from
		}
	}
	if hint.To == "" {
		if to, ok := value["to"].(string); ok {
			hint.To = to
		}
	}
	return hint, true
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

func symbolString(raw json.RawMessage) string {
	var sym string
	if err := json.Unmarshal(raw, &sym); err == nil {
		return sym
	}
	var tagged map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tagged); err == nil {
		if s, ok := tagged["symbol"]; ok {
			_ = json.Unmarshal(s, &sym)
			return sym
		}
		if s, ok := tagged["string"]; ok {
			_ = json.Unmarshal(s, &sym)
			return sym
		}
	}
	return ""
}

func addressString(raw json.RawMessage) string {
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

func formatFloat(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
