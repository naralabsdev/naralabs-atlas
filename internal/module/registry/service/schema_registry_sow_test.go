package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/naralabs/naralabs-atlas/internal/module/registry/model"
)

type sowManifest struct {
	Network               string `json:"network"`
	SOWMinimumContracts   int    `json:"sow_minimum_contracts"`
	SOWMinimumEventTypes  int    `json:"sow_minimum_event_types"`
	Contracts             []struct {
		ContractID string `json:"contractId"`
		Label      string `json:"label"`
		Events     []struct {
			EventName  string `json:"eventName"`
			SchemaFile string `json:"schemaFile"`
		} `json:"events"`
	} `json:"contracts"`
}

func loadSOWManifest(t *testing.T) sowManifest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	path := filepath.Join(root, "testdata", "registry", "samples", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest sowManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return manifest
}

func loadSampleSchemaBody(t *testing.T, schemaFile string) json.RawMessage {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	path := filepath.Join(root, "testdata", "registry", "samples", "schemas", schemaFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", schemaFile, err)
	}
	return json.RawMessage(raw)
}

// TestDeliverable1_RegistrySampleCatalog verifies Instawards Deliverable 1 sample data
// and exercises publish, list, and retrieve flows against the registry service.
func TestDeliverable1_RegistrySampleCatalog(t *testing.T) {
	manifest := loadSOWManifest(t)

	eventCount := 0
	contractSet := map[string]struct{}{}
	for _, c := range manifest.Contracts {
		contractSet[c.ContractID] = struct{}{}
		eventCount += len(c.Events)
	}

	if len(contractSet) < manifest.SOWMinimumContracts {
		t.Fatalf("sample contracts=%d want >= %d", len(contractSet), manifest.SOWMinimumContracts)
	}
	if eventCount < manifest.SOWMinimumEventTypes {
		t.Fatalf("sample event types=%d want >= %d", eventCount, manifest.SOWMinimumEventTypes)
	}

	repo := &mockSchemaRepo{}
	svc := NewSchemaService(repo, manifest.Network)
	ctx := context.Background()

	for _, contract := range manifest.Contracts {
		for _, ev := range contract.Events {
			body := loadSampleSchemaBody(t, ev.SchemaFile)
			if err := validateSchemaBody(ev.EventName, body); err != nil {
				t.Fatalf("contract %s event %s invalid body: %v", contract.ContractID, ev.EventName, err)
			}

			pub, err := svc.Publish(ctx, model.PublishInput{
				PublisherUserID: "sow-demo-publisher",
				ContractID:      contract.ContractID,
				Network:         manifest.Network,
				EventName:       ev.EventName,
				SchemaBody:      body,
				Author:          "NaraLabs Instawards samples",
			})
			if err != nil {
				t.Fatalf("publish %s/%s: %v", contract.ContractID, ev.EventName, err)
			}
			if pub.Version < 1 {
				t.Fatalf("expected version >= 1 for %s, got %d", ev.EventName, pub.Version)
			}
		}
	}

	listResp, err := svc.List(ctx, manifest.Network, "", 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if listResp.Total != eventCount || len(listResp.Items) != eventCount {
		t.Fatalf("list total=%d len=%d want %d", listResp.Total, len(listResp.Items), eventCount)
	}

	for _, contract := range manifest.Contracts {
		byContract, err := svc.ListByContract(ctx, manifest.Network, contract.ContractID)
		if err != nil {
			t.Fatal(err)
		}
		if len(byContract) != len(contract.Events) {
			t.Fatalf("contract %s listed %d events want %d", contract.ContractID, len(byContract), len(contract.Events))
		}

		for _, ev := range contract.Events {
			got, err := svc.GetEventSchema(ctx, manifest.Network, contract.ContractID, ev.EventName, 0)
			if err != nil {
				t.Fatalf("get schema %s/%s: %v", contract.ContractID, ev.EventName, err)
			}
			if got.EventName != ev.EventName {
				t.Fatalf("unexpected event name %s", got.EventName)
			}

			versions, err := svc.ListEventVersions(ctx, manifest.Network, contract.ContractID, ev.EventName)
			if err != nil {
				t.Fatal(err)
			}
			if versions.Total != 1 {
				t.Fatalf("expected 1 version for %s, got %d", ev.EventName, versions.Total)
			}
		}
	}
}
