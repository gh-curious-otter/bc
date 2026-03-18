package cost

import "time"

// Backend is the storage interface for cost tracking.
// Store is the default SQLite implementation.
type Backend interface {
	// Record adds a new cost entry.
	Record(agentID, teamID, model string, inputTokens, outputTokens int64, costUSD float64) (*Record, error)
	// GetByID returns a cost record by ID.
	GetByID(id int64) (*Record, error)
	// GetByAgent returns cost records for an agent.
	GetByAgent(agentID string, limit int) ([]*Record, error)
	// GetByAgentWithOffset returns paginated cost records for an agent.
	GetByAgentWithOffset(agentID string, limit, offset int) ([]*Record, error)
	// GetByTeam returns cost records for a team.
	GetByTeam(teamID string, limit int) ([]*Record, error)
	// GetAll returns all cost records.
	GetAll(limit int) ([]*Record, error)
	// GetAllWithOffset returns paginated cost records.
	GetAllWithOffset(limit, offset int) ([]*Record, error)
	// SummaryByAgent returns aggregated costs grouped by agent.
	SummaryByAgent() ([]*Summary, error)
	// SummaryByTeam returns aggregated costs grouped by team.
	SummaryByTeam() ([]*Summary, error)
	// SummaryByModel returns aggregated costs grouped by model.
	SummaryByModel() ([]*Summary, error)
	// WorkspaceSummary returns the total workspace cost summary.
	WorkspaceSummary() (*Summary, error)
	// AgentSummary returns the cost summary for a specific agent.
	AgentSummary(agentID string) (*Summary, error)
	// TeamSummary returns the cost summary for a specific team.
	TeamSummary(teamID string) (*Summary, error)
	// SetBudget creates or updates a budget for a scope.
	SetBudget(scope string, period BudgetPeriod, limitUSD, alertAt float64, hardStop bool) (*Budget, error)
	// GetBudget returns the budget for a scope.
	GetBudget(scope string) (*Budget, error)
	// GetAllBudgets returns all configured budgets.
	GetAllBudgets() ([]*Budget, error)
	// DeleteBudget removes a budget for a scope.
	DeleteBudget(scope string) error
	// CheckBudget returns the current budget status for a scope.
	CheckBudget(scope string) (*BudgetStatus, error)
	// Clear removes all cost records.
	Clear() error
	// GetDailyCosts returns daily cost totals since the given time.
	GetDailyCosts(since time.Time) ([]*DailyCost, error)
	// GetAgentDailyCosts returns per-agent daily cost totals since the given time.
	GetAgentDailyCosts(since time.Time) ([]*AgentDailyCost, error)
	// GetSummarySince returns the cost summary since the given time.
	GetSummarySince(since time.Time) (*Summary, error)
	// GetAgentSummarySince returns per-agent cost summaries since the given time.
	GetAgentSummarySince(since time.Time) ([]*Summary, error)
	// ProjectCost projects future costs based on historical data.
	ProjectCost(lookbackDays int, projectDuration time.Duration) (*Projection, error)
	// Close releases database resources.
	Close() error
}
