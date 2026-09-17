package decoder

import (
	"encoding/json"
	"testing"
)

func TestDecodeCounterEvent(t *testing.T) {
	schema := json.RawMessage(`{
		"name": "counter_incremented",
		"prefix_topics": ["cntr", "incr"],
		"data_format": "single_value",
		"params": [
			{"name": "count", "type": "u32", "location": "data"}
		]
	}`)

	topics := []json.RawMessage{
		json.RawMessage(`{"symbol":"cntr"}`),
		json.RawMessage(`{"symbol":"incr"}`),
	}
	value := json.RawMessage(`{"u32":42}`)

	result := DecodeEvent(topics, value, schema, "")
	if result.DecodeStatus != "decoded" {
		t.Fatalf("expected decoded, got %s", result.DecodeStatus)
	}
	if result.EventName != "counter_incremented" {
		t.Fatalf("unexpected event name: %s", result.EventName)
	}
	if result.Fields["count"] != int64(42) {
		t.Fatalf("unexpected count: %v", result.Fields["count"])
	}
}

func TestDecodeWithoutSchemaReturnsRaw(t *testing.T) {
	topics := []json.RawMessage{json.RawMessage(`{"symbol":"cntr"}`)}
	value := json.RawMessage(`{"u32":1}`)

	result := DecodeEvent(topics, value, json.RawMessage(`{
		"name": "x",
		"prefix_topics": ["other"],
		"params": [{"name": "a", "type": "u32", "location": "data"}]
	}`), "")
	if result.DecodeStatus != "raw" {
		t.Fatalf("expected raw fallback, got %s", result.DecodeStatus)
	}
}

func TestDecodePrefixMismatchReturnsRaw(t *testing.T) {
	schema := json.RawMessage(`{
		"name": "counter_incremented",
		"prefix_topics": ["cntr", "incr"],
		"data_format": "single_value",
		"params": [{"name": "count", "type": "u32", "location": "data"}]
	}`)

	topics := []json.RawMessage{
		json.RawMessage(`{"symbol":"other"}`),
		json.RawMessage(`{"symbol":"incr"}`),
	}
	value := json.RawMessage(`{"u32":42}`)

	result := DecodeEvent(topics, value, schema, "")
	if result.DecodeStatus != "raw" {
		t.Fatalf("expected raw, got %s", result.DecodeStatus)
	}
}
