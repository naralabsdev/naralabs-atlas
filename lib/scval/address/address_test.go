package address_test

import (
	"testing"

	"github.com/naralabs/naralabs-atlas/lib/scval/address"
)

func TestExtractAddressFromTopics(t *testing.T) {
	topics := `["transfer","GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"]`
	rows := address.Extract(topics, `{}`)
	if len(rows) == 0 {
		t.Fatal("expected at least one address")
	}
}

func TestExtractSkipsShortStrings(t *testing.T) {
	rows := address.Extract(`["transfer","abc"]`, `{}`)
	if len(rows) != 0 {
		t.Fatalf("rows=%v", rows)
	}
}

func TestExtractNestedValueAddress(t *testing.T) {
	value := `{"recipient":{"address":"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"}}`
	rows := address.Extract(`[]`, value)
	if len(rows) == 0 {
		t.Fatal("expected nested address")
	}
}
