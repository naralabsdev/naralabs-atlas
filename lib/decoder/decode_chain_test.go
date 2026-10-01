package decoder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDecodeApproveVecChainFixture(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata")
	schema, err := os.ReadFile(filepath.Join(root, "registry", "samples", "schemas", "approve__CDLZFC3S.json"))
	if err != nil {
		t.Fatal(err)
	}
	topicsRaw, _ := os.ReadFile(filepath.Join(root, "decoder", "fixtures", "approve.topics.json"))
	valueRaw, _ := os.ReadFile(filepath.Join(root, "decoder", "fixtures", "approve.value.json"))

	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}
	result := DecodeEvent(topics, valueRaw, schema, "")
	if result.DecodeStatus != "decoded" {
		t.Fatalf("status=%s", result.DecodeStatus)
	}
	if result.Fields["amount"] != "500000" {
		t.Fatalf("amount=%v", result.Fields["amount"])
	}
	if result.Fields["expiration_ledger"] != int64(123456) {
		t.Fatalf("expiration=%v", result.Fields["expiration_ledger"])
	}
}
