package cost

import (
	"strings"
	"testing"
	"time"
)

func TestParseSessionReader_AssistantEntries(t *testing.T) {
	jsonl := strings.NewReader(`
{"type":"user","sessionId":"s1","timestamp":"2026-03-01T10:00:00Z","cwd":"/ws","message":{"role":"user","content":"hello"}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-03-01T10:00:01Z","cwd":"/ws","message":{"model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":10}}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-03-01T10:00:02Z","cwd":"/ws","message":{"model":"claude-sonnet-4-5","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`)

	entries, err := parseSessionReader(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	// Second assistant entry has all-zero tokens and should be skipped.
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Model != "claude-opus-4-6" {
		t.Errorf("want model claude-opus-4-6, got %s", e.Model)
	}
	if e.InputTokens != 100 {
		t.Errorf("want 100 input tokens, got %d", e.InputTokens)
	}
	if e.OutputTokens != 50 {
		t.Errorf("want 50 output tokens, got %d", e.OutputTokens)
	}
	if e.CacheCreationTokens != 200 {
		t.Errorf("want 200 cache_creation tokens, got %d", e.CacheCreationTokens)
	}
	if e.CacheReadTokens != 10 {
		t.Errorf("want 10 cache_read tokens, got %d", e.CacheReadTokens)
	}
	if e.SessionID != "s1" {
		t.Errorf("want session s1, got %s", e.SessionID)
	}
}

func TestParseSessionReader_SkipsMalformedLines(t *testing.T) {
	jsonl := strings.NewReader(`not json
{"type":"assistant","sessionId":"s2","timestamp":"2026-03-01T10:00:01Z","cwd":"/ws","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":5,"output_tokens":3}}}
`)
	entries, err := parseSessionReader(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
}

func TestParseSessionReader_TimestampParsed(t *testing.T) {
	jsonl := strings.NewReader(`{"type":"assistant","sessionId":"s3","timestamp":"2026-03-15T14:30:00.123Z","cwd":"/","message":{"model":"claude-opus-4-6","usage":{"input_tokens":1,"output_tokens":1}}}`)
	entries, err := parseSessionReader(jsonl)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	want := time.Date(2026, 3, 15, 14, 30, 0, 123_000_000, time.UTC)
	if !entries[0].Timestamp.Equal(want) {
		t.Errorf("want timestamp %v, got %v", want, entries[0].Timestamp)
	}
}

func TestCalcCost_KnownModel(t *testing.T) {
	// claude-opus-4: $15/M input, $75/M output
	cost := CalcCost("claude-opus-4-6", 1_000_000, 0, 0, 0)
	if cost < 14.9 || cost > 15.1 {
		t.Errorf("want ~$15 for 1M input tokens on opus-4, got %.4f", cost)
	}
}

func TestCalcCost_CacheTokens(t *testing.T) {
	// cache_write for opus-4: $18.75/M
	cost := CalcCost("claude-opus-4-6", 0, 0, 1_000_000, 0)
	if cost < 18.7 || cost > 18.8 {
		t.Errorf("want ~$18.75 for 1M cache_write tokens on opus-4, got %.4f", cost)
	}
}

func TestCalcCost_UnknownModelFallsBack(t *testing.T) {
	// Unknown model should use defaultPricing (sonnet-tier)
	cost := CalcCost("claude-unknown-99", 1_000_000, 0, 0, 0)
	if cost <= 0 {
		t.Errorf("expected positive cost for unknown model, got %f", cost)
	}
}

func TestPricingFor_Prefixes(t *testing.T) {
	cases := []struct {
		model string
		wantInput float64
	}{
		{"claude-opus-4-6", 15.00},
		{"claude-opus-4-5-20250514", 15.00},
		{"claude-sonnet-4-5-20250514", 3.00},
		{"claude-haiku-4-5-20251001", 0.80},
		{"claude-3-5-sonnet-20241022", 3.00},
		{"claude-3-5-haiku-20241022", 0.80},
		{"claude-3-opus-20240229", 15.00},
	}
	for _, tc := range cases {
		p := PricingFor(tc.model)
		if p.InputPerM != tc.wantInput {
			t.Errorf("model %s: want input $%.2f/M, got $%.2f/M", tc.model, tc.wantInput, p.InputPerM)
		}
	}
}
