package service_test

import (
	"testing"
	"time"

	"github.com/naralabs/naralabs-atlas/internal/module/ingest/service"
)

func TestResolveColdStartRetention(t *testing.T) {
	start, err := service.ResolveColdStart(0, 20000, 17280, "")
	if err != nil {
		t.Fatal(err)
	}
	if start != 2720 {
		t.Fatalf("expected 2720, got %d", start)
	}
}

func TestResolveColdStartExistingCursor(t *testing.T) {
	start, err := service.ResolveColdStart(5000, 20000, 17280, "")
	if err != nil {
		t.Fatal(err)
	}
	if start != 5000 {
		t.Fatalf("expected existing cursor 5000, got %d", start)
	}
}

func TestNextAdaptivePoll(t *testing.T) {
	min := time.Second
	max := 5 * time.Second
	current := 2 * time.Second

	busy := service.NextAdaptivePoll(current, min, max, false)
	if busy >= current {
		t.Fatalf("expected faster poll when busy, got %s", busy)
	}

	idle := service.NextAdaptivePoll(current, min, max, true)
	if idle <= current {
		t.Fatalf("expected slower poll when caught up, got %s", idle)
	}
}

func TestReorgFromLedger(t *testing.T) {
	from := service.ReorgFromLedger(100, 12)
	if from != 88 {
		t.Fatalf("expected 88, got %d", from)
	}
}
