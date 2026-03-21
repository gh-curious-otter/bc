// Package demon provides scheduled task management for bc.
package demon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Demon represents a scheduled task.
type Demon struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastRun     time.Time `json:"last_run,omitempty"`
	NextRun     time.Time `json:"next_run,omitempty"`
	Name        string    `json:"name"`
	Schedule    string    `json:"schedule"` // Cron expression (5-field)
	Command     string    `json:"command"`  // Command to execute
	Owner       string    `json:"owner,omitempty"`
	Description string    `json:"description,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`      // Inline prompt for AI-powered tasks
	PromptFile  string    `json:"prompt_file,omitempty"` // Path to prompt file
	RunCount    int       `json:"run_count,omitempty"`
	Enabled     bool      `json:"enabled"`
}

// RunLog represents a single execution of a demon.
type RunLog struct {
	Timestamp time.Time `json:"timestamp"`
	Duration  int64     `json:"duration_ms"` // Duration in milliseconds
	ExitCode  int       `json:"exit_code"`
	Success   bool      `json:"success"`
}

// CronSchedule represents a parsed cron expression.
type CronSchedule struct {
	Minute     []int // 0-59
	Hour       []int // 0-23
	DayOfMonth []int // 1-31
	Month      []int // 1-12
	DayOfWeek  []int // 0-6 (0 = Sunday)
}

// Store manages demon configurations.
type Store struct {
	demonsDir string
}

// NewStore creates a new demon store.
func NewStore(rootDir string) *Store {
	return &Store{
		demonsDir: filepath.Join(rootDir, ".bc", "demons"),
	}
}

// Init creates the demons directory if it doesn't exist.
func (s *Store) Init() error {
	return os.MkdirAll(s.demonsDir, 0750)
}

// Create creates a new demon configuration.
func (s *Store) Create(name, schedule, command string) (*Demon, error) {
	return s.CreateWithPrompt(name, schedule, command, "", "")
}

// CreateWithPrompt creates a new demon configuration with optional prompt support.
func (s *Store) CreateWithPrompt(name, schedule, command, prompt, promptFile string) (*Demon, error) {
	// Validate cron schedule
	if _, err := ParseCron(schedule); err != nil {
		return nil, fmt.Errorf("invalid cron schedule: %w", err)
	}

	// Check if demon already exists
	if s.Exists(name) {
		return nil, fmt.Errorf("demon %q already exists", name)
	}

	// Validate prompt options (can't have both inline and file)
	if prompt != "" && promptFile != "" {
		return nil, fmt.Errorf("cannot specify both --prompt and --prompt-file")
	}

	// If prompt file specified, validate it exists
	if promptFile != "" {
		if _, err := os.Stat(promptFile); err != nil {
			return nil, fmt.Errorf("prompt file not found: %w", err)
		}
	}

	now := time.Now().UTC()
	demon := &Demon{
		Name:       name,
		Schedule:   schedule,
		Command:    command,
		Prompt:     prompt,
		PromptFile: promptFile,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Calculate next run
	if cron, err := ParseCron(schedule); err == nil {
		demon.NextRun = cron.Next(now)
	}

	if err := s.save(demon); err != nil {
		return nil, err
	}

	return demon, nil
}

// Get retrieves a demon by name.
func (s *Store) Get(name string) (*Demon, error) {
	path := s.demonPath(name)
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from trusted demonsDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read demon: %w", err)
	}

	var demon Demon
	if err := json.Unmarshal(data, &demon); err != nil {
		return nil, fmt.Errorf("failed to parse demon: %w", err)
	}

	return &demon, nil
}

// List returns all demons.
func (s *Store) List() ([]*Demon, error) {
	entries, err := os.ReadDir(s.demonsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read demons dir: %w", err)
	}

	var demons []*Demon
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			demon, err := s.Get(name)
			if err != nil {
				continue // Skip invalid entries
			}
			// #1534 fix: Skip demons with empty names (corrupted data)
			if demon != nil && demon.Name != "" {
				demons = append(demons, demon)
			}
		}
	}

	return demons, nil
}

// Delete removes a demon.
func (s *Store) Delete(name string) error {
	path := s.demonPath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("demon %q not found", name)
		}
		return fmt.Errorf("failed to delete demon: %w", err)
	}
	return nil
}

// Exists checks if a demon exists.
func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.demonPath(name))
	return err == nil
}

// Update modifies an existing demon using the provided update function.
func (s *Store) Update(name string, updateFn func(*Demon)) error {
	demon, err := s.Get(name)
	if err != nil {
		return err
	}
	if demon == nil {
		return fmt.Errorf("demon %q not found", name)
	}

	updateFn(demon)
	demon.UpdatedAt = time.Now().UTC()

	return s.save(demon)
}

// ListByOwner returns all demons owned by a specific agent.
func (s *Store) ListByOwner(owner string) ([]*Demon, error) {
	demons, err := s.List()
	if err != nil {
		return nil, err
	}

	var result []*Demon
	for _, d := range demons {
		if d.Owner == owner {
			result = append(result, d)
		}
	}

	return result, nil
}

// ListEnabled returns all enabled demons.
func (s *Store) ListEnabled() ([]*Demon, error) {
	demons, err := s.List()
	if err != nil {
		return nil, err
	}

	var result []*Demon
	for _, d := range demons {
		if d.Enabled {
			result = append(result, d)
		}
	}

	return result, nil
}

// Enable enables a demon.
func (s *Store) Enable(name string) error {
	return s.Update(name, func(d *Demon) {
		d.Enabled = true
	})
}

// Disable disables a demon.
func (s *Store) Disable(name string) error {
	return s.Update(name, func(d *Demon) {
		d.Enabled = false
	})
}

// RecordRun updates the demon after a successful run.
func (s *Store) RecordRun(name string) error {
	return s.Update(name, func(d *Demon) {
		now := time.Now().UTC()
		d.LastRun = now
		d.RunCount++

		// Calculate next run time
		if cron, err := ParseCron(d.Schedule); err == nil {
			d.NextRun = cron.Next(now)
		}
	})
}

// RecordRunLog appends a run log entry for a demon.
func (s *Store) RecordRunLog(name string, log RunLog) error {
	if err := s.Init(); err != nil {
		return err
	}

	path := s.logPath(name)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) //nolint:gosec // path constructed from trusted demonsDir
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

// GetRunLogs retrieves the run logs for a demon.
// If limit > 0, returns only the most recent entries.
func (s *Store) GetRunLogs(name string, limit int) ([]RunLog, error) {
	path := s.logPath(name)
	f, err := os.Open(path) //nolint:gosec // path constructed from trusted demonsDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var logs []RunLog
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var log RunLog
		if err := json.Unmarshal(line, &log); err != nil {
			continue // Skip malformed entries
		}
		logs = append(logs, log)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Return most recent entries if limit is set
	if limit > 0 && len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}

	return logs, nil
}

func (s *Store) logPath(name string) string {
	return filepath.Join(s.demonsDir, name+".log.jsonl")
}

// SetOwner sets the owner of a demon.
func (s *Store) SetOwner(name, owner string) error {
	return s.Update(name, func(d *Demon) {
		d.Owner = owner
	})
}

// SetDescription sets the description of a demon.
func (s *Store) SetDescription(name, description string) error {
	return s.Update(name, func(d *Demon) {
		d.Description = description
	})
}

func (s *Store) save(demon *Demon) error {
	if err := s.Init(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(demon, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal demon: %w", err)
	}

	path := s.demonPath(demon.Name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write demon: %w", err)
	}

	return nil
}

func (s *Store) demonPath(name string) string {
	return filepath.Join(s.demonsDir, name+".json")
}

// ParseCron parses a 5-field cron expression.
// Format: minute hour day-of-month month day-of-week
// Example: "0 * * * *" = every hour
//
//	"*/5 * * * *" = every 5 minutes
//	"0 9 * * 1-5" = 9am weekdays
func ParseCron(expr string) (*CronSchedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}

	schedule := &CronSchedule{}
	var err error

	schedule.Minute, err = parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}

	schedule.Hour, err = parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}

	schedule.DayOfMonth, err = parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}

	schedule.Month, err = parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}

	schedule.DayOfWeek, err = parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return schedule, nil
}

// parseField parses a single cron field.
// Supports: *, */n, n, n-m, n,m,o
func parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return rangeSlice(min, max), nil
	}

	// Handle */n (step values)
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step value: %s", field)
		}
		var values []int
		for i := min; i <= max; i += step {
			values = append(values, i)
		}
		return values, nil
	}

	// Handle comma-separated values
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		var values []int
		for _, part := range parts {
			v, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || v < min || v > max {
				return nil, fmt.Errorf("invalid value: %s", part)
			}
			values = append(values, v)
		}
		return values, nil
	}

	// Handle ranges (n-m)
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range: %s", field)
		}
		start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || start < min || end > max || start > end {
			return nil, fmt.Errorf("invalid range: %s", field)
		}
		return rangeSlice(start, end), nil
	}

	// Single value
	v, err := strconv.Atoi(field)
	if err != nil || v < min || v > max {
		return nil, fmt.Errorf("invalid value: %s (must be %d-%d)", field, min, max)
	}
	return []int{v}, nil
}

func rangeSlice(min, max int) []int {
	result := make([]int, max-min+1)
	for i := range result {
		result[i] = min + i
	}
	return result
}

// Next calculates the next run time after the given time.
func (c *CronSchedule) Next(after time.Time) time.Time {
	// Start from the next minute
	t := after.Add(time.Minute).Truncate(time.Minute)

	// Try up to 4 years (enough for any valid cron)
	maxIterations := 60 * 24 * 366 * 4
	for range maxIterations {
		if c.matches(t) {
			return t
		}
		t = t.Add(time.Minute)
	}

	// Should never happen with valid cron
	return time.Time{}
}

func (c *CronSchedule) matches(t time.Time) bool {
	return slices.Contains(c.Minute, t.Minute()) &&
		slices.Contains(c.Hour, t.Hour()) &&
		slices.Contains(c.DayOfMonth, t.Day()) &&
		slices.Contains(c.Month, int(t.Month())) &&
		slices.Contains(c.DayOfWeek, int(t.Weekday()))
}
