package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	registermodel "github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	registryservice "github.com/naralabs/naralabs-atlas/internal/module/registry/service"
)

const testContractID = "CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4"

type stubSchemaLookup struct {
	byEvent    map[string]registermodel.EventSchema
	byContract []registermodel.EventSchema
}

func (s *stubSchemaLookup) ListByContract(_ context.Context, _, contractID string) ([]registermodel.EventSchema, error) {
	if contractID != testContractID {
		return nil, nil
	}
	return s.byContract, nil
}

func (s *stubSchemaLookup) GetEventSchema(_ context.Context, _, contractID, eventName string, version int) (registermodel.EventSchema, error) {
	if contractID != testContractID {
		return registermodel.EventSchema{}, registryservice.ErrSchemaNotFound
	}
	key := eventName
	if version > 0 {
		key = eventName
	}
	if schema, ok := s.byEvent[key]; ok {
		if version > 0 && schema.Version != version {
			return registermodel.EventSchema{}, registryservice.ErrSchemaNotFound
		}
		return schema, nil
	}
	return registermodel.EventSchema{}, registryservice.ErrSchemaNotFound
}

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	path := filepath.Join(root, "testdata", "decoder", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return raw
}

func counterSchemaModel(t *testing.T) registermodel.EventSchema {
	t.Helper()
	return registermodel.EventSchema{
		ContractID: testContractID,
		Network:    "testnet",
		EventName:  "counter_incremented",
		Version:    1,
		SchemaBody: json.RawMessage(loadTestdata(t, "counter_incremented.schema.json")),
		Status:     "published",
	}
}

func counterDecodeInput(t *testing.T) DecodeEventInput {
	t.Helper()
	topicsRaw := loadTestdata(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}
	return DecodeEventInput{
		Network:    "testnet",
		ContractID: testContractID,
		EventName:  "counter_incremented",
		TopicsJSON: topics,
		ValueJSON:  json.RawMessage(loadTestdata(t, "counter_incremented.value.json")),
	}
}

func TestDecodeServiceWithRegisteredSchema(t *testing.T) {
	schema := counterSchemaModel(t)
	svc := NewDecodeService(&stubSchemaLookup{
		byEvent: map[string]registermodel.EventSchema{
			"counter_incremented": schema,
		},
	})

	topicsRaw := loadTestdata(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}

	result, err := svc.Decode(context.Background(), DecodeEventInput{
		Network:    "testnet",
		ContractID: testContractID,
		EventName:  "counter_incremented",
		TopicsJSON: topics,
		ValueJSON:  json.RawMessage(loadTestdata(t, "counter_incremented.value.json")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DecodeStatus != "decoded" {
		t.Fatalf("status=%s", result.DecodeStatus)
	}
	if result.SchemaVersion != 1 {
		t.Fatalf("schema version=%d", result.SchemaVersion)
	}
	if result.Fields["count"] != int64(42) {
		t.Fatalf("fields=%v", result.Fields)
	}
}

func TestDecodeServiceAutoMatchesSchemaByPrefix(t *testing.T) {
	schema := counterSchemaModel(t)
	svc := NewDecodeService(&stubSchemaLookup{
		byContract: []registermodel.EventSchema{schema},
	})

	topicsRaw := loadTestdata(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}

	result, err := svc.Decode(context.Background(), DecodeEventInput{
		Network:    "testnet",
		ContractID: testContractID,
		TopicsJSON: topics,
		ValueJSON:  json.RawMessage(loadTestdata(t, "counter_incremented.value.json")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DecodeStatus != "decoded" {
		t.Fatalf("status=%s", result.DecodeStatus)
	}
	if result.EventName != "counter_incremented" {
		t.Fatalf("event=%s", result.EventName)
	}
}

func TestDecodeServiceWithoutRegistryReturnsRaw(t *testing.T) {
	svc := NewDecodeService(&stubSchemaLookup{})

	topicsRaw := loadTestdata(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}

	result, err := svc.Decode(context.Background(), DecodeEventInput{
		Network:    "testnet",
		ContractID: testContractID,
		TopicsJSON: topics,
		ValueJSON:  json.RawMessage(loadTestdata(t, "counter_incremented.value.json")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DecodeStatus != "raw" {
		t.Fatalf("status=%s", result.DecodeStatus)
	}
	if len(result.Level2.Topics) != 2 {
		t.Fatalf("level2 topics=%d", len(result.Level2.Topics))
	}
}

func TestDecodeServiceBatch(t *testing.T) {
	schema := counterSchemaModel(t)
	svc := NewDecodeService(&stubSchemaLookup{
		byEvent: map[string]registermodel.EventSchema{
			"counter_incremented": schema,
		},
	})

	topicsRaw := loadTestdata(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}

	item := DecodeEventInput{
		Network:    "testnet",
		ContractID: testContractID,
		EventName:  "counter_incremented",
		TopicsJSON: topics,
		ValueJSON:  json.RawMessage(loadTestdata(t, "counter_incremented.value.json")),
	}

	batch, err := svc.DecodeBatch(context.Background(), []DecodeEventInput{item, item})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Total != 2 || len(batch.Items) != 2 {
		t.Fatalf("batch total=%d len=%d", batch.Total, len(batch.Items))
	}
	for i, row := range batch.Items {
		if row.DecodeStatus != "decoded" {
			t.Fatalf("item %d status=%s", i, row.DecodeStatus)
		}
	}
}

func TestDecodeServiceInvalidInput(t *testing.T) {
	svc := NewDecodeService(&stubSchemaLookup{})
	_, err := svc.Decode(context.Background(), DecodeEventInput{Network: "testnet"})
	if err != ErrInvalidDecodeInput {
		t.Fatalf("err=%v", err)
	}
}

func TestDecodeServiceBatchTooLarge(t *testing.T) {
	svc := NewDecodeService(&stubSchemaLookup{})
	items := make([]DecodeEventInput, maxBatchSize+1)
	for i := range items {
		items[i] = counterDecodeInput(t)
	}
	_, err := svc.DecodeBatch(context.Background(), items)
	if err != ErrBatchTooLarge {
		t.Fatalf("err=%v", err)
	}
}
