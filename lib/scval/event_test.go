package scval

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
)

func TestNormalizeValuePrefersJSON(t *testing.T) {
	raw := json.RawMessage(`{"amount":"10"}`)
	got := DefaultParser.NormalizeValue("ignored-xdr", raw)
	if string(got) != string(raw) {
		t.Fatalf("got %s", got)
	}
}

func TestNormalizeValueEmptyXDR(t *testing.T) {
	got := DefaultParser.NormalizeValue("", nil)
	if string(got) != "null" {
		t.Fatalf("got %s", got)
	}
}

func TestNormalizeTopicsFromXDR(t *testing.T) {
	topic := mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  ptrSym("mint"),
	})
	got := DefaultParser.NormalizeTopics([]string{topic}, nil)
	var items []json.RawMessage
	if err := json.Unmarshal(got, &items); err != nil || len(items) != 1 {
		t.Fatalf("topics=%s err=%v", got, err)
	}
}

func TestFailureCountIncrementsOnBadXDR(t *testing.T) {
	before := FailureCount()
	DefaultParser.ParseBase64("!!!bad-xdr!!!")
	if FailureCount() <= before {
		t.Fatal("expected failure counter increment")
	}
}

func TestParseIntegerWidths(t *testing.T) {
	cases := []struct {
		key string
		val xdr.ScVal
	}{
		{"u32", xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: ptrU32(9)}},
		{"i64", xdr.ScVal{Type: xdr.ScValTypeScvI64, I64: ptrI64(-11)}},
		{
			key: "i128",
			val: xdr.ScVal{
				Type: xdr.ScValTypeScvI128,
				I128: &xdr.Int128Parts{Hi: 0, Lo: 128},
			},
		},
		{
			key: "u256",
			val: xdr.ScVal{
				Type: xdr.ScValTypeScvU256,
				U256: &xdr.UInt256Parts{LoLo: 256},
			},
		},
		{
			key: "i256",
			val: xdr.ScVal{
				Type: xdr.ScValTypeScvI256,
				I256: &xdr.Int256Parts{LoLo: 256},
			},
		},
	}

	for _, tc := range cases {
		got := DefaultParser.ParseBase64(mustMarshalScVal(t, tc.val))
		var doc map[string]any
		if err := json.Unmarshal(got, &doc); err != nil {
			t.Fatalf("%s: %v", tc.key, err)
		}
		if doc[tc.key] == nil {
			t.Fatalf("%s missing in %s", tc.key, got)
		}
	}
}

func TestParseScError(t *testing.T) {
	code := xdr.ScErrorCodeScecInvalidInput
	contractCode := xdr.Uint32(7)
	scErr := xdr.ScError{
		Type:         xdr.ScErrorTypeSceContract,
		Code:         &code,
		ContractCode: &contractCode,
	}
	got := DefaultParser.ParseBase64(mustMarshalScVal(t, xdr.ScVal{
		Type:  xdr.ScValTypeScvError,
		Error: &scErr,
	}))
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}
	errDoc, ok := doc["error"].(map[string]any)
	if !ok || errDoc["contract_code"] != float64(7) {
		t.Fatalf("doc=%v", doc)
	}
}

func ptrU32(v uint32) *xdr.Uint32 {
	u := xdr.Uint32(v)
	return &u
}

func ptrI64(v int64) *xdr.Int64 {
	i := xdr.Int64(v)
	return &i
}
