// Package cost provides cost tracking and reporting for bc agents.
//
// The package uses SQLite for persistent storage of cost records and budgets.
// Each workspace maintains its own cost database in .bc/costs.db.
//
// # Basic Usage
//
// Create and open a cost store:
//
//	store := cost.NewStore("/path/to/workspace")
//	if err := store.Open(); err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
// Record a cost entry:
//
//	record, err := store.Record("agent-1", "team-alpha", "claude-3-opus",
//	    1000,  // input tokens
//	    500,   // output tokens
//	    0.05,  // cost in USD
//	)
//
// Get cost summaries:
//
//	// By agent
//	summaries, _ := store.SummaryByAgent()
//
//	// By model
//	summaries, _ := store.SummaryByModel()
//
//	// Total workspace cost
//	total, _ := store.WorkspaceSummary()
//
// # Budgets
//
// Set and check budgets:
//
//	// Set monthly budget for workspace
//	store.SetBudget("workspace", cost.BudgetPeriodMonthly, 100.0, 0.8, false)
//
//	// Check budget status
//	status, _ := store.CheckBudget("workspace")
//	if status.IsNearLimit {
//	    log.Warn("approaching budget limit")
//	}
package cost

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rpuneet/bc/pkg/log"
)

// BudgetPeriod represents the time period for a budget.
type BudgetPeriod string

const (
	BudgetPeriodDaily   BudgetPeriod = "daily"
	BudgetPeriodWeekly  BudgetPeriod = "weekly"
	BudgetPeriodMonthly BudgetPeriod = "monthly"
)

// Budget represents a cost budget configuration.
type Budget struct {
	UpdatedAt time.Time    `json:"updated_at"`
	Period    BudgetPeriod `json:"period"`
	Scope     string       `json:"scope"` // "workspace", "agent:<id>", "team:<id>"
	ID        int64        `json:"id"`
	LimitUSD  float64      `json:"limit_usd"`
	AlertAt   float64      `json:"alert_at"`  // Percentage (0.0-1.0) at which to alert
	HardStop  bool         `json:"hard_stop"` // If true, stop when limit reached
}

// BudgetStatus represents the current status against a budget.
type BudgetStatus struct {
	Budget       *Budget `json:"budget"`
	CurrentSpend float64 `json:"current_spend"`
	Remaining    float64 `json:"remaining"`
	PercentUsed  float64 `json:"percent_used"`
	IsOverBudget bool    `json:"is_over_budget"`
	IsNearLimit  bool    `json:"is_near_limit"` // True if >= AlertAt percentage
}

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

	// #1011: Add WAL mode and busy timeout for better concurrency
	db, err := sql.Open("sqlite3", s.path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// #1011: Configure connection pool for SQLite's single-writer model
	// SQLite only allows one writer at a time, so limit connections
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	if err := s.initSchema(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	// #1011: Set optimal SQLite pragmas for performance
	ctx := context.Background()
	pragmas := `
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -2000;
		PRAGMA temp_store = MEMORY;
	`
	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		// Log warning but don't fail - pragmas are optional optimization
		_ = err // Ignore pragma errors
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

		CREATE TABLE IF NOT EXISTS cost_budgets (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			scope      TEXT NOT NULL UNIQUE,
			period     TEXT NOT NULL DEFAULT 'monthly' CHECK (period IN ('daily', 'weekly', 'monthly')),
			limit_usd  REAL NOT NULL DEFAULT 0,
			alert_at   REAL NOT NULL DEFAULT 0.8,
			hard_stop  INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
		CREATE INDEX IF NOT EXISTS idx_cost_budgets_scope ON cost_budgets(scope);
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
	var parseErr error
	r.Timestamp, parseErr = time.Parse(time.RFC3339, timestamp)
	if parseErr != nil {
		log.Warn("invalid timestamp in cost record", "id", r.ID, "raw", timestamp, "error", parseErr)
	}
	return &r, nil
}

// GetByAgent returns all cost records for an agent.
func (s *Store) GetByAgent(agentID string, limit int) ([]*Record, error) {
	return s.GetByAgentWithOffset(agentID, limit, 0)
}

// GetByAgentWithOffset returns cost records for an agent with pagination support.
func (s *Store) GetByAgentWithOffset(agentID string, limit, offset int) ([]*Record, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd, timestamp
		 FROM cost_records WHERE agent_id = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`,
		agentID, limit, offset,
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
	return s.GetAllWithOffset(limit, 0)
}

// GetAllWithOffset returns cost records with pagination support.
func (s *Store) GetAllWithOffset(limit, offset int) ([]*Record, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_id, team_id, model, input_tokens, output_tokens, total_tokens, cost_usd, timestamp
		 FROM cost_records ORDER BY timestamp DESC LIMIT ? OFFSET ?`,
		limit, offset,
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
		var parseErr error
		r.Timestamp, parseErr = time.Parse(time.RFC3339, timestamp)
		if parseErr != nil {
			log.Warn("invalid timestamp in cost record", "id", r.ID, "raw", timestamp, "error", parseErr)
		}
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

// SetBudget creates or updates a budget for the given scope.
func (s *Store) SetBudget(scope string, period BudgetPeriod, limitUSD, alertAt float64, hardStop bool) (*Budget, error) {
	ctx := context.Background()

	hardStopInt := 0
	if hardStop {
		hardStopInt = 1
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cost_budgets (scope, period, limit_usd, alert_at, hard_stop, updated_at)
		 VALUES (?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(scope) DO UPDATE SET
		   period = excluded.period,
		   limit_usd = excluded.limit_usd,
		   alert_at = excluded.alert_at,
		   hard_stop = excluded.hard_stop,
		   updated_at = excluded.updated_at`,
		scope, period, limitUSD, alertAt, hardStopInt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set budget: %w", err)
	}

	return s.GetBudget(scope)
}

// GetBudget returns the budget for a given scope.
func (s *Store) GetBudget(scope string) (*Budget, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT id, scope, period, limit_usd, alert_at, hard_stop, updated_at
		 FROM cost_budgets WHERE scope = ?`,
		scope,
	)

	var b Budget
	var hardStop int
	var updatedAt string

	err := row.Scan(&b.ID, &b.Scope, &b.Period, &b.LimitUSD, &b.AlertAt, &hardStop, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get budget: %w", err)
	}

	b.HardStop = hardStop == 1
	var parseErr error
	b.UpdatedAt, parseErr = time.Parse(time.RFC3339, updatedAt)
	if parseErr != nil {
		log.Warn("invalid timestamp in budget", "scope", b.Scope, "raw", updatedAt, "error", parseErr)
	}
	return &b, nil
}

// GetAllBudgets returns all configured budgets.
func (s *Store) GetAllBudgets() ([]*Budget, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scope, period, limit_usd, alert_at, hard_stop, updated_at
		 FROM cost_budgets ORDER BY scope`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var budgets []*Budget
	for rows.Next() {
		var b Budget
		var hardStop int
		var updatedAt string

		if err := rows.Scan(&b.ID, &b.Scope, &b.Period, &b.LimitUSD, &b.AlertAt, &hardStop, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}

		b.HardStop = hardStop == 1
		var parseErr error
		b.UpdatedAt, parseErr = time.Parse(time.RFC3339, updatedAt)
		if parseErr != nil {
			log.Warn("invalid timestamp in budget", "scope", b.Scope, "raw", updatedAt, "error", parseErr)
		}
		budgets = append(budgets, &b)
	}
	return budgets, rows.Err()
}

// DeleteBudget removes a budget for the given scope.
func (s *Store) DeleteBudget(scope string) error {
	ctx := context.Background()
	result, err := s.db.ExecContext(ctx, "DELETE FROM cost_budgets WHERE scope = ?", scope)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("budget not found for scope %q", scope)
	}
	return nil
}

// CheckBudget returns the current status against a budget.
func (s *Store) CheckBudget(scope string) (*BudgetStatus, error) {
	budget, err := s.GetBudget(scope)
	if err != nil {
		return nil, err
	}
	if budget == nil {
		return nil, nil
	}

	// Calculate period start time
	now := time.Now().UTC()
	var periodStart time.Time
	switch budget.Period {
	case BudgetPeriodDaily:
		periodStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case BudgetPeriodWeekly:
		// Start of week (Sunday)
		daysFromSunday := int(now.Weekday())
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-daysFromSunday, 0, 0, 0, 0, time.UTC)
	case BudgetPeriodMonthly:
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	// Get spend for the period
	var currentSpend float64
	ctx := context.Background()

	query := `SELECT COALESCE(SUM(cost_usd), 0) FROM cost_records WHERE timestamp >= ?`
	args := []any{periodStart.Format(time.RFC3339)}

	// Add scope filter
	if scope != "workspace" {
		if len(scope) > 6 && scope[:6] == "agent:" {
			query += " AND agent_id = ?"
			args = append(args, scope[6:])
		} else if len(scope) > 5 && scope[:5] == "team:" {
			query += " AND team_id = ?"
			args = append(args, scope[5:])
		}
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&currentSpend); err != nil {
		return nil, fmt.Errorf("failed to calculate current spend: %w", err)
	}

	status := &BudgetStatus{
		Budget:       budget,
		CurrentSpend: currentSpend,
		Remaining:    budget.LimitUSD - currentSpend,
	}

	if budget.LimitUSD > 0 {
		status.PercentUsed = currentSpend / budget.LimitUSD
		status.IsOverBudget = currentSpend >= budget.LimitUSD
		status.IsNearLimit = status.PercentUsed >= budget.AlertAt
	}

	if status.Remaining < 0 {
		status.Remaining = 0
	}

	return status, nil
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

// DailyCost represents aggregated cost data for a single day.
type DailyCost struct {
	Date         string  `json:"date"`
	CostUSD      float64 `json:"cost_usd"`
	TotalTokens  int64   `json:"total_tokens"`
	RecordCount  int64   `json:"record_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
}

// AgentDailyCost represents daily cost data for a specific agent.
type AgentDailyCost struct {
	AgentID      string  `json:"agent_id"`
	Date         string  `json:"date"`
	CostUSD      float64 `json:"cost_usd"`
	TotalTokens  int64   `json:"total_tokens"`
	RecordCount  int64   `json:"record_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
}

// Projection represents a cost projection based on historical data.
type Projection struct {
	Duration        time.Duration `json:"duration"`
	DailyAvgCost    float64       `json:"daily_avg_cost"`
	ProjectedCost   float64       `json:"projected_cost"`
	DaysAnalyzed    int           `json:"days_analyzed"`
	TotalHistorical float64       `json:"total_historical"`
}

// GetDailyCosts returns daily cost totals since the given time.
func (s *Store) GetDailyCosts(since time.Time) ([]*DailyCost, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT
			date(timestamp) as day,
			SUM(cost_usd) as cost,
			SUM(total_tokens) as tokens,
			COUNT(*) as records,
			SUM(input_tokens) as input,
			SUM(output_tokens) as output
		 FROM cost_records
		 WHERE timestamp >= ?
		 GROUP BY date(timestamp)
		 ORDER BY day ASC`,
		since.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily costs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var costs []*DailyCost
	for rows.Next() {
		var dc DailyCost
		if err := rows.Scan(&dc.Date, &dc.CostUSD, &dc.TotalTokens, &dc.RecordCount, &dc.InputTokens, &dc.OutputTokens); err != nil {
			return nil, fmt.Errorf("failed to scan daily cost: %w", err)
		}
		costs = append(costs, &dc)
	}
	return costs, rows.Err()
}

// GetAgentDailyCosts returns daily cost totals per agent since the given time.
func (s *Store) GetAgentDailyCosts(since time.Time) ([]*AgentDailyCost, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT
			agent_id,
			date(timestamp) as day,
			SUM(cost_usd) as cost,
			SUM(total_tokens) as tokens,
			COUNT(*) as records,
			SUM(input_tokens) as input,
			SUM(output_tokens) as output
		 FROM cost_records
		 WHERE timestamp >= ?
		 GROUP BY agent_id, date(timestamp)
		 ORDER BY agent_id, day ASC`,
		since.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent daily costs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var costs []*AgentDailyCost
	for rows.Next() {
		var adc AgentDailyCost
		if err := rows.Scan(&adc.AgentID, &adc.Date, &adc.CostUSD, &adc.TotalTokens, &adc.RecordCount, &adc.InputTokens, &adc.OutputTokens); err != nil {
			return nil, fmt.Errorf("failed to scan agent daily cost: %w", err)
		}
		costs = append(costs, &adc)
	}
	return costs, rows.Err()
}

// GetSummarySince returns a summary of costs since the given time.
func (s *Store) GetSummarySince(since time.Time) (*Summary, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(cost_usd), 0),
			COUNT(*)
		 FROM cost_records
		 WHERE timestamp >= ?`,
		since.Format(time.RFC3339),
	)

	var sum Summary
	if err := row.Scan(&sum.InputTokens, &sum.OutputTokens, &sum.TotalTokens, &sum.TotalCostUSD, &sum.RecordCount); err != nil {
		return nil, fmt.Errorf("failed to scan summary: %w", err)
	}
	return &sum, nil
}

// GetAgentSummarySince returns per-agent summaries since the given time.
func (s *Store) GetAgentSummarySince(since time.Time) ([]*Summary, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT
			agent_id,
			SUM(input_tokens),
			SUM(output_tokens),
			SUM(total_tokens),
			SUM(cost_usd),
			COUNT(*)
		 FROM cost_records
		 WHERE timestamp >= ?
		 GROUP BY agent_id
		 ORDER BY SUM(cost_usd) DESC`,
		since.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent summary since: %w", err)
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

// ProjectCost calculates a projected cost based on historical daily average.
func (s *Store) ProjectCost(lookbackDays int, projectDuration time.Duration) (*Projection, error) {
	since := time.Now().AddDate(0, 0, -lookbackDays)
	dailyCosts, err := s.GetDailyCosts(since)
	if err != nil {
		return nil, err
	}

	proj := &Projection{
		Duration:     projectDuration,
		DaysAnalyzed: len(dailyCosts),
	}

	if len(dailyCosts) == 0 {
		return proj, nil
	}

	// Calculate total and daily average
	for _, dc := range dailyCosts {
		proj.TotalHistorical += dc.CostUSD
	}
	proj.DailyAvgCost = proj.TotalHistorical / float64(len(dailyCosts))

	// Project forward
	projectDays := projectDuration.Hours() / 24
	proj.ProjectedCost = proj.DailyAvgCost * projectDays

	return proj, nil
}
