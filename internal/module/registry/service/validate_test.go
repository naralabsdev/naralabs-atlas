package service

import (
	"encoding/json"
	"testing"
)

func validTransferSchema() json.RawMessage {
	return json.RawMessage(`{
		"name": "transfer",
		"args": [
			{"name": "from", "type": "address"},
			{"name": "to", "type": "address"},
			{"name": "amount", "type": "i128"}
		]
	}`)
}

func TestValidateSchemaBody(t *testing.T) {
	if err := validateSchemaBody("transfer", validTransferSchema()); err != nil {
		t.Fatalf("expected valid schema: %v", err)
	}

	if err := validateSchemaBody("transfer", json.RawMessage(`{"name":"swap","args":[{"name":"x","type":"symbol"}]}`)); err == nil {
		t.Fatal("expected name mismatch error")
	}

	if err := validateSchemaBody("transfer", json.RawMessage(`{"name":"transfer"}`)); err == nil {
		t.Fatal("expected missing args error")
	}
}

func TestNormalizeContractID(t *testing.T) {
	if _, err := normalizeContractID(""); err == nil {
		t.Fatal("expected empty contract error")
	}
	if _, err := normalizeContractID("GABC"); err == nil {
		t.Fatal("expected invalid contract error")
	}
	if id, err := normalizeContractID(" CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if id[0] != 'C' {
		t.Fatal("expected trimmed contract id")
	}
}

func TestNormalizeNetwork(t *testing.T) {
	if _, err := normalizeNetwork("devnet"); err == nil {
		t.Fatal("expected invalid network")
	}
	if n, err := normalizeNetwork(" Testnet "); err != nil || n != "testnet" {
		t.Fatalf("got %q err=%v", n, err)
	}
}
