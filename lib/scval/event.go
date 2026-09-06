package scval

import "encoding/json"

// NormalizeTopics builds a JSON array for event topics.
// RPC-supplied topic JSON is preferred; missing entries are parsed from XDR.
func (p *Parser) NormalizeTopics(topicsXDR []string, topicsJSON []json.RawMessage) json.RawMessage {
	if len(topicsJSON) > 0 {
		out, err := json.Marshal(topicsJSON)
		if err == nil {
			return out
		}
	}

	items := make([]json.RawMessage, 0, len(topicsXDR))
	for _, topicXDR := range topicsXDR {
		items = append(items, p.ParseBase64(topicXDR))
	}

	out, err := json.Marshal(items)
	if err != nil {
		return json.RawMessage("[]")
	}
	return out
}

// NormalizeValue builds a JSON value for the event body.
// RPC-supplied value JSON is preferred; otherwise value XDR is parsed locally.
func (p *Parser) NormalizeValue(valueXDR string, valueJSON json.RawMessage) json.RawMessage {
	if len(valueJSON) > 0 && string(valueJSON) != "null" {
		return valueJSON
	}
	if valueXDR == "" {
		return json.RawMessage("null")
	}
	return p.ParseBase64(valueXDR)
}
