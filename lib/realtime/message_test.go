package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEncodeDecodeIngestMessage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	msg := IngestMessage{
		Network:            "testnet",
		LastIngestedLedger: 12345,
		Events: []IngestEvent{
			{
				ID:         "evt-1",
				ContractID: "CABC",
				Ledger:     100,
				TxnHash:    "hash",
				TopicsJSON: `[{"symbol":"transfer"}]`,
				ValueJSON:  `{"i128":"100"}`,
				IngestedAt: now,
			},
		},
	}

	raw, err := EncodeIngestMessage(msg)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := DecodeIngestMessage(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Network != msg.Network {
		t.Fatalf("network mismatch: %s", got.Network)
	}
	if len(got.Events) != 1 || got.Events[0].ID != "evt-1" {
		t.Fatalf("events mismatch: %+v", got.Events)
	}
}

func TestHubBroadcastFiltersByNetwork(t *testing.T) {
	hub := NewHub(10)
	payload := map[string]string{"ok": "true"}
	if err := hub.Broadcast("testnet", MsgHomeStatsUpdated, payload); err != nil {
		t.Fatalf("broadcast: %v", err)
	}
}

func TestEncodeClientMessage(t *testing.T) {
	raw, err := EncodeClientMessage(ClientMessage{
		Type:    MsgPing,
		Network: "testnet",
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var decoded ClientMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Type != MsgPing {
		t.Fatalf("type mismatch: %s", decoded.Type)
	}
}
