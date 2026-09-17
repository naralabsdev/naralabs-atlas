package decoder

import (
	"encoding/json"
	"fmt"
	"strings"
)

type EventParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Doc      string `json:"doc,omitempty"`
}

type EventDefinition struct {
	Name         string       `json:"name"`
	PrefixTopics []string     `json:"prefix_topics"`
	DataFormat   string       `json:"data_format"`
	Params       []EventParam `json:"params"`
	Args         []EventParam `json:"args"`
	Doc          string       `json:"doc,omitempty"`
}

type Level2Payload struct {
	Topics []json.RawMessage `json:"topics"`
	Value  json.RawMessage   `json:"value"`
}

type Result struct {
	DecodeStatus  string         `json:"decodeStatus"`
	EventName     string         `json:"eventName,omitempty"`
	SchemaVersion int            `json:"schemaVersion,omitempty"`
	Fields        map[string]any `json:"fields,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	Level2        Level2Payload  `json:"level2"`
}

func DecodeEvent(topics []json.RawMessage, value json.RawMessage, schemaBody json.RawMessage, eventName string) Result {
	level2 := Level2Payload{
		Topics: append([]json.RawMessage(nil), topics...),
		Value:  append(json.RawMessage(nil), value...),
	}

	def, err := parseEventDefinition(schemaBody)
	if err != nil {
		return rawResult(level2)
	}

	if eventName != "" && !strings.EqualFold(eventName, def.Name) {
		return rawResult(level2)
	}

	if len(def.PrefixTopics) > 0 && !matchPrefixTopics(topics, def.PrefixTopics) {
		return rawResult(level2)
	}

	fields, err := extractFields(def, topics, value)
	if err != nil || len(fields) == 0 {
		return rawResult(level2)
	}

	return Result{
		DecodeStatus: "decoded",
		EventName:    def.Name,
		Fields:       fields,
		Summary:      buildSummary(def.Name, fields),
		Level2:       level2,
	}
}

func DecodeEventWithVersion(
	topics []json.RawMessage,
	value json.RawMessage,
	schemaBody json.RawMessage,
	eventName string,
	schemaVersion int,
) Result {
	result := DecodeEvent(topics, value, schemaBody, eventName)
	if result.DecodeStatus == "decoded" {
		result.SchemaVersion = schemaVersion
	}
	return result
}

func parseEventDefinition(raw json.RawMessage) (EventDefinition, error) {
	var def EventDefinition
	if err := json.Unmarshal(raw, &def); err != nil {
		return EventDefinition{}, err
	}
	if def.Name == "" {
		return EventDefinition{}, fmt.Errorf("missing event name")
	}
	if len(def.Params) == 0 && len(def.Args) > 0 {
		def.Params = def.Args
	}
	if def.DataFormat == "" {
		def.DataFormat = "single_value"
	}
	return def, nil
}

func matchPrefixTopics(topics []json.RawMessage, prefixes []string) bool {
	if len(prefixes) > len(topics) {
		return false
	}
	for i, prefix := range prefixes {
		symbol := topicSymbol(topics[i])
		if !strings.EqualFold(symbol, prefix) {
			return false
		}
	}
	return true
}

func extractFields(def EventDefinition, topics []json.RawMessage, value json.RawMessage) (map[string]any, error) {
	fields := make(map[string]any)
	topicIndex := len(def.PrefixTopics)

	for _, param := range def.Params {
		location := strings.ToLower(strings.TrimSpace(param.Location))
		switch location {
		case "data", "":
			val, err := extractDataParam(def.DataFormat, value, param)
			if err != nil {
				return nil, err
			}
			fields[param.Name] = val
		case "topic_list", "topic":
			if topicIndex >= len(topics) {
				return nil, fmt.Errorf("missing topic for %s", param.Name)
			}
			val, err := extractTypedValue(topics[topicIndex], param.Type)
			if err != nil {
				return nil, err
			}
			fields[param.Name] = val
			topicIndex++
		default:
			return nil, fmt.Errorf("unsupported param location %q", param.Location)
		}
	}

	return fields, nil
}

func extractDataParam(dataFormat string, value json.RawMessage, param EventParam) (any, error) {
	switch strings.ToLower(dataFormat) {
	case "single_value", "":
		return extractTypedValue(value, param.Type)
	case "map":
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(value, &doc); err != nil {
			return nil, err
		}
		if tagged, ok := doc["map"]; ok {
			return extractFromMapTagged(tagged, param)
		}
		raw, ok := doc[param.Name]
		if !ok {
			return nil, fmt.Errorf("missing map field %s", param.Name)
		}
		return extractTypedValue(raw, param.Type)
	case "vec":
		return extractFromVec(value, param)
	default:
		return extractTypedValue(value, param.Type)
	}
}

func extractFromMapTagged(raw json.RawMessage, param EventParam) (any, error) {
	var wrapper struct {
		Entries []struct {
			K json.RawMessage `json:"k"`
			V json.RawMessage `json:"v"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, err
	}
	for _, entry := range wrapper.Entries {
		key, err := extractTypedValue(entry.K, "symbol")
		if err != nil {
			continue
		}
		if fmt.Sprint(key) == param.Name {
			return extractTypedValue(entry.V, param.Type)
		}
	}
	return nil, fmt.Errorf("missing map entry %s", param.Name)
}

func extractFromVec(value json.RawMessage, param EventParam) (any, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(value, &doc); err != nil {
		return nil, err
	}
	raw, ok := doc["vec"]
	if !ok {
		return extractTypedValue(value, param.Type)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || len(items) == 0 {
		return nil, fmt.Errorf("empty vec value")
	}
	return extractTypedValue(items[0], param.Type)
}

func extractTypedValue(raw json.RawMessage, typ string) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("empty value")
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	want := strings.ToLower(strings.TrimSpace(typ))
	if val, ok := doc[want]; ok {
		return coerceValue(want, val)
	}

	// Allow symbol values where string is expected.
	if want == "string" {
		if val, ok := doc["symbol"]; ok {
			return coerceValue("string", val)
		}
	}

	// Single-key tagged documents.
	if len(doc) == 1 {
		for key, val := range doc {
			if key == want || (want == "string" && key == "symbol") {
				return coerceValue(want, val)
			}
		}
	}

	return nil, fmt.Errorf("type mismatch: expected %s in %s", want, string(raw))
}

func coerceValue(typ string, val any) (any, error) {
	switch typ {
	case "bool":
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid bool")
		}
		return b, nil
	case "u32", "i32", "u64", "i64", "timepoint", "duration":
		switch n := val.(type) {
		case float64:
			return int64(n), nil
		case json.Number:
			i, err := n.Int64()
			return i, err
		default:
			return nil, fmt.Errorf("invalid integer")
		}
	case "i128", "u128", "i256", "u256":
		switch v := val.(type) {
		case string:
			return v, nil
		case float64:
			return fmt.Sprintf("%.0f", v), nil
		default:
			return fmt.Sprint(v), nil
		}
	case "address", "string", "symbol", "bytes":
		return fmt.Sprint(val), nil
	default:
		return val, nil
	}
}

func topicSymbol(raw json.RawMessage) string {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	if sym, ok := doc["symbol"].(string); ok {
		return sym
	}
	if sym, ok := doc["sym"].(string); ok {
		return sym
	}
	for key, val := range doc {
		if key == "symbol" || key == "sym" {
			return fmt.Sprint(val)
		}
	}
	return ""
}

func buildSummary(eventName string, fields map[string]any) string {
	if len(fields) == 0 {
		return eventName
	}
	if count, ok := fields["count"]; ok {
		return fmt.Sprintf("%s: count=%v", eventName, count)
	}
	if amount, ok := fields["amount"]; ok {
		return fmt.Sprintf("%s: amount=%v", eventName, amount)
	}
	parts := make([]string, 0, len(fields))
	for name, val := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", name, val))
	}
	return fmt.Sprintf("%s (%s)", eventName, strings.Join(parts, ", "))
}

func rawResult(level2 Level2Payload) Result {
	return Result{
		DecodeStatus: "raw",
		Level2:       level2,
	}
}
