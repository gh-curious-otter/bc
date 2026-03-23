package cost

import (
	"context"
	"database/sql"
	"time"
)

// CostBackend is the storage interface implemented by both SQLiteStore (the
// existing Store) and PostgresStore. Higher-level wrappers delegate all
// persistence operations to this interface.
type CostBackend interface {
	// Lifecycle
	Close() error
	DB() *sql.DB

	// Record operations
	Record(ctx context.Context, agentID, teamID, model string, inputTokens, outputTokens int64, costUSD float64) (*Record, error)
	GetByID(ctx context.Context, id int64) (*Record, error)
	GetByAgent(ctx context.Context, agentID string, limit int) ([]*Record, error)
	GetByAgentWithOffset(ctx context.Context, agentID string, limit, offset int) ([]*Record, error)
	GetByTeam(ctx context.Context, teamID string, limit int) ([]*Record, error)
	GetAll(ctx context.Context, limit int) ([]*Record, error)
	GetAllWithOffset(ctx context.Context, limit, offset int) ([]*Record, error)
	Clear(ctx context.Context) error

	// Summary operations
	SummaryByAgent(ctx context.Context) ([]*Summary, error)
	SummaryByTeam(ctx context.Context) ([]*Summary, error)
	SummaryByModel(ctx context.Context) ([]*Summary, error)
	WorkspaceSummary(ctx context.Context) (*Summary, error)
	AgentSummary(ctx context.Context, agentID string) (*Summary, error)
	TeamSummary(ctx context.Context, teamID string) (*Summary, error)
	GetSummarySince(ctx context.Context, since time.Time) (*Summary, error)
	GetAgentSummarySince(ctx context.Context, since time.Time) ([]*Summary, error)

	// Budget operations
	SetBudget(ctx context.Context, scope string, period BudgetPeriod, limitUSD, alertAt float64, hardStop bool) (*Budget, error)
	GetBudget(ctx context.Context, scope string) (*Budget, error)
	GetAllBudgets(ctx context.Context) ([]*Budget, error)
	DeleteBudget(ctx context.Context, scope string) error
	CheckBudget(ctx context.Context, scope string) (*BudgetStatus, error)

	// Daily cost operations
	GetDailyCosts(ctx context.Context, since time.Time) ([]*DailyCost, error)
	GetAgentDailyCosts(ctx context.Context, since time.Time) ([]*AgentDailyCost, error)

	// Projection
	ProjectCost(ctx context.Context, lookbackDays int, projectDuration time.Duration) (*Projection, error)
}
