package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDisplayDailyUsage(t *testing.T) {
	dailyJSON := `{
		"daily": [
			{
				"date": "2026-03-01",
				"inputTokens": 500,
				"outputTokens": 2000,
				"cacheCreationTokens": 100,
				"cacheReadTokens": 800,
				"totalTokens": 3400,
				"totalCost": 1.50,
				"modelsUsed": ["claude-opus-4-20250514"]
			},
			{
				"date": "2026-03-02",
				"inputTokens": 300,
				"outputTokens": 1500,
				"cacheCreationTokens": 50,
				"cacheReadTokens": 600,
				"totalTokens": 2450,
				"totalCost": 0.90,
				"modelsUsed": ["claude-sonnet-4-20250514"]
			}
		],
		"totals": {
			"inputTokens": 800,
			"outputTokens": 3500,
			"cacheCreationTokens": 150,
			"cacheReadTokens": 1400,
			"totalTokens": 5850,
			"totalCost": 2.40
		}
	}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displayDailyUsage(cmd, []byte(dailyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "Claude Code Daily Usage") {
		t.Error("missing header")
	}

	// Check column headers
	if !strings.Contains(output, "DATE") || !strings.Contains(output, "CACHE W") {
		t.Error("missing column headers")
	}

	// Check data rows
	if !strings.Contains(output, "2026-03-01") {
		t.Error("missing first date")
	}
	if !strings.Contains(output, "2026-03-02") {
		t.Error("missing second date")
	}

	// Check totals
	if !strings.Contains(output, "5850 tokens") {
		t.Error("missing total tokens")
	}
	if !strings.Contains(output, "$2.40") {
		t.Error("missing total cost")
	}

	// Check cache hit rate
	if !strings.Contains(output, "hit rate") {
		t.Error("missing cache hit rate")
	}
}

func TestDisplayDailyUsage_Empty(t *testing.T) {
	emptyJSON := `{"daily": [], "totals": {"inputTokens": 0, "outputTokens": 0, "cacheCreationTokens": 0, "cacheReadTokens": 0, "totalTokens": 0, "totalCost": 0}}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displayDailyUsage(cmd, []byte(emptyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No usage data found") {
		t.Error("expected empty message")
	}
}

func TestDisplayMonthlyUsage(t *testing.T) {
	monthlyJSON := `{
		"type": "monthly",
		"data": [
			{
				"month": "2026-02",
				"models": ["claude-opus-4-20250514"],
				"inputTokens": 5000,
				"outputTokens": 50000,
				"cacheCreationTokens": 500,
				"cacheReadTokens": 4000,
				"totalTokens": 59500,
				"costUSD": 25.00
			}
		],
		"summary": {
			"totalInputTokens": 5000,
			"totalOutputTokens": 50000,
			"totalCacheCreationTokens": 500,
			"totalCacheReadTokens": 4000,
			"totalTokens": 59500,
			"totalCostUSD": 25.00
		}
	}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displayMonthlyUsage(cmd, []byte(monthlyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Monthly Usage") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "2026-02") {
		t.Error("missing month")
	}
	if !strings.Contains(output, "$25.00") {
		t.Error("missing cost")
	}
}

func TestDisplayMonthlyUsage_Empty(t *testing.T) {
	emptyJSON := `{"type": "monthly", "data": [], "summary": {"totalInputTokens": 0, "totalOutputTokens": 0, "totalCacheCreationTokens": 0, "totalCacheReadTokens": 0, "totalTokens": 0, "totalCostUSD": 0}}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displayMonthlyUsage(cmd, []byte(emptyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No monthly usage data found") {
		t.Error("expected empty message")
	}
}

func TestDisplaySessionUsage(t *testing.T) {
	sessionJSON := `{
		"type": "session",
		"data": [
			{
				"session": "abc-123",
				"models": ["claude-opus-4-20250514"],
				"inputTokens": 1000,
				"outputTokens": 10000,
				"cacheCreationTokens": 200,
				"cacheReadTokens": 1500,
				"totalTokens": 12700,
				"costUSD": 5.50,
				"lastActivity": "2026-03-02"
			},
			{
				"session": "def-456",
				"models": ["claude-sonnet-4-20250514"],
				"inputTokens": 800,
				"outputTokens": 8000,
				"cacheCreationTokens": 100,
				"cacheReadTokens": 1200,
				"totalTokens": 10100,
				"costUSD": 3.20,
				"lastActivity": "2026-03-01"
			}
		],
		"summary": {
			"totalInputTokens": 1800,
			"totalOutputTokens": 18000,
			"totalCacheCreationTokens": 300,
			"totalCacheReadTokens": 2700,
			"totalTokens": 22800,
			"totalCostUSD": 8.70
		}
	}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displaySessionUsage(cmd, []byte(sessionJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Session Usage") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "abc-123") {
		t.Error("missing first session")
	}
	if !strings.Contains(output, "def-456") {
		t.Error("missing second session")
	}
	if !strings.Contains(output, "2 sessions") {
		t.Error("missing session count")
	}
	if !strings.Contains(output, "$8.70") {
		t.Error("missing total cost")
	}
}

func TestDisplaySessionUsage_Empty(t *testing.T) {
	emptyJSON := `{"type": "session", "data": [], "summary": {"totalInputTokens": 0, "totalOutputTokens": 0, "totalCacheCreationTokens": 0, "totalCacheReadTokens": 0, "totalTokens": 0, "totalCostUSD": 0}}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displaySessionUsage(cmd, []byte(emptyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No session usage data found") {
		t.Error("expected empty message")
	}
}

func TestDisplayDailyUsage_InvalidJSON(t *testing.T) {
	err := displayDailyUsage(costUsageCmd, []byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDisplayMonthlyUsage_InvalidJSON(t *testing.T) {
	err := displayMonthlyUsage(costUsageCmd, []byte("{invalid}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDisplaySessionUsage_InvalidJSON(t *testing.T) {
	err := displaySessionUsage(costUsageCmd, []byte("{invalid}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCcusageJSONTypes_Unmarshal(t *testing.T) {
	// Verify all JSON types unmarshal correctly
	t.Run("daily_entry", func(t *testing.T) {
		data := `{"date":"2026-03-01","inputTokens":100,"outputTokens":200,"cacheCreationTokens":10,"cacheReadTokens":50,"totalTokens":360,"totalCost":0.50,"modelsUsed":["opus"]}`
		var entry ccusageDailyEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if entry.Date != "2026-03-01" {
			t.Errorf("date = %q, want 2026-03-01", entry.Date)
		}
		if entry.TotalCost != 0.50 {
			t.Errorf("totalCost = %f, want 0.50", entry.TotalCost)
		}
		if len(entry.ModelsUsed) != 1 || entry.ModelsUsed[0] != "opus" {
			t.Errorf("modelsUsed = %v, want [opus]", entry.ModelsUsed)
		}
	})

	t.Run("monthly_entry", func(t *testing.T) {
		data := `{"month":"2026-02","models":["opus","sonnet"],"inputTokens":100,"outputTokens":200,"cacheCreationTokens":10,"cacheReadTokens":50,"totalTokens":360,"costUSD":1.25}`
		var entry ccusageMonthlyEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if entry.Month != "2026-02" {
			t.Errorf("month = %q, want 2026-02", entry.Month)
		}
		if entry.CostUSD != 1.25 {
			t.Errorf("costUSD = %f, want 1.25", entry.CostUSD)
		}
		if len(entry.Models) != 2 {
			t.Errorf("models count = %d, want 2", len(entry.Models))
		}
	})

	t.Run("session_entry", func(t *testing.T) {
		data := `{"session":"sess-1","models":["opus"],"inputTokens":100,"outputTokens":200,"cacheCreationTokens":10,"cacheReadTokens":50,"totalTokens":360,"costUSD":0.75,"lastActivity":"2026-03-01"}`
		var entry ccusageSessionEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if entry.Session != "sess-1" {
			t.Errorf("session = %q, want sess-1", entry.Session)
		}
		if entry.LastActivity != "2026-03-01" {
			t.Errorf("lastActivity = %q, want 2026-03-01", entry.LastActivity)
		}
	})
}

func TestCostUsageCmd_Flags(t *testing.T) {
	// Verify command and flags are registered
	cmd := costCmd
	usageCmd, _, err := cmd.Find([]string{"usage"})
	if err != nil {
		t.Fatalf("usage subcommand not found: %v", err)
	}
	if usageCmd.Use != "usage" {
		t.Errorf("Use = %q, want usage", usageCmd.Use)
	}

	// Check flags exist
	flags := []string{"monthly", "session", "since", "until"}
	for _, name := range flags {
		if usageCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not found", name)
		}
	}
}

func TestDisplayDailyUsage_NoCacheData(t *testing.T) {
	// Verify no cache line when no cache tokens
	dailyJSON := `{
		"daily": [{"date": "2026-03-01", "inputTokens": 500, "outputTokens": 2000, "cacheCreationTokens": 0, "cacheReadTokens": 0, "totalTokens": 2500, "totalCost": 1.00, "modelsUsed": ["opus"]}],
		"totals": {"inputTokens": 500, "outputTokens": 2000, "cacheCreationTokens": 0, "cacheReadTokens": 0, "totalTokens": 2500, "totalCost": 1.00}
	}`

	cmd := costUsageCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := displayDailyUsage(cmd, []byte(dailyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(buf.String(), "hit rate") {
		t.Error("should not show cache hit rate when no cache data")
	}
}
