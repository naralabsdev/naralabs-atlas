package token_test

import (
	"testing"

	"github.com/naralabs/naralabs-atlas/lib/scval/token"
)

func TestExtractTransfer(t *testing.T) {
	topics := `["transfer","GFROM...","GTO..."]`
	value := `{"amount":"100","symbol":"USDC"}`
	hint, ok := token.ExtractTransfer(topics, value)
	if !ok {
		t.Fatal("expected transfer hint")
	}
	if hint.Action != "transfer" {
		t.Fatalf("action=%s", hint.Action)
	}
	if hint.Amount != "100" {
		t.Fatalf("amount=%s", hint.Amount)
	}
}

func TestExtractTransferMint(t *testing.T) {
	hint, ok := token.ExtractTransfer(`["mint"]`, `{"amount":"5"}`)
	if !ok || hint.Action != "mint" || hint.Amount != "5" {
		t.Fatalf("hint=%+v ok=%v", hint, ok)
	}
}

func TestExtractTransferNoMatch(t *testing.T) {
	_, ok := token.ExtractTransfer(`["approve"]`, `{}`)
	if ok {
		t.Fatal("expected no hint")
	}
}

func TestExtractTransferFromValueFields(t *testing.T) {
	hint, ok := token.ExtractTransfer(`["transfer"]`, `{"from":"GFROM","to":"GTO","amount":"99"}`)
	if !ok {
		t.Fatal()
	}
	if hint.From != "GFROM" || hint.To != "GTO" {
		t.Fatalf("hint=%+v", hint)
	}
}

func TestExtractTransferTaggedTopicSymbol(t *testing.T) {
	topics := `[{"symbol":"burn"}]`
	hint, ok := token.ExtractTransfer(topics, `{"amount":42.5}`)
	if !ok || hint.Action != "burn" {
		t.Fatalf("hint=%+v", hint)
	}
}

func TestExtractTransferAmountNumber(t *testing.T) {
	_, ok := token.ExtractTransfer(`["transfer"]`, `{"amount":100}`)
	if !ok {
		t.Fatal("expected numeric amount")
	}
}
