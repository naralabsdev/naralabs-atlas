package decoder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Fixture paths mirror SEP-0048 counter sample used in docs and decode API examples.
func loadDecoderFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "decoder", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func TestDecodeCounterIncrementedFixture(t *testing.T) {
	schema := loadDecoderFixture(t, "counter_incremented.schema.json")
	topicsRaw := loadDecoderFixture(t, "counter_incremented.topics.json")
	value := loadDecoderFixture(t, "counter_incremented.value.json")

	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatalf("parse topics fixture: %v", err)
	}

	result := DecodeEvent(topics, value, schema, "")
	if result.DecodeStatus != "decoded" {
		t.Fatalf("expected decoded, got %s", result.DecodeStatus)
	}
	if result.EventName != "counter_incremented" {
		t.Fatalf("event name: %s", result.EventName)
	}
	if result.Fields["count"] != int64(42) {
		t.Fatalf("count field: %v", result.Fields["count"])
	}
	if result.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
}
