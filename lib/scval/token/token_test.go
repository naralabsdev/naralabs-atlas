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
