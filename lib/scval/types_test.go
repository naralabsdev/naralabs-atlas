package scval

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
)

func TestParseBase64AdditionalTypes(t *testing.T) {
	voidVal := xdr.ScVal{Type: xdr.ScValTypeScvVoid}
	gotVoid := DefaultParser.ParseBase64(mustMarshalScVal(t, voidVal))
	assertTaggedField(t, gotVoid, "void", nil)

	i32Val := xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: ptrI32(-7)}
	gotI32 := DefaultParser.ParseBase64(mustMarshalScVal(t, i32Val))
	assertTaggedField(t, gotI32, "i32", float64(-7))

	strVal := xdr.ScVal{Type: xdr.ScValTypeScvString, Str: ptrStr("hello")}
	gotStr := DefaultParser.ParseBase64(mustMarshalScVal(t, strVal))
	assertTaggedField(t, gotStr, "string", "hello")

	u128Val := xdr.ScVal{
		Type: xdr.ScValTypeScvU128,
		U128: &xdr.UInt128Parts{Hi: 0, Lo: 42},
	}
	gotU128 := DefaultParser.ParseBase64(mustMarshalScVal(t, u128Val))
	var doc map[string]any
	if err := json.Unmarshal(gotU128, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["u128"] != "42" {
		t.Fatalf("u128=%v", doc["u128"])
	}

	tp := xdr.TimePoint(123456)
	gotTP := DefaultParser.ParseBase64(mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvTimepoint, Timepoint: &tp,
	}))
	assertTaggedField(t, gotTP, "timepoint", float64(123456))

	dur := xdr.Duration(99)
	gotDur := DefaultParser.ParseBase64(mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvDuration, Duration: &dur,
	}))
	assertTaggedField(t, gotDur, "duration", float64(99))

	rawBytes := xdr.ScBytes{0xab, 0xcd}
	gotBytes := DefaultParser.ParseBase64(mustMarshalScVal(t, xdr.ScVal{
		Type: xdr.ScValTypeScvBytes, Bytes: &rawBytes,
	}))
	assertTaggedField(t, gotBytes, "bytes", "abcd")
}

func TestParseBase64EmptyReturnsNull(t *testing.T) {
	if string(DefaultParser.ParseBase64("")) != "null" {
		t.Fatal("expected null for empty input")
	}
}

func TestValToTaggedNilMapAndVec(t *testing.T) {
	mapVal := xdr.ScVal{Type: xdr.ScValTypeScvMap, Map: nil}
	gotMap, err := DefaultParser.valToTagged(mapVal)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := gotMap.(map[string]any)
	if !ok || m["map"] == nil {
		t.Fatalf("got %v", gotMap)
	}

	vecVal := xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: nil}
	gotVec, err := DefaultParser.valToTagged(vecVal)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := gotVec.(map[string]any)
	if !ok {
		t.Fatalf("got %T", gotVec)
	}
	if _, ok := v["vec"]; !ok {
		t.Fatalf("got %v", gotVec)
	}
}

func TestValToTaggedNonEmptyVec(t *testing.T) {
	inner := xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: ptrU64(7)}
	items := xdr.ScVec{inner}
	vecPtr := &items
	vecVal := xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &vecPtr}

	got, err := DefaultParser.valToTagged(vecVal)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got %T", got)
	}
	itemsOut, ok := wrapped["vec"].([]any)
	if !ok || len(itemsOut) != 1 {
		t.Fatalf("vec=%v", wrapped["vec"])
	}
}

func ptrI32(v int32) *xdr.Int32 {
	i := xdr.Int32(v)
	return &i
}

func ptrStr(v string) *xdr.ScString {
	s := xdr.ScString(v)
	return &s
}
