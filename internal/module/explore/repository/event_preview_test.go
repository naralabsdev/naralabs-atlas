package repository

import "testing"

func TestEventTypeFromTopics(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "event"},
		{name: "null", input: "null", want: "event"},
		{name: "symbol tag", input: `[{"symbol":"transfer"}]`, want: "transfer"},
		{name: "plain string", input: `["mint"]`, want: "mint"},
		{name: "invalid json", input: "{", want: "event"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := eventTypeFromTopics(tc.input); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSummaryPreviewFee(t *testing.T) {
	got := summaryPreview("fee", `[{"symbol":"fee"}]`, `{"i128":"-9613"}`)
	want := "Network fee −0.0009613 XLM"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSummaryPreviewFeeCredit(t *testing.T) {
	got := summaryPreview("fee", "[]", `{"i128":"17211"}`)
	want := "Fee credit 0.0017211 XLM"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSummaryPreviewBoolEvent(t *testing.T) {
	got := summaryPreview("forwarder_ReportProcessed", "[]", `{"bool":true}`)
	want := "Forwarder Report Processed · Confirmed"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSummaryPreviewMapEvent(t *testing.T) {
	got := summaryPreview("sentinel", "[]", `{"map":{"entries":[{"k":{"symbol":"delay_hours"},"v":{"u32":24}}]}}`)
	if got == "" || got == "Sentinel" {
		t.Fatalf("unexpected summary %q", got)
	}
}

func TestFormatEventLabel(t *testing.T) {
	if got := formatEventLabel("forwarder_ReportProcessed"); got != "Forwarder Report Processed" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatStroopsAmount(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"9613", "0.0009613 XLM"},
		{"-9613", "−0.0009613 XLM"},
		{"10000000", "1 XLM"},
		{"10000001", "1.0000001 XLM"},
	}
	for _, tc := range tests {
		if got := formatStroopsAmount(tc.in); got != tc.want {
			t.Fatalf("formatStroopsAmount(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDecodeStatus(t *testing.T) {
	if decodeStatus(0) != "raw" {
		t.Fatal("expected raw")
	}
	if decodeStatus(1) != "decoded" {
		t.Fatal("expected decoded")
	}
}

func TestTaggedString(t *testing.T) {
	if got := taggedString([]byte(`{"symbol":"approve"}`), "symbol"); got != "approve" {
		t.Fatalf("got %q", got)
	}
	if got := taggedString([]byte(`{"other":"x"}`), "symbol"); got != "" {
		t.Fatalf("got %q", got)
	}
}
