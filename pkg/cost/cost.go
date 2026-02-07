// Package cost provides cost tracking for bc.
// Tracks API usage costs per agent.
package cost

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Record represents a single cost entry.
type Record struct {
	Timestamp    time.Time `json:"timestamp"`
	Agent        string    `json:"agent"`
	Model        string    `json:"model"`
	Operation    string    `json:"operation,omitempty"` // e.g., "completion", "embedding"
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
}

// Summary aggregates cost data.
//
//nolint:govet // fieldalignment: keeping fields in logical order for readability
type Summary struct {
	TotalCostUSD      float64   `json:"total_cost_usd"`
	TotalInputTokens  int       `json:"total_input_tokens"`
	TotalOutputTokens int       `json:"total_output_tokens"`
	RecordCount       int       `json:"record_count"`
	FirstRecord       time.Time `json:"first_record,omitempty"`
	LastRecord        time.Time `json:"last_record,omitempty"`
}

// Store manages cost records.
type Store struct {
	costsDir string
	mu       sync.RWMutex
}

// NewStore creates a new cost store.
func NewStore(rootDir string) *Store {
	return &Store{
		costsDir: filepath.Join(rootDir, ".bc", "costs"),
	}
}

// Init creates the costs directory if it doesn't exist.
func (s *Store) Init() error {
	return os.MkdirAll(s.costsDir, 0750)
}

// Record adds a cost record for an agent.
func (s *Store) Record(record *Record) error {
	if record.Agent == "" {
		return fmt.Errorf("agent name is required")
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadAgent(record.Agent)
	if err != nil {
		return err
	}

	records = append(records, *record)
	return s.saveAgent(record.Agent, records)
}

// GetAgentSummary returns a cost summary for a specific agent.
func (s *Store) GetAgentSummary(agent string) (*Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, err := s.loadAgent(agent)
	if err != nil {
		return nil, err
	}

	return summarize(records), nil
}

// GetWorkspaceSummary returns a cost summary for the entire workspace.
func (s *Store) GetWorkspaceSummary() (*Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents, err := s.listAgents()
	if err != nil {
		return nil, err
	}

	var allRecords []Record
	for _, agent := range agents {
		records, loadErr := s.loadAgent(agent)
		if loadErr != nil {
			continue
		}
		allRecords = append(allRecords, records...)
	}

	return summarize(allRecords), nil
}

// GetAgentRecords returns all cost records for an agent.
func (s *Store) GetAgentRecords(agent string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadAgent(agent)
}

// ListAgents returns all agents with cost records.
func (s *Store) ListAgents() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listAgents()
}

func (s *Store) listAgents() ([]string, error) {
	entries, err := os.ReadDir(s.costsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read costs dir: %w", err)
	}

	var agents []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()
			agents = append(agents, name[:len(name)-5]) // Remove .json extension
		}
	}
	return agents, nil
}

func (s *Store) agentPath(agent string) string {
	return filepath.Join(s.costsDir, agent+".json")
}

func (s *Store) loadAgent(agent string) ([]Record, error) {
	path := s.agentPath(agent)
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from trusted costsDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read cost records: %w", err)
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to parse cost records: %w", err)
	}

	return records, nil
}

func (s *Store) saveAgent(agent string, records []Record) error {
	if err := s.Init(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cost records: %w", err)
	}

	path := s.agentPath(agent)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write cost records: %w", err)
	}

	return nil
}

func summarize(records []Record) *Summary {
	summary := &Summary{}
	for i, r := range records {
		summary.TotalCostUSD += r.CostUSD
		summary.TotalInputTokens += r.InputTokens
		summary.TotalOutputTokens += r.OutputTokens
		summary.RecordCount++

		if i == 0 || r.Timestamp.Before(summary.FirstRecord) {
			summary.FirstRecord = r.Timestamp
		}
		if r.Timestamp.After(summary.LastRecord) {
			summary.LastRecord = r.Timestamp
		}
	}
	return summary
}
