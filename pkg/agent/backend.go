package agent

// AgentBackend is the storage interface for agent state persistence.
// SQLiteStore is the default implementation; a memory backend can be used in tests.
type AgentBackend interface {
	// Save persists a single agent.
	Save(a *Agent) error
	// Load reads a single agent by name. Returns nil, nil if not found.
	Load(name string) (*Agent, error)
	// LoadRoot reads the root agent. Returns nil, nil if not found.
	LoadRoot() (*Agent, error)
	// Delete removes a single agent by name.
	Delete(name string) error
	// LoadAll reads every agent into a map keyed by name.
	LoadAll() (map[string]*Agent, error)
	// SaveAll persists every agent in the map inside a single transaction.
	SaveAll(agents map[string]*Agent) error
	// UpdateState updates only the state column for a given agent.
	UpdateState(name string, state State) error
	// UpdateField updates a single text column for a given agent.
	UpdateField(name, field, value string) error
	// SaveStats inserts a single Docker stats sample.
	SaveStats(rec *AgentStatsRecord) error
	// QueryStats returns up to limit recent stats rows for an agent, newest first.
	QueryStats(agentName string, limit int) ([]*AgentStatsRecord, error)
	// Close releases database resources.
	Close() error
}
