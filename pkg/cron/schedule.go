package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// fieldSpec holds the valid values for one cron field.
type fieldSpec struct {
	values [60]bool // large enough for any field
}

// matches reports whether v is set in this field spec.
func (f *fieldSpec) matches(v int) bool {
	if v < 0 || v >= len(f.values) {
		return false
	}
	return f.values[v]
}

// parsedSchedule holds all five parsed cron fields.
type parsedSchedule struct {
	minute  fieldSpec
	hour    fieldSpec
	dom     fieldSpec
	month   fieldSpec
	dow     fieldSpec
}

// matches reports whether t satisfies the schedule.
func (s *parsedSchedule) matches(t time.Time) bool {
	return s.minute.matches(t.Minute()) &&
		s.hour.matches(t.Hour()) &&
		s.dom.matches(t.Day()) &&
		s.month.matches(int(t.Month())) &&
		s.dow.matches(int(t.Weekday()))
}

// parseField parses a single cron field expression into a fieldSpec.
// min and max are the inclusive bounds for this field (e.g., 0-59 for minutes).
func parseField(expr string, min, max int) (fieldSpec, error) {
	var f fieldSpec

	for _, part := range strings.Split(expr, ",") {
		if err := parseFieldPart(part, min, max, &f); err != nil {
			return f, err
		}
	}
	return f, nil
}

func parseFieldPart(part string, min, max int, f *fieldSpec) error {
	// Handle step: "*/5", "1-10/2", "5/2"
	step := 1
	if idx := strings.Index(part, "/"); idx >= 0 {
		var err error
		step, err = strconv.Atoi(part[idx+1:])
		if err != nil || step < 1 {
			return fmt.Errorf("invalid step %q", part[idx+1:])
		}
		part = part[:idx]
	}

	// Handle wildcard
	if part == "*" {
		for v := min; v <= max; v += step {
			f.values[v] = true
		}
		return nil
	}

	// Handle range: "1-5"
	if idx := strings.Index(part, "-"); idx >= 0 {
		lo, err := strconv.Atoi(part[:idx])
		if err != nil {
			return fmt.Errorf("invalid range start %q", part[:idx])
		}
		hi, err := strconv.Atoi(part[idx+1:])
		if err != nil {
			return fmt.Errorf("invalid range end %q", part[idx+1:])
		}
		if lo < min || hi > max || lo > hi {
			return fmt.Errorf("range %d-%d out of bounds [%d-%d]", lo, hi, min, max)
		}
		for v := lo; v <= hi; v += step {
			f.values[v] = true
		}
		return nil
	}

	// Single value
	v, err := strconv.Atoi(part)
	if err != nil {
		return fmt.Errorf("invalid value %q", part)
	}
	if v < min || v > max {
		return fmt.Errorf("value %d out of bounds [%d-%d]", v, min, max)
	}
	// Apply step from a single value: e.g. "5/2" means 5,7,9,...
	for i := v; i <= max; i += step {
		f.values[i] = true
	}
	return nil
}

// parseSchedule parses a 5-field cron expression.
// Fields: minute hour day-of-month month day-of-week
// Example: "0 9 * * 1-5" (9 AM weekdays)
func parseSchedule(expr string) (*parsedSchedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have exactly 5 fields (got %d): %q", len(fields), expr)
	}

	minute, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field %q: %w", fields[0], err)
	}
	hour, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field %q: %w", fields[1], err)
	}
	dom, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field %q: %w", fields[2], err)
	}
	month, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field %q: %w", fields[3], err)
	}
	dow, err := parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field %q: %w", fields[4], err)
	}

	return &parsedSchedule{
		minute: minute,
		hour:   hour,
		dom:    dom,
		month:  month,
		dow:    dow,
	}, nil
}

// ValidateSchedule validates a 5-field cron expression.
// Returns an error describing the problem, or nil if valid.
func ValidateSchedule(expr string) error {
	_, err := parseSchedule(expr)
	return err
}

// NextRun returns the next time after `from` that matches the cron expression.
// Returns an error if the expression is invalid or no match is found within 4 years.
func NextRun(expr string, from time.Time) (time.Time, error) {
	sched, err := parseSchedule(expr)
	if err != nil {
		return time.Time{}, err
	}

	// Advance by one minute and truncate seconds so we don't re-trigger now.
	t := from.Add(time.Minute).Truncate(time.Minute)

	// Iterate minute by minute; cap at ~4 years to handle pathological expressions.
	const maxMinutes = 525600 * 4
	for i := 0; i < maxMinutes; i++ {
		if sched.matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("no matching time found within 4 years for schedule %q", expr)
}
