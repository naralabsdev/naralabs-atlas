package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	decoderservice "github.com/naralabs/naralabs-atlas/internal/module/decoder/service"
	registermodel "github.com/naralabs/naralabs-atlas/internal/module/registry/model"
	registryservice "github.com/naralabs/naralabs-atlas/internal/module/registry/service"
)

const handlerTestContractID = "CBGROPHYFCL5FNTNJF6HXVJEWPLOMHZJ3MNISPTB2JHXLGKQMO2MCWK4"

type stubAPIKeys struct {
	err error
}

func (s stubAPIKeys) RequireAPIKey(context.Context, string) error {
	return s.err
}

type handlerSchemaStub struct {
	schema registermodel.EventSchema
}

func (h handlerSchemaStub) ListByContract(context.Context, string, string) ([]registermodel.EventSchema, error) {
	return nil, nil
}

func (h handlerSchemaStub) GetEventSchema(context.Context, string, string, string, int) (registermodel.EventSchema, error) {
	if h.schema.EventName == "" {
		return registermodel.EventSchema{}, registryservice.ErrSchemaNotFound
	}
	return h.schema, nil
}

func loadHandlerFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	path := filepath.Join(root, "testdata", "decoder", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return raw
}

func TestHandleDecodeRequiresAPIKey(t *testing.T) {
	h := NewDecodeHandler(decoderservice.NewDecodeService(nil), stubAPIKeys{err: errors.New("missing key")}, "")
	_, err := h.HandleDecode(context.Background(), &DecodeInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	var statusErr huma.StatusError
	if !errors.As(err, &statusErr) || statusErr.GetStatus() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %v", err)
	}
}

func TestHandleDecodeSuccess(t *testing.T) {
	topicsRaw := loadHandlerFixture(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}

	schema := registermodel.EventSchema{
		ContractID: handlerTestContractID,
		Network:    "testnet",
		EventName:  "counter_incremented",
		Version:    1,
		SchemaBody: json.RawMessage(loadHandlerFixture(t, "counter_incremented.schema.json")),
		Status:     "published",
	}

	svc := decoderservice.NewDecodeService(handlerSchemaStub{schema: schema})
	h := NewDecodeHandler(svc, stubAPIKeys{}, "")

	out, err := h.HandleDecode(context.Background(), &DecodeInput{
		Authorization: "Bearer nl_api_test",
		Body: DecodeEventBody{
			Network:    "testnet",
			ContractID: handlerTestContractID,
			EventName:  "counter_incremented",
			TopicsJSON: topics,
			ValueJSON:  json.RawMessage(loadHandlerFixture(t, "counter_incremented.value.json")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Body.DecodeStatus != "decoded" {
		t.Fatalf("status=%s", out.Body.DecodeStatus)
	}
	if out.Body.Fields["count"] != int64(42) {
		t.Fatalf("fields=%v", out.Body.Fields)
	}
}

func TestHandlePlaygroundDecodeSuccess(t *testing.T) {
	const token = "bff-secret-for-test"
	topicsRaw := loadHandlerFixture(t, "counter_incremented.topics.json")
	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatal(err)
	}

	schema := registermodel.EventSchema{
		ContractID: handlerTestContractID,
		Network:    "testnet",
		EventName:  "counter_incremented",
		Version:    1,
		SchemaBody: json.RawMessage(loadHandlerFixture(t, "counter_incremented.schema.json")),
		Status:     "published",
	}

	svc := decoderservice.NewDecodeService(handlerSchemaStub{schema: schema})
	h := NewDecodeHandler(svc, nil, token)

	out, err := h.HandlePlaygroundDecode(context.Background(), &PlaygroundDecodeInput{
		Authorization: "Bearer " + token,
		Body: DecodeEventBody{
			Network:    "testnet",
			ContractID: handlerTestContractID,
			EventName:  "counter_incremented",
			TopicsJSON: topics,
			ValueJSON:  json.RawMessage(loadHandlerFixture(t, "counter_incremented.value.json")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Body.DecodeStatus != "decoded" {
		t.Fatalf("status=%s", out.Body.DecodeStatus)
	}
}
