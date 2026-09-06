package scval

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
)

func TestParseBase64Symbol(t *testing.T) {
	t.Parallel()

	raw := mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  ptrSym("transfer"),
	})

	got := DefaultParser.ParseBase64(raw)
	assertTaggedField(t, got, "symbol", "transfer")
}

func TestParseBase64Bool(t *testing.T) {
	t.Parallel()

	raw := mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvBool,
		B:    ptrBool(true),
	})

	got := DefaultParser.ParseBase64(raw)
	assertTaggedField(t, got, "bool", true)
}

func TestNormalizeTopicsPrefersRPCJSON(t *testing.T) {
	t.Parallel()

	rpcTopics := []json.RawMessage{
		json.RawMessage(`{"symbol":"transfer"}`),
		json.RawMessage(`{"address":"CABC"}`),
	}

	got := DefaultParser.NormalizeTopics([]string{"ignored-xdr"}, rpcTopics)

	var decoded []json.RawMessage
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal topics json: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(decoded))
	}
	if string(decoded[0]) != `{"symbol":"transfer"}` {
		t.Fatalf("unexpected first topic: %s", decoded[0])
	}
}

func TestNormalizeValueUsesXDRWhenJSONMissing(t *testing.T) {
	t.Parallel()

	raw := mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvU64,
		U64:  ptrU64(42),
	})

	got := DefaultParser.NormalizeValue(raw, nil)
	assertTaggedField(t, got, "u64", float64(42))
}

func TestParseBase64InvalidReturnsOpaque(t *testing.T) {
	t.Parallel()

	got := DefaultParser.ParseBase64("not-valid-xdr!!!")
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("unmarshal opaque: %v", err)
	}
	opaque, ok := doc["opaque"].(map[string]any)
	if !ok {
		t.Fatalf("expected opaque wrapper, got %v", doc)
	}
	if opaque["reason"] != "xdr_parse_failed" {
		t.Fatalf("unexpected opaque reason: %v", opaque["reason"])
	}
}

func mustMarshalScVal(t *testing.T, val xdr.ScVal) string {
	t.Helper()
	out, err := xdr.MarshalBase64(val)
	if err != nil {
		t.Fatalf("marshal scval: %v", err)
	}
	return out
}

func assertTaggedField(t *testing.T, raw json.RawMessage, key string, want any) {
	t.Helper()

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal tagged json: %v", err)
	}
	got, ok := doc[key]
	if !ok {
		t.Fatalf("missing key %q in %s", key, raw)
	}
	if got != want {
		t.Fatalf("key %q: got %v want %v", key, got, want)
	}
}

func ptrSym(v string) *xdr.ScSymbol {
	s := xdr.ScSymbol(v)
	return &s
}

func ptrBool(v bool) *bool { return &v }

func ptrU64(v uint64) *xdr.Uint64 {
	u := xdr.Uint64(v)
	return &u
}
