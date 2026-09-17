package repository

import (
	"testing"
	"time"
)

func TestMapStoredEvent(t *testing.T) {
	item := MapStoredEvent(
		"evt-1",
		"CABC123",
		"txnhash",
		42,
		`[{"symbol":"transfer"}]`,
		`{"i128":"100"}`,
		false,
		time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
	)

	if item.EventType != "transfer" {
		t.Fatalf("event type: %s", item.EventType)
	}
	if item.DecodeStatus != "raw" {
		t.Fatalf("decode status: %s", item.DecodeStatus)
	}
	if item.SummaryPreview == "" {
		t.Fatal("expected summary preview")
	}
}
