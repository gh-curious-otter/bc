// Package stats provides workspace metrics and statistics tracking.
package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/queue"
)

// WorkItemMetrics tracks work item statistics.
type WorkItemMetrics struct {
	// By status
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Assigned int `json:"assigned"`
	Working  int `json:"working"`
	Done     int `json:"done"`
	Failed   int `json:"failed"`

	// By type (parsed from title prefixes like [epic], [task], [bug])
	Epics int `json:"epics"`
	Tasks int `json:"tasks"`
	Bugs  int `json:"bugs"`
	Other int `json:"other"`

	// Derived metrics
	CompletionRate   float64       `json:"completion_rate"`   // Done / Total
	FailureRate      float64       `json:"failure_rate"`      // Failed / Total
	AvgTimeToComplete time.Duration `json:"avg_time_to_complete"`
}

// AgentMetrics tracks agent statistics.
type AgentMetrics struct {
	// Counts
	TotalAgents   int `json:"total_agents"`
	ActiveAgents  int `json:"active_agents"`
	Coordinators  int `json:"coordinators"`
	Workers       int `json:"workers"`

	// By state
	Idle     int `json:"idle"`
	Working  int `json:"working"`
	Done     int `json:"done"`
	Stuck    int `json:"stuck"`
	Error    int `json:"error"`
	Stopped  int `json:"stopped"`

	// Per-agent stats
	AgentStats []AgentStat `json:"agent_stats"`
}

// AgentStat holds stats for a single agent.
type AgentStat struct {
	Name           string        `json:"name"`
	Role           string        `json:"role"`
	State          string        `json:"state"`
	TasksCompleted int           `json:"tasks_completed"`
	TasksFailed    int           `json:"tasks_failed"`
	Uptime         time.Duration `json:"uptime"`
}

// Stats holds all workspace statistics.
type Stats struct {
	// Metadata
	WorkspacePath string    `json:"workspace_path"`
	CollectedAt   time.Time `json:"collected_at"`

	// Metrics
	WorkItems WorkItemMetrics `json:"work_items"`
	Agents    AgentMetrics    `json:"agents"`

	// Historical tracking
	TotalTasksEverCompleted int `json:"total_tasks_ever_completed"`
	TotalTasksEverFailed    int `json:"total_tasks_ever_failed"`

	// Internal
	path string
}

// New creates a new Stats instance for the given workspace.
func New(stateDir string) *Stats {
	return &Stats{
		path:        filepath.Join(stateDir, "stats.json"),
		CollectedAt: time.Now(),
	}
}

// Load reads stats from disk and refreshes with live data.
func Load(stateDir string) (*Stats, error) {
	s := New(stateDir)

	// Try to load existing stats for historical data
	data, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(data, s) // Ignore error, use defaults
	}

	// Refresh with live data
	if err := s.refresh(stateDir); err != nil {
		return nil, err
	}

	return s, nil
}

// refresh updates stats from current workspace state.
func (s *Stats) refresh(stateDir string) error {
	s.CollectedAt = time.Now()
	s.WorkspacePath = filepath.Dir(stateDir)

	// Load work queue
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	if err := q.Load(); err != nil {
		// Queue might not exist yet
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load queue: %w", err)
		}
	}
	s.collectWorkItemMetrics(q)

	// Load agents
	mgr := agent.NewWorkspaceManager(
		filepath.Join(stateDir, "agents"),
		filepath.Dir(stateDir),
	)
	if err := mgr.LoadState(); err != nil {
		// Agents might not exist yet
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load agents: %w", err)
		}
	}
	mgr.RefreshState()
	s.collectAgentMetrics(mgr, q)

	return nil
}

// collectWorkItemMetrics populates work item stats from the queue.
func (s *Stats) collectWorkItemMetrics(q *queue.Queue) {
	items := q.ListAll()
	m := &s.WorkItems

	m.Total = len(items)
	m.Pending = 0
	m.Assigned = 0
	m.Working = 0
	m.Done = 0
	m.Failed = 0
	m.Epics = 0
	m.Tasks = 0
	m.Bugs = 0
	m.Other = 0

	var totalCompletionTime time.Duration
	completedCount := 0

	for _, item := range items {
		// Count by status
		switch item.Status {
		case queue.StatusPending:
			m.Pending++
		case queue.StatusAssigned:
			m.Assigned++
		case queue.StatusWorking:
			m.Working++
		case queue.StatusDone:
			m.Done++
			// Track completion time
			if !item.CreatedAt.IsZero() && !item.UpdatedAt.IsZero() {
				totalCompletionTime += item.UpdatedAt.Sub(item.CreatedAt)
				completedCount++
			}
		case queue.StatusFailed:
			m.Failed++
		}

		// Count by type (parse from title)
		titleLower := strings.ToLower(item.Title)
		switch {
		case strings.HasPrefix(titleLower, "[epic]") || strings.Contains(titleLower, "epic:"):
			m.Epics++
		case strings.HasPrefix(titleLower, "[bug]") || strings.Contains(titleLower, "bug:") || strings.HasPrefix(titleLower, "fix"):
			m.Bugs++
		case strings.HasPrefix(titleLower, "[task]") || strings.Contains(titleLower, "task:"):
			m.Tasks++
		default:
			m.Other++
		}
	}

	// Calculate rates
	if m.Total > 0 {
		m.CompletionRate = float64(m.Done) / float64(m.Total)
		m.FailureRate = float64(m.Failed) / float64(m.Total)
	}

	// Calculate average completion time
	if completedCount > 0 {
		m.AvgTimeToComplete = totalCompletionTime / time.Duration(completedCount)
	}

	// Update historical totals
	if m.Done > s.TotalTasksEverCompleted {
		s.TotalTasksEverCompleted = m.Done
	}
	if m.Failed > s.TotalTasksEverFailed {
		s.TotalTasksEverFailed = m.Failed
	}
}

// collectAgentMetrics populates agent stats from the manager.
func (s *Stats) collectAgentMetrics(mgr *agent.Manager, q *queue.Queue) {
	agents := mgr.ListAgents()
	m := &s.Agents

	m.TotalAgents = len(agents)
	m.ActiveAgents = 0
	m.Coordinators = 0
	m.Workers = 0
	m.Idle = 0
	m.Working = 0
	m.Done = 0
	m.Stuck = 0
	m.Error = 0
	m.Stopped = 0
	m.AgentStats = nil

	for _, a := range agents {
		// Count by role
		switch a.Role {
		case agent.RoleCoordinator:
			m.Coordinators++
		case agent.RoleWorker:
			m.Workers++
		}

		// Count by state
		switch a.State {
		case agent.StateIdle:
			m.Idle++
			m.ActiveAgents++
		case agent.StateWorking:
			m.Working++
			m.ActiveAgents++
		case agent.StateDone:
			m.Done++
			m.ActiveAgents++
		case agent.StateStuck:
			m.Stuck++
			m.ActiveAgents++
		case agent.StateError:
			m.Error++
		case agent.StateStopped:
			m.Stopped++
		}

		// Per-agent stats
		stat := AgentStat{
			Name:  a.Name,
			Role:  string(a.Role),
			State: string(a.State),
		}

		// Calculate uptime
		if a.State != agent.StateStopped && !a.StartedAt.IsZero() {
			stat.Uptime = time.Since(a.StartedAt)
		}

		// Count tasks by agent
		agentItems := q.ListByAgent(a.Name)
		for _, item := range agentItems {
			switch item.Status {
			case queue.StatusDone:
				stat.TasksCompleted++
			case queue.StatusFailed:
				stat.TasksFailed++
			}
		}

		m.AgentStats = append(m.AgentStats, stat)
	}
}

// Save persists stats to disk.
func (s *Stats) Save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Summary returns a formatted string for display.
func (s *Stats) Summary() string {
	var b strings.Builder

	b.WriteString("=== Workspace Stats ===\n")
	b.WriteString(fmt.Sprintf("Collected: %s\n\n", s.CollectedAt.Format(time.RFC3339)))

	// Work Items
	b.WriteString("--- Work Items ---\n")
	b.WriteString(fmt.Sprintf("Total:    %d\n", s.WorkItems.Total))
	b.WriteString(fmt.Sprintf("Pending:  %d\n", s.WorkItems.Pending))
	b.WriteString(fmt.Sprintf("Assigned: %d\n", s.WorkItems.Assigned))
	b.WriteString(fmt.Sprintf("Working:  %d\n", s.WorkItems.Working))
	b.WriteString(fmt.Sprintf("Done:     %d\n", s.WorkItems.Done))
	b.WriteString(fmt.Sprintf("Failed:   %d\n", s.WorkItems.Failed))
	b.WriteString("\n")

	b.WriteString("By Type:\n")
	b.WriteString(fmt.Sprintf("  Epics: %d  Tasks: %d  Bugs: %d  Other: %d\n",
		s.WorkItems.Epics, s.WorkItems.Tasks, s.WorkItems.Bugs, s.WorkItems.Other))
	b.WriteString("\n")

	b.WriteString("Rates:\n")
	b.WriteString(fmt.Sprintf("  Completion: %.1f%%\n", s.WorkItems.CompletionRate*100))
	b.WriteString(fmt.Sprintf("  Failure:    %.1f%%\n", s.WorkItems.FailureRate*100))
	if s.WorkItems.AvgTimeToComplete > 0 {
		b.WriteString(fmt.Sprintf("  Avg Time:   %s\n", formatDuration(s.WorkItems.AvgTimeToComplete)))
	}
	b.WriteString("\n")

	// Agents
	b.WriteString("--- Agents ---\n")
	b.WriteString(fmt.Sprintf("Total:  %d (%d active)\n", s.Agents.TotalAgents, s.Agents.ActiveAgents))
	b.WriteString(fmt.Sprintf("Roles:  %d coordinators, %d workers\n",
		s.Agents.Coordinators, s.Agents.Workers))
	b.WriteString(fmt.Sprintf("States: %d idle, %d working, %d stuck, %d stopped\n",
		s.Agents.Idle, s.Agents.Working, s.Agents.Stuck, s.Agents.Stopped))
	b.WriteString("\n")

	if len(s.Agents.AgentStats) > 0 {
		b.WriteString("Per Agent:\n")
		for _, a := range s.Agents.AgentStats {
			uptimeStr := "-"
			if a.Uptime > 0 {
				uptimeStr = formatDuration(a.Uptime)
			}
			b.WriteString(fmt.Sprintf("  %-15s %-12s %-10s completed:%d failed:%d uptime:%s\n",
				a.Name, a.Role, a.State, a.TasksCompleted, a.TasksFailed, uptimeStr))
		}
	}

	return b.String()
}

// Utilization returns current agent utilization (working/active).
func (s *Stats) Utilization() float64 {
	if s.Agents.ActiveAgents == 0 {
		return 0
	}
	return float64(s.Agents.Working) / float64(s.Agents.ActiveAgents)
}

// formatDuration formats a duration for human display.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	sec := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, sec)
	}
	return fmt.Sprintf("%ds", sec)
}
