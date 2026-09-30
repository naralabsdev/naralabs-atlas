package decoder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

type sowManifest struct {
	Fixtures []sowFixture `json:"fixtures"`
}

type sowFixture struct {
	ID          string `json:"id"`
	EventName   string `json:"eventName"`
	Expect      string `json:"expect"`
	SchemaFile  string `json:"schemaFile"`
	TopicsFile  string `json:"topicsFile"`
	ValueFile   string `json:"valueFile"`
}

func decoderTestdataRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "decoder")
}

func loadSOWFixtures(t *testing.T) []sowFixture {
	t.Helper()
	root := decoderTestdataRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest sowManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(manifest.Fixtures) == 0 {
		t.Fatal("empty fixture manifest")
	}
	return manifest.Fixtures
}

func resolveDecoderFile(root, rel string) string {
	if rel == "" {
		return ""
	}
	if rel == "counter_incremented.schema.json" ||
		rel == "counter_incremented.topics.json" ||
		rel == "counter_incremented.value.json" {
		return filepath.Join(root, rel)
	}
	base := filepath.Base(rel)
	if len(rel) > 8 && rel[:8] == "schemas/" {
		return filepath.Join(root, "..", "registry", "samples", "schemas", base)
	}
	return filepath.Join(root, rel)
}

func runSOWFixture(t *testing.T, root string, fx sowFixture) Result {
	t.Helper()
	schema := readFile(t, resolveDecoderFile(root, fx.SchemaFile))
	topicsRaw := readFile(t, resolveDecoderFile(root, fx.TopicsFile))
	value := readFile(t, resolveDecoderFile(root, fx.ValueFile))

	var topics []json.RawMessage
	if err := json.Unmarshal(topicsRaw, &topics); err != nil {
		t.Fatalf("%s topics: %v", fx.ID, err)
	}
	return DecodeEvent(topics, value, schema, "")
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

// TestDeliverable2_DecoderSampleFixtures verifies Instawards SOW Deliverable 2 targets on prepared fixtures:
// >=10 distinct decoded event types, >=90% expected outcomes, p95 single decode under 500ms (local).
func TestDeliverable2_DecoderSampleFixtures(t *testing.T) {
	root := decoderTestdataRoot(t)
	fixtures := loadSOWFixtures(t)

	decodedEvents := map[string]struct{}{}
	var matched, total int

	for _, fx := range fixtures {
		total++
		result := runSOWFixture(t, root, fx)
		ok := false
		switch fx.Expect {
		case "decoded":
			ok = result.DecodeStatus == "decoded" && result.EventName == fx.EventName
			if ok {
				decodedEvents[fx.EventName] = struct{}{}
			}
		case "raw":
			ok = result.DecodeStatus == "raw"
		default:
			t.Fatalf("unknown expect %q for %s", fx.Expect, fx.ID)
		}
		if !ok {
			t.Errorf("fixture %s: want %s, got status=%s event=%s", fx.ID, fx.Expect, result.DecodeStatus, result.EventName)
		} else {
			matched++
		}
	}

	if len(decodedEvents) < 10 {
		t.Fatalf("SOW target: need >=10 decoded event types, got %d: %v", len(decodedEvents), eventNames(decodedEvents))
	}
	rate := float64(matched) / float64(total)
	if rate < 0.90 {
		t.Fatalf("SOW target: decode success rate %.1f%% on fixtures (need >=90%%)", rate*100)
	}

	// Latency smoke (single-event decode, counter fixture).
	var samples []time.Duration
	counter := fixtures[0]
	for i := 0; i < 200; i++ {
		start := time.Now()
		_ = runSOWFixture(t, root, counter)
		samples = append(samples, time.Since(start))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p95 := samples[len(samples)*95/100]
	if p95 > 500*time.Millisecond {
		t.Fatalf("SOW target: p95 decode latency %v exceeds 500ms", p95)
	}
}

func eventNames(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
