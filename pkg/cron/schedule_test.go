package cron

import (
	"testing"
	"time"
)

func TestValidateSchedule(t *testing.T) {
	valid := []string{
		"* * * * *",
		"0 9 * * *",
		"0 9 * * 1-5",
		"*/5 * * * *",
		"0,30 * * * *",
		"0 9-17 * * 1-5",
		"0 0 1 * *",
		"0 0 1 1 *",
		"59 23 31 12 6",
	}
	for _, expr := range valid {
		if err := ValidateSchedule(expr); err != nil {
			t.Errorf("ValidateSchedule(%q) = %v, want nil", expr, err)
		}
	}

	invalid := []struct {
		expr string
		desc string
	}{
		{"", "empty"},
		{"* * * *", "only 4 fields"},
		{"* * * * * *", "6 fields"},
		{"60 * * * *", "minute out of range"},
		{"* 24 * * *", "hour out of range"},
		{"* * 0 * *", "dom out of range (0)"},
		{"* * * 13 *", "month out of range"},
		{"* * * * 7", "dow out of range"},
		{"abc * * * *", "non-numeric minute"},
		{"1-5/0 * * * *", "zero step"},
	}
	for _, tc := range invalid {
		if err := ValidateSchedule(tc.expr); err == nil {
			t.Errorf("ValidateSchedule(%q) = nil, want error (%s)", tc.expr, tc.desc)
		}
	}
}

func TestNextRun(t *testing.T) {
	// Fixed base time: 2026-03-18 08:00 UTC (Wednesday)
	base := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		expr string
		want time.Time
	}{
		{
			// Every minute → next is 08:01
			expr: "* * * * *",
			want: time.Date(2026, 3, 18, 8, 1, 0, 0, time.UTC),
		},
		{
			// 9 AM daily → next is same day at 09:00
			expr: "0 9 * * *",
			want: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
		},
		{
			// Every 5 minutes → 08:05
			expr: "*/5 * * * *",
			want: time.Date(2026, 3, 18, 8, 5, 0, 0, time.UTC),
		},
		{
			// 9 AM weekdays (Mon-Fri); base is Wednesday → next is same day 09:00
			expr: "0 9 * * 1-5",
			want: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		got, err := NextRun(tc.expr, base)
		if err != nil {
			t.Errorf("NextRun(%q) error: %v", tc.expr, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("NextRun(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestNextRun_InvalidExpr(t *testing.T) {
	_, err := NextRun("not-valid", time.Now())
	if err == nil {
		t.Error("NextRun with invalid expr should return error")
	}
}
