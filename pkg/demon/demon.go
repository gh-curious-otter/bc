// Package demon provides scheduled task (demon) management for bc.
// Demons are background tasks that run on a schedule, owned by agents.
package demon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Demon represents a scheduled background task.
type Demon struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastRunAt   time.Time `json:"last_run_at,omitempty"`
	NextRunAt   time.Time `json:"next_run_at,omitempty"`
	Name        string    `json:"name"`
	Schedule    string    `json:"schedule"` // cron expression
	Command     string    `json:"command"`  // command to execute
	Owner       string    `json:"owner"`    // agent that owns this demon
	Description string    `json:"description,omitempty"`
	RunCount    int       `json:"run_count"`
	Enabled     bool      `json:"enabled"`
}

// Store manages demon persistence in .bc/demons.json
type Store struct {
	path string
	mu   sync.RWMutex
}

// NewStore creates a demon store at the given .bc directory.
func NewStore(bcDir string) *Store {
	return &Store{
		path: filepath.Join(bcDir, "demons.json"),
	}
}

// storeData is the JSON structure for the demons file.
type storeData struct {
	Demons []Demon `json:"demons"`
}

// Load reads all demons from the store.
func (s *Store) Load() ([]Demon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path) //nolint:gosec // path constructed from trusted bcDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read demons file: %w", err)
	}

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal demons: %w", err)
	}

	return sd.Demons, nil
}

// Save persists all demons to the store.
func (s *Store) Save(demons []Demon) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd := storeData{Demons: demons}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal demons: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write demons file: %w", err)
	}

	return nil
}

// Create adds a new demon to the store.
func (s *Store) Create(demon *Demon) error {
	demons, err := s.Load()
	if err != nil {
		return err
	}

	// Check for duplicate name
	for _, d := range demons {
		if d.Name == demon.Name {
			return fmt.Errorf("demon %q already exists", demon.Name)
		}
	}

	now := time.Now()
	demon.CreatedAt = now
	demon.UpdatedAt = now

	demons = append(demons, *demon)
	return s.Save(demons)
}

// Get retrieves a demon by name.
func (s *Store) Get(name string) (*Demon, error) {
	demons, err := s.Load()
	if err != nil {
		return nil, err
	}

	for _, d := range demons {
		if d.Name == name {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("demon %q not found", name)
}

// Update modifies an existing demon.
func (s *Store) Update(name string, updateFn func(*Demon)) error {
	demons, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range demons {
		if demons[i].Name == name {
			updateFn(&demons[i])
			demons[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("demon %q not found", name)
	}

	return s.Save(demons)
}

// Delete removes a demon by name.
func (s *Store) Delete(name string) error {
	demons, err := s.Load()
	if err != nil {
		return err
	}

	filtered := make([]Demon, 0, len(demons))
	found := false
	for _, d := range demons {
		if d.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, d)
	}

	if !found {
		return fmt.Errorf("demon %q not found", name)
	}

	return s.Save(filtered)
}

// ListByOwner returns all demons owned by a specific agent.
func (s *Store) ListByOwner(owner string) ([]Demon, error) {
	demons, err := s.Load()
	if err != nil {
		return nil, err
	}

	var result []Demon
	for _, d := range demons {
		if d.Owner == owner {
			result = append(result, d)
		}
	}

	return result, nil
}

// ListEnabled returns all enabled demons.
func (s *Store) ListEnabled() ([]Demon, error) {
	demons, err := s.Load()
	if err != nil {
		return nil, err
	}

	var result []Demon
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

// RecordRun updates the demon after a run.
func (s *Store) RecordRun(name string, nextRun time.Time) error {
	return s.Update(name, func(d *Demon) {
		d.LastRunAt = time.Now()
		d.NextRunAt = nextRun
		d.RunCount++
	})
}
