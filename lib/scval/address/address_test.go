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
