// Package cost provides cost tracking and reporting for bc.
package cost

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Record represents a cost entry for an API call.
type Record struct {
	Timestamp    time.Time `json:"timestamp"`
	AgentID      string    `json:"agent_id"`
	Model        string    `json:"model"`
	TeamID       string    `json:"team_id,omitempty"`
	ID           int64     `json:"id"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	CostUSD      float64   `json:"cost_usd"`
}

// Summary represents aggregated cost data.
type Summary struct {
	AgentID      string  `json:"agent_id,omitempty"`
	TeamID       string  `json:"team_id,omitempty"`
	Model        string  `json:"model,omitempty"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	RecordCount  int64   `json:"record_count"`
}

// Store provides SQLite-backed cost tracking.
type Store struct {
	db   *sql.DB
	path string
}

// NewStore creates a new cost store for the given workspace.
func NewStore(workspacePath string) *Store {
	return &Store{
		path: filepath.Join(workspacePath, ".bc", "costs.db"),
	}
}

// Open initializes the SQLite database.
func (s *Store) Open() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", s.path+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := s.initSchema(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	s.db = db
	return nil
}

// initSchema creates the database tables.
func (s *Store) initSchema(db *sql.DB) error {
	ctx := context.Background()

	schema := `
		CREATE TABLE IF NOT EXISTS cost_records (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id      TEXT NOT NULL,
			team_id       TEXT,
			model         TEXT NOT NULL,
			input_tokens  INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens  INTEGER NOT NULL DEFAULT 0,
			cost_usd      REAL NOT NULL DEFAULT 0,
			timestamp     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
		CREATE INDEX IF NOT EXISTS idx_cost_records_agent ON cost_records(agent_id);
		CREATE INDEX IF NOT EXISTS idx_cost_records_team ON cost_records(team_id);
		CREATE INDEX IF NOT EXISTS idx_cost_records_model ON cost_records(model);
		CREATE INDEX IF NOT EXISTS idx_cost_records_timestamp ON cost_records(timestamp DESC);
	`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return err
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Record adds a new cost record.
func (s *Store) Record(agentID, teamID, model string, inputTokens, outputTokens int64, costUSD float64) (*Record, error) {
	ctx := context.Background()
	totalTokens := inputTokens + outputTokens

	var teamPtr *string
	if teamID != "" {
		teamPtr = &teamID
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO cost_records (agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		agentID, teamPtr, model, inputTokens, outputTokens, totalTokens, costUSD,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record cost: %w", err)
	}

	id, _ := result.LastInsertId()
	return s.GetByID(id)
}

// GetByID returns a cost record by ID.
func (s *Store) GetByID(id int64) (*Record, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT id, agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd, timestamp
		 FROM cost_records WHERE id = ?`,
		id,
	)
	return s.scanRecord(row)
}

func (s *Store) scanRecord(row *sql.Row) (*Record, error) {
	var r Record
	var timestamp string
	var teamID sql.NullString

	err := row.Scan(&r.ID, &r.AgentID, &teamID, &r.Model, &r.InputTokens, &r.OutputTokens, &r.TotalTokens, &r.CostUSD, &timestamp)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	r.TeamID = teamID.String
	r.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
	return &r, nil
}

// GetByAgent returns all cost records for an agent.
func (s *Store) GetByAgent(agentID string, limit int) ([]*Record, error) {
	if limit <= 0 {
		limit = 100
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd, timestamp
		 FROM cost_records WHERE agent_id = ? ORDER BY timestamp DESC LIMIT ?`,
		agentID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get records by agent: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return s.scanRecords(rows)
}

// GetByTeam returns all cost records for a team.
func (s *Store) GetByTeam(teamID string, limit int) ([]*Record, error) {
	if limit <= 0 {
		limit = 100
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd, timestamp
		 FROM cost_records WHERE team_id = ? ORDER BY timestamp DESC LIMIT ?`,
		teamID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get records by team: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return s.scanRecords(rows)
}

// GetAll returns all cost records.
func (s *Store) GetAll(limit int) ([]*Record, error) {
	if limit <= 0 {
		limit = 100
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd, timestamp
		 FROM cost_records ORDER BY timestamp DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all records: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return s.scanRecords(rows)
}

func (s *Store) scanRecords(rows *sql.Rows) ([]*Record, error) {
	var records []*Record
	for rows.Next() {
		var r Record
		var timestamp string
		var teamID sql.NullString

		if err := rows.Scan(&r.ID, &r.AgentID, &teamID, &r.Model, &r.InputTokens, &r.OutputTokens, &r.TotalTokens, &r.CostUSD, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		r.TeamID = teamID.String
		r.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
		records = append(records, &r)
	}
	return records, rows.Err()
}

// SummaryByAgent returns aggregated costs per agent.
func (s *Store) SummaryByAgent() ([]*Summary, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT agent_id, SUM(input_tokens), SUM(output_tokens), SUM(total_tokens), SUM(cost_usd), COUNT(*)
		 FROM cost_records GROUP BY agent_id ORDER BY SUM(cost_usd) DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary by agent: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var summaries []*Summary
	for rows.Next() {
		var sum Summary
		if err := rows.Scan(&sum.AgentID, &sum.InputTokens, &sum.OutputTokens, &sum.TotalTokens, &sum.TotalCostUSD, &sum.RecordCount); err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		summaries = append(summaries, &sum)
	}
	return summaries, rows.Err()
}

// SummaryByTeam returns aggregated costs per team.
func (s *Store) SummaryByTeam() ([]*Summary, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT team_id, SUM(input_tokens), SUM(output_tokens), SUM(total_tokens), SUM(cost_usd), COUNT(*)
		 FROM cost_records WHERE team_id IS NOT NULL GROUP BY team_id ORDER BY SUM(cost_usd) DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary by team: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var summaries []*Summary
	for rows.Next() {
		var sum Summary
		var teamID sql.NullString
		if err := rows.Scan(&teamID, &sum.InputTokens, &sum.OutputTokens, &sum.TotalTokens, &sum.TotalCostUSD, &sum.RecordCount); err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		sum.TeamID = teamID.String
		summaries = append(summaries, &sum)
	}
	return summaries, rows.Err()
}

// SummaryByModel returns aggregated costs per model.
func (s *Store) SummaryByModel() ([]*Summary, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT model, SUM(input_tokens), SUM(output_tokens), SUM(total_tokens), SUM(cost_usd), COUNT(*)
		 FROM cost_records GROUP BY model ORDER BY SUM(cost_usd) DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary by model: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var summaries []*Summary
	for rows.Next() {
		var sum Summary
		if err := rows.Scan(&sum.Model, &sum.InputTokens, &sum.OutputTokens, &sum.TotalTokens, &sum.TotalCostUSD, &sum.RecordCount); err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		summaries = append(summaries, &sum)
	}
	return summaries, rows.Err()
}

// WorkspaceSummary returns the total cost summary for the entire workspace.
func (s *Store) WorkspaceSummary() (*Summary, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT SUM(input_tokens), SUM(output_tokens), SUM(total_tokens), SUM(cost_usd), COUNT(*)
		 FROM cost_records`,
	)

	var sum Summary
	var inputTokens, outputTokens, totalTokens sql.NullInt64
	var costUSD sql.NullFloat64
	var recordCount sql.NullInt64

	if err := row.Scan(&inputTokens, &outputTokens, &totalTokens, &costUSD, &recordCount); err != nil {
		return nil, fmt.Errorf("failed to scan workspace summary: %w", err)
	}

	sum.InputTokens = inputTokens.Int64
	sum.OutputTokens = outputTokens.Int64
	sum.TotalTokens = totalTokens.Int64
	sum.TotalCostUSD = costUSD.Float64
	sum.RecordCount = recordCount.Int64

	return &sum, nil
}

// AgentSummary returns the cost summary for a specific agent.
func (s *Store) AgentSummary(agentID string) (*Summary, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT SUM(input_tokens), SUM(output_tokens), SUM(total_tokens), SUM(cost_usd), COUNT(*)
		 FROM cost_records WHERE agent_id = ?`,
		agentID,
	)

	var sum Summary
	var inputTokens, outputTokens, totalTokens sql.NullInt64
	var costUSD sql.NullFloat64
	var recordCount sql.NullInt64

	if err := row.Scan(&inputTokens, &outputTokens, &totalTokens, &costUSD, &recordCount); err != nil {
		return nil, fmt.Errorf("failed to scan agent summary: %w", err)
	}

	sum.AgentID = agentID
	sum.InputTokens = inputTokens.Int64
	sum.OutputTokens = outputTokens.Int64
	sum.TotalTokens = totalTokens.Int64
	sum.TotalCostUSD = costUSD.Float64
	sum.RecordCount = recordCount.Int64

	return &sum, nil
}

// TeamSummary returns the cost summary for a specific team.
func (s *Store) TeamSummary(teamID string) (*Summary, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT SUM(input_tokens), SUM(output_tokens), SUM(total_tokens), SUM(cost_usd), COUNT(*)
		 FROM cost_records WHERE team_id = ?`,
		teamID,
	)

	var sum Summary
	var inputTokens, outputTokens, totalTokens sql.NullInt64
	var costUSD sql.NullFloat64
	var recordCount sql.NullInt64

	if err := row.Scan(&inputTokens, &outputTokens, &totalTokens, &costUSD, &recordCount); err != nil {
		return nil, fmt.Errorf("failed to scan team summary: %w", err)
	}

	sum.TeamID = teamID
	sum.InputTokens = inputTokens.Int64
	sum.OutputTokens = outputTokens.Int64
	sum.TotalTokens = totalTokens.Int64
	sum.TotalCostUSD = costUSD.Float64
	sum.RecordCount = recordCount.Int64

	return &sum, nil
}

// Clear removes all cost records.
func (s *Store) Clear() error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx, "DELETE FROM cost_records")
	if err != nil {
		return fmt.Errorf("failed to clear cost records: %w", err)
	}
	return nil
}
